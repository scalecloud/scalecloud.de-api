package apimanager

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"github.com/scalecloud/scalecloud.de-api/stripemanager/secret"
	"go.uber.org/zap"
)

type Api struct {
	production     bool
	proxyIP        string
	router         *gin.Engine
	paymentHandler *stripemanager.PaymentHandler
	webhookHandler *WebhookHandler
	validate       *validator.Validate
	log            *zap.Logger
}

type WebhookHandler struct {
	StripeConnection *stripemanager.StripeConnection
	Log              *zap.Logger
}

func InitAPI(log *zap.Logger, production bool, proxyIP string) (*Api, error) {
	log.Info("Init api")

	err := mongomanager.CheckMongoConnectionFiles(log)
	if err != nil {
		return &Api{}, err
	}
	err = secret.CheckStripeKeyFiles(log)
	if err != nil {
		return &Api{}, err
	}

	router := gin.Default()

	firebaseConnection, err := firebasemanager.InitFirebaseConnection(context.Background(), log)
	if err != nil {
		return &Api{}, err
	}

	mongoConnection, err := mongomanager.InitMongoConnection(context.Background(), log)
	if err != nil {
		return &Api{}, err
	}

	err = mongoConnection.CheckMongoConnectability(context.Background())
	if err != nil {
		return &Api{}, err
	}

	err = mongoConnection.CheckDatabaseAndCollectionExists(context.Background())
	if err != nil {
		return &Api{}, err
	}

	err = mongoConnection.EnsureIndexes()
	if err != nil {
		return &Api{}, err
	}

	stripeConnection, err := stripemanager.InitStripeConnection(context.Background(), log)
	if err != nil {
		return &Api{}, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())

	api := &Api{
		production: production,
		proxyIP:    proxyIP,
		router:     router,
		paymentHandler: &stripemanager.PaymentHandler{
			FirebaseConnection: firebaseConnection,
			MongoConnection:    mongoConnection,
			StripeConnection:   stripeConnection,
			Log:                log.Named("paymenthandler"),
		},
		webhookHandler: &WebhookHandler{
			StripeConnection: stripeConnection,
			Log:              log.Named("webhookhandler"),
		},
		validate: validate,
		log:      log.Named("apimanager"),
	}
	return api, nil
}

func (api *Api) CloseMongoClient() {
	err := api.paymentHandler.MongoConnection.Client.Disconnect(context.Background())
	if err != nil {
		api.log.Fatal("Error closing MongoDB", zap.Error(err))
	}
}

func (api *Api) RunAPI() {
	api.initHeaders()
	api.initRoutes()
	api.initCertificate()
	api.initTrustedProxies()
	api.startListening()
}

