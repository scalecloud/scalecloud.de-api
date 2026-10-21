package apimanager

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/scalecloud/scalecloud.de-api/appconfig"
	"github.com/scalecloud/scalecloud.de-api/authentikmanager"
	"github.com/scalecloud/scalecloud.de-api/authmanager"
	"github.com/scalecloud/scalecloud.de-api/emailmanager"
	"github.com/scalecloud/scalecloud.de-api/healthmanager"
	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
	"github.com/scalecloud/scalecloud.de-api/newslettermanager"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"github.com/scalecloud/scalecloud.de-api/stripemanager/secret"
	"go.uber.org/zap"
)

type Api struct {
	config         appconfig.Config
	router         *gin.Engine
	paymentHandler *stripemanager.PaymentHandler
	webhookHandler *WebhookHandler
	authConnection *authmanager.OIDCConnection
	validate       *validator.Validate
	log            *zap.Logger
	healthMonitor  *healthmanager.Monitor
	cancelHealth   context.CancelFunc
}

type WebhookHandler struct {
	StripeConnection *stripemanager.StripeConnection
	Log              *zap.Logger
}

func InitAPI(log *zap.Logger, config appconfig.Config) (*Api, error) {
	log.Info("Init api")

	authConnection, err := authmanager.InitOIDCConnection(context.Background())
	if err != nil {
		return &Api{}, err
	}
	authentikConnection, err := authentikmanager.NewAdminAPIClientFromEnv()
	if err != nil {
		return &Api{}, err
	}
	invitationClaimURL := config.FrontendURL + "/claim-invite"

	err = mongodbmanager.CheckMongoDBConfig(log)
	if err != nil {
		return &Api{}, err
	}
	err = secret.CheckStripeConfig(log)
	if err != nil {
		return &Api{}, err
	}

	router := gin.Default()

	router.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))

	emailConnection, err := emailmanager.InitEMailConnection(log)
	if err != nil {
		return &Api{}, err
	}

	mongoDBConnection, err := mongodbmanager.InitMongoDBConnection(context.Background(), log)
	if err != nil {
		return &Api{}, err
	}

	newsletterConnection, err := newslettermanager.InitNewsletterConnection(context.Background(), log, mongoDBConnection, emailConnection)
	if err != nil {
		return &Api{}, err
	}

	stripeConnection, err := stripemanager.InitStripeConnection(context.Background(), log)
	if err != nil {
		return &Api{}, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	healthMonitor := healthmanager.NewMonitor([]healthmanager.Check{
		{Name: "mongodb", Required: true, Run: mongoDBConnection.CheckHealth},
		{Name: "stripe", Required: true, Run: stripeConnection.CheckHealth},
		{Name: "authentik", Required: true, Run: func(ctx context.Context) error {
			if err := authConnection.CheckHealth(ctx); err != nil {
				return err
			}
			return authentikConnection.CheckHealth(ctx)
		}},
		{Name: "smtp", Required: false, Run: emailConnection.CheckHealth},
	}, 60*time.Second, 5*time.Second, log.Named("health"))
	healthContext, cancelHealth := context.WithCancel(context.Background())
	healthMonitor.CheckNow(healthContext)
	healthMonitor.Start(healthContext)

	api := &Api{
		config:         config,
		router:         router,
		authConnection: authConnection,
		paymentHandler: &stripemanager.PaymentHandler{
			AuthentikConnection:  authentikConnection,
			InvitationClaimURL:   invitationClaimURL,
			MongoDBConnection:    mongoDBConnection,
			StripeConnection:     stripeConnection,
			EMailConnection:      emailConnection,
			NewsletterConnection: newsletterConnection,
			Log:                  log.Named("paymenthandler"),
		},
		webhookHandler: &WebhookHandler{
			StripeConnection: stripeConnection,
			Log:              log.Named("webhookhandler"),
		},
		validate:      validate,
		log:           log.Named("apimanager"),
		healthMonitor: healthMonitor,
		cancelHealth:  cancelHealth,
	}
	return api, nil
}

func (api *Api) CloseMongoClient() {
	if api.cancelHealth != nil {
		api.cancelHealth()
	}
	err := api.paymentHandler.MongoDBConnection.Client.Disconnect(context.Background())
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
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{api.config.FrontendURL}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "Baggage", "sentry-trace"}
	corsConfig.AllowCredentials = true
	corsConfig.ExposeHeaders = []string{"Content-Length"}
	corsConfig.MaxAge = 12 * time.Hour
	api.router.Use(cors.New(corsConfig))
}

