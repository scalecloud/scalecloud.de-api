package apimanager

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"github.com/scalecloud/scalecloud.de-api/stripemanager/secret"
	"go.uber.org/zap"
)

type Api struct {
	router         *gin.Engine
	paymentHandler *stripemanager.PaymentHandler
	webhookHandler *WebhookHandler
	log            *zap.Logger
}

type WebhookHandler struct {
	StripeConnection *stripemanager.StripeConnection
	Log              *zap.Logger
}

func InitAPI(log *zap.Logger) (*Api, error) {
	log.Info("Init api")

	err := mongomanager.InitMongo(log)
	if err != nil {
		return &Api{}, err
	}
	err = secret.InitStripe(log)
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

	stripeConnection, err := stripemanager.InitStripeConnection(context.Background(), log)
	if err != nil {
		return &Api{}, err
	}

	api := &Api{
		router: router,
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
		log: log.Named("apimanager"),
	}
	return api, nil
}

func (a *Api) RunAPI() {
	a.initHeaders()
	a.initRoutes()
	a.initCertificate()
	a.initTrustedPlatform()
	a.startListening()
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

func (api *Api) startListening() {
	api.log.Info("Starting listening for requests")
	err := api.router.Run(":15000")
	if err != nil {
		api.log.Error("Could not start listening for requests", zap.Error(err))
	}
}

func (api *Api) initRoutes() {
	api.log.Info("Setting up routes...")

	webhook := api.router.Group("/webhook/")
	webhook.Use(api.StripeRequired)
	{
		webhook.POST("/stripe", api.webhookHandler.handleStripeWebhook)
	}

	dashboard := api.router.Group("/dashboard")
	dashboard.Use(api.authRequired)
	{
		dashboard.GET("/subscriptions", api.getSubscriptionsOverview)
		dashboard.GET("/subscription/:id", api.getSubscriptionByID)
		dashboard.POST("/get-subscription-payment-method", api.getSubscriptionPaymentMethod)
		dashboard.POST("/get-change-payment-setup-intent", api.getChangePaymentSetupIntent)
		dashboard.POST("/resume-subscription", api.resumeSubscription)
		dashboard.POST("/cancel-subscription", api.cancelSubscription)
		dashboard.GET("/billing-portal", api.handleBillingPortal)
	}
	checkoutPortal := api.router.Group("/checkout-portal")
	checkoutPortal.Use(api.authRequired)
	{
		checkoutPortal.POST("/create-checkout-session", api.createCheckoutSession)

	}
	checkoutIntegration := api.router.Group("/checkout-integration")
	checkoutIntegration.Use(api.authRequired)
	{
		checkoutIntegration.POST("/create-checkout-subscription", api.createCheckoutSubscription)
		checkoutIntegration.POST("/update-checkout-subscription", api.updateCheckoutSubscription)
		checkoutIntegration.POST("/get-checkout-product", api.getCheckoutProduct)
	}
	checkoutSetupIntent := api.router.Group("/checkout-setup-intent")
	checkoutSetupIntent.Use(api.authRequired)
	{
		checkoutSetupIntent.POST("/create-setup-intent", api.createCheckoutSetupIntent)
	}
}

func (api *Api) initCertificate() {
	api.log.Warn("init certificate not implemented yet.")
	/* error := autotls.Run(router, "api.scalecloud.de")
	if error != nil {
		logger.Error("Could not setup certificate", zap.Error(error))
	} */
}

func (api *Api) initTrustedPlatform() {
	api.log.Info("init trusted platform not implemented yet.")
	/* router.TrustedPlatform = gin.PlatformGoogleAppEngine */
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
	}
	api.log.Debug("Authenticated", zap.String("token:", token))
	c.Next()
}