func (api *Api) initHeaders() {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:4200"}
	config.AllowMethods = []string{"GET"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	config.AllowCredentials = true
	config.ExposeHeaders = []string{"Content-Length"}
	config.MaxAge = 12 * time.Hour
	api.router.Use(cors.New(config))
}

func (api *Api) initRoutes() {
	api.log.Info("Setting up routes...")

	webhook := api.router.Group("/webhook/")
	webhook.Use(api.StripeRequired)
	{
		webhook.POST("/stripe", api.handleStripeWebhook)
	}

	dashboard := api.router.Group("/dashboard")
	dashboard.Use(api.authRequired)
	{
		dashboard.GET("/subscriptions", api.getSubscriptionsOverview)
		dashboard.GET("/subscription/:id", api.getSubscriptionByID)
		dashboard.POST("/subscription/list-seats", api.getSubscriptionListSeats)
		dashboard.POST("/subscription/add-seats", api.getSubscriptionAddSeat)
		dashboard.POST("/subscription/remove-seats", api.getSubscriptionRemoveSeat)
		dashboard.POST("/get-payment-method-overview", api.getPaymentMethodOverview)
		dashboard.POST("/get-change-payment-setup-intent", api.getChangePaymentSetupIntent)
		dashboard.POST("/resume-subscription", api.resumeSubscription)
		dashboard.POST("/cancel-subscription", api.cancelSubscription)
		dashboard.GET("/billing-portal", api.handleBillingPortal)
	}
	checkoutIntegration := api.router.Group("/checkout-integration")
	checkoutIntegration.Use(api.authRequired)
	{
		checkoutIntegration.POST("/create-checkout-subscription", api.createCheckoutSubscription)
		checkoutIntegration.POST("/get-checkout-product", api.getCheckoutProduct)
	}
	checkoutSetupIntent := api.router.Group("/checkout-setup-intent")
	checkoutSetupIntent.Use(api.authRequired)
	{
		checkoutSetupIntent.POST("/create-setup-intent", api.createCheckoutSetupIntent)
	}
}

func (api *Api) initCertificate() {
	if api.production {
		api.log.Info("Setting up certificate...")
		err := autotls.Run(api.router, "api.scalecloud.de")
		if err != nil {
			api.log.Error("Could not setup certificate", zap.Error(err))
			panic(err)
		}
		api.log.Info("Certificate setup done.")
	}
}

func (api *Api) initTrustedProxies() {
	if api.production {
		if api.proxyIP == "" {
			api.log.Fatal("Proxy IP is empty")
		} else {
			err := api.router.SetTrustedProxies([]string{api.proxyIP})
			if err != nil {
				api.log.Fatal("Could not set trusted proxy", zap.Error(err))
			} else {
				api.log.Info("Trusted proxy set", zap.String("proxyIP", api.proxyIP))
			}
		}
	} else {
		err := api.router.SetTrustedProxies([]string{"127.0.0.1"})
		if err != nil {
			api.log.Fatal("Could not set trusted proxy", zap.Error(err))
		} else {
			api.log.Info("Trusted proxy set", zap.String("proxyIP", "127.0.0.1"))
		}
	}
}

func (api *Api) startListening() {
	api.log.Info("Starting listening for requests")
	err := api.router.Run(":15000")
	if err != nil {
		api.log.Error("Could not start listening for requests", zap.Error(err))
	}
}

func (api *Api) authRequired(c *gin.Context) {
	token, err := firebasemanager.GetBearerToken(c)
	if err != nil {
		api.log.Warn("Unauthorized", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	err = api.paymentHandler.FirebaseConnection.VerifyIDToken(c, token)
	if err != nil {
		api.log.Warn("Unauthorized", zap.String("token:", token))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	api.log.Debug("Authenticated", zap.String("token:", token))
	c.Next()
}

func (api *Api) handleTokenDetails(c *gin.Context) (firebasemanager.TokenDetails, error) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return firebasemanager.TokenDetails{}, err
	}
	return tokenDetails, nil
}

func (api *Api) handleBind(c *gin.Context, s interface{}) bool {
	err := c.BindJSON(s)
	if err != nil {
		api.log.Warn("Error binding json", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	api.log.Info("Request", zap.Any("request", s))
	return true
}

func (api *Api) validateReply(c *gin.Context, err error, reply interface{}) bool {
	if err != nil {
		api.log.Error("Validate reply", zap.Error(err))
		c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return false
	}
	return api.validateStruct(c, reply)
}

func (api *Api) validateStruct(c *gin.Context, s interface{}) bool {
	if s == nil {
		api.log.Error("Struct is nil")
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": "Struct is nil"})
		return false
	}
	err := api.validate.Struct(s)
	if err != nil {
		api.log.Error("Error validating struct", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func (api *Api) validateAndWriteReply(c *gin.Context, err error, reply interface{}) {
	if api.validateReply(c, err, reply) {
		api.writeReply(c, reply)
	}
}

func (api *Api) writeReply(c *gin.Context, reply interface{}) {
	api.log.Info("Reply", zap.Any("reply", reply))
	c.IndentedJSON(http.StatusOK, reply)
}
