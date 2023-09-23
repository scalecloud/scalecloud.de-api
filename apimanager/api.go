package apimanager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
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
	api.initTrustedPlatform()
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
	//checkoutPortal.Use(api.authRequired)
	//{
	checkoutPortal.POST("/create-checkout-session", api.createCheckoutSession)

	//}
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
	return api.handleStructFull(c, s)
}

func (api *Api) handleStructFull(c *gin.Context, s interface{}) bool {
	err := isStructFull(s)
	if err != nil {
		api.log.Warn("Error validating struct", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func isStructFull(s interface{}) (err error) {
	if s == nil {
		return errors.New("Input param is nil")
	}
	// first make sure that the input is a struct
	// having any other type, especially a pointer to a struct,
	// might result in panic
	structType := reflect.TypeOf(s)
	if structType.Kind() != reflect.Struct {
		return errors.New("Input param should be a struct")
	}

	// now go one by one through the fields and validate their value
	structVal := reflect.ValueOf(s)
	fieldNum := structVal.NumField()

	for i := 0; i < fieldNum; i++ {
		// Field(i) returns i'th value of the struct
		field := structVal.Field(i)
		fieldName := structType.Field(i).Name

		// CAREFUL! IsZero interprets empty strings and int equal 0 as a zero value.
		// To check only if the pointers have been initialized,
		// you can check the kind of the field:
		// if field.Kind() == reflect.Pointer { // check }

		// IsZero panics if the value is invalid.
		// Most functions and methods never return an invalid Value.
		isSet := field.IsValid() && !field.IsZero()

		if !isSet {
			err = errors.New(fmt.Sprintf("%s in not set.", fieldName))
		}

	}

	return err
}

func (api *Api) writeReply(c *gin.Context, err error, reply interface{}) {
	if err != nil {
		api.log.Error("Error creating checkout session", zap.Error(err))
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if api.handleStructFull(c, reply) {
		api.log.Info("Reply", zap.Any("reply", reply))
		c.IndentedJSON(http.StatusOK, reply)
	}
}
