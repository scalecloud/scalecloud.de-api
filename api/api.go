package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/scalecloud.de-api"
	"go.uber.org/zap"
)

const messageBearer = "Bearer token not found"

var logger, _ = zap.NewProduction()

func InitApi() {
	logger.Info("Init api")
	scalecloud.Init()
	startAPI()
}

func startAPI() {
	router := gin.Default()
	initHeaders(router)
	initRoutes(router)
	initCertificate(router)
	initTrustedPlatform(router)
	startListening(router)
}

func initHeaders(router *gin.Engine) {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:4200"}
	config.AllowMethods = []string{"GET"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	config.AllowCredentials = true
	config.ExposeHeaders = []string{"Content-Length"}
	config.MaxAge = 12 * time.Hour
	router.Use(cors.New(config))
}

func startListening(router *gin.Engine) {
	logger.Info("Starting listening for requests")
	err := router.Run(":15000")
	if err != nil {
		logger.Error("Could not start listening for requests", zap.Error(err))
	}
}

func initRoutes(router *gin.Engine) {
	logger.Info("Setting up routes...")
	// Subscription

	// Account
	dashboard := router.Group("/dashboard")
	dashboard.Use(AuthRequired)
	{
		dashboard.GET("/subscriptions", getSubscriptionsOverview)
		dashboard.GET("/subscription/:id", getSubscriptionByID)
		dashboard.POST("/get-subscription-payment-method", getSubscriptionPaymentMethod)
		dashboard.POST("/change-subscription-payment-method", changeSubscriptionPaymentMethod)
		dashboard.POST("/resume-subscription", resumeSubscription)
		dashboard.POST("/cancel-subscription", cancelSubscription)
		dashboard.GET("/billing-portal", getBillingPortal)
	}
	checkoutPortal := router.Group("/checkout-portal")
	checkoutPortal.Use(AuthRequired)
	{
		checkoutPortal.POST("/create-checkout-session", createCheckoutSession)

	}
	checkoutIntegration := router.Group("/checkout-integration")
	checkoutIntegration.Use(AuthRequired)
	{
		checkoutIntegration.POST("/create-checkout-subscription", CreateCheckoutSubscription)
		checkoutIntegration.POST("/update-checkout-subscription", updateCheckoutSubscription)
		checkoutIntegration.POST("/get-checkout-product", getCheckoutProduct)
	}
	checkoutSetupIntent := router.Group("/checkout-setup-intent")
	checkoutSetupIntent.Use(AuthRequired)
	{
		checkoutSetupIntent.POST("/create-setup-intent", CreateCheckoutSetupIntent)
	}
}

func initCertificate(router *gin.Engine) {
	logger.Warn("init certificate not implemented yet.")
	/* error := autotls.Run(router, "api.scalecloud.de")
	if error != nil {
		logger.Error("Could not setup certificate", zap.Error(error))
	} */
}

func initTrustedPlatform(router *gin.Engine) {
	logger.Info("init trusted platform not implemented yet.")
	/* router.TrustedPlatform = gin.PlatformGoogleAppEngine */
}

// Authentication
func AuthRequired(c *gin.Context) {
	token, hasAuth := getBearerToken(c)
	if hasAuth && token != "" && scalecloud.IsAuthenticated(c, token) {
		logger.Debug("Authenticated", zap.String("token:", token))
		c.Next()
	} else {
		logger.Warn("Unauthorized", zap.String("token:", token))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

func getBearerToken(c *gin.Context) (token string, ok bool) {
	jwtToken := c.Request.Header.Get("Authorization")
	if jwtToken == "" {
		return "", false
	} else {
		return jwtToken, true
	}
}