func (api *Api) initRoutes() {
	api.log.Info("Setting up cache...")
	store := persistence.NewInMemoryStore(time.Minute * 5)

	api.log.Info("Setting up routes...")
	api.router.GET("/health/live", api.healthLive)
	api.router.GET("/health/ready", api.healthReady)

	api.router.POST("/invitation/preview", api.previewSeatInvitation)
	api.router.POST("/invitation/claim", api.authRequired, api.claimSeatInvitation)

	webhook := api.router.Group("/webhook/")
	webhook.Use(api.StripeRequired)
	{
		webhook.POST("/stripe", api.handleStripeWebhook)
	}

	product := api.router.Group("/product/tiers")
	{
		product.GET("/nextcloud", cache.CachePage(store, time.Hour*24, api.getProductTiersNextcloud))
		product.GET("/synology", cache.CachePage(store, time.Hour*24, api.getProductTiersSynology))
	}

	dashboard := api.router.Group("/dashboard")
	dashboard.Use(api.authRequired)
	{
		dashboard.GET("/subscriptions", api.getSubscriptionsOverview)
		dashboard.GET("/subscription/:id", api.getSubscriptionByID)
		dashboard.GET("/subscription/:id/cancel-state", api.getCancelState)
		dashboard.POST("/subscription/permission", api.GetMyPermission)
		dashboard.POST("/subscription/list-seats", api.getSubscriptionListSeats)
		dashboard.POST("/subscription/seat-detail", api.getSubscriptionSeatDetail)
		dashboard.POST("/subscription/update-seat", api.getSubscriptionUpdateSeat)
		dashboard.POST("/subscription/add-seat", api.getSubscriptionAddSeat)
		dashboard.POST("/subscription/delete-seat", api.getSubscriptionRemoveSeat)
		dashboard.POST("/subscription/invoices", api.getSubscriptionInvoices)
		dashboard.POST("/subscription/billing-address", api.getBillingAddress)
		dashboard.POST("/subscription/update-billing-address", api.updateBillingAddress)
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
	newsletters := api.router.Group("/newsletter")
	{
		newsletters.POST("/subscribe", api.newsletterSubscribe)
		newsletters.POST("/confirm", api.newsletterConfirm)
		newsletters.POST("/unsubscribe", api.newsletterUnsubscribe)
	}
}

func (api *Api) healthLive(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (api *Api) healthReady(c *gin.Context) {
	snapshot := api.healthMonitor.Snapshot()
	statusCode := http.StatusOK
	if snapshot.Status != "ready" {
		statusCode = http.StatusServiceUnavailable
	}
	c.JSON(statusCode, snapshot)
}

func (api *Api) initCertificate() {
	if api.config.Environment.IsProductionLike() {
		api.log.Info("Setting up certificate...")
		err := autotls.Run(api.router, api.config.APIDomain)
		if err != nil {
			api.log.Error("Could not setup certificate", zap.Error(err))
			panic(err)
		}
		api.log.Info("Certificate setup done.")
	}
}

func (api *Api) initTrustedProxies() {
	if api.config.Environment.IsProductionLike() {
		err := api.router.SetTrustedProxies([]string{api.config.TrustedProxyIP})
		if err != nil {
			api.log.Fatal("Could not set trusted proxy", zap.Error(err))
		}
		api.log.Info("Trusted proxy set", zap.String("proxyIP", api.config.TrustedProxyIP))
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
	authorization := strings.Fields(c.GetHeader("Authorization"))
	if len(authorization) != 2 || !strings.EqualFold(authorization[0], "Bearer") {
		api.log.Warn("Unauthorized: invalid authorization header")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	tokenDetails, err := api.authConnection.VerifyToken(c, authorization[1])
	if err != nil {
		api.log.Warn("Unauthorized: token verification failed")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.Set("tokenDetails", tokenDetails)
	c.Next()
}

func (api *Api) handleTokenDetails(c *gin.Context) (authmanager.TokenDetails, error) {
	tokenDetails, ok := c.Get("tokenDetails")
	if !ok {
		err := errors.New("authenticated token details are missing")
		api.log.Error("Error getting token details", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return authmanager.TokenDetails{}, err
	}
	details, ok := tokenDetails.(authmanager.TokenDetails)
	if !ok {
		err := errors.New("authenticated token details have an invalid type")
		api.log.Error("Error getting token details", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return authmanager.TokenDetails{}, err
	}
	return details, nil
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
		if err.Error() == http.StatusText(http.StatusForbidden) {
			api.log.Warn("Access denied", zap.Error(err))
			c.SecureJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return false
		} else {
			api.log.Error("Validate reply", zap.Error(err))
			c.SecureJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return false
		}
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
