package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/scalecloud.de-api"
	"github.com/scalecloud/scalecloud.de-api/stripe"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

const messageBearer = "Bearer token not found"

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
		dashboard.GET("/billing-portal", getBillingPortal)
	}
	checkout := router.Group("/checkout")
	dashboard.Use(AuthRequired)
	{
		checkout.POST("/create-checkout-session", createCheckoutSession)
		checkout.POST("/create-checkout-subscription", createCheckoutSubscription)
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

func getSubscriptionsOverview(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}
	subscriptionsOverview, error := scalecloud.GetSubscriptionsOverview(c, token)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	logger.Info("getSubscriptionsOverview", zap.Any("subscriptionsOverview", subscriptionsOverview))
	if subscriptionsOverview != nil {
		c.IndentedJSON(http.StatusOK, subscriptionsOverview)
	} else {
		c.SecureJSON(http.StatusNotFound, gin.H{"message": "subscriptionsOverview not found"})
	}
}

func getSubscriptionByID(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}
	subscriptionID := c.Param("id")
	logger.Debug("getSubscriptionByID", zap.String("subscriptionID", subscriptionID))
	subscriptionDetail, error := scalecloud.GetSubscriptionByID(c, token, subscriptionID)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	logger.Info("Found subscriptionDetail", zap.Any("subscriptionDetail", subscriptionDetail))
	if subscriptionDetail != (stripe.SubscriptionDetail{}) {
		c.IndentedJSON(http.StatusOK, subscriptionDetail)
	} else {
		c.SecureJSON(http.StatusNotFound, gin.H{"message": "subscriptionDetail not found"})
	}
}

func getBillingPortal(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}
	billingPortal, error := scalecloud.GetBillingPortal(c, token)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	logger.Info("getBillingPortal", zap.Any("billingPortal", billingPortal))
	c.IndentedJSON(http.StatusOK, billingPortal)
}

func createCheckoutSession(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}

	var productModel stripe.ProductModel
	if err := c.BindJSON(&productModel); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if productModel.ProductID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "productID not found"})
		return
	}
	logger.Debug("productID", zap.Any("productID", productModel.ProductID))
	if productModel.Quantity == 0 {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	logger.Debug("quantity", zap.Any("quantity", productModel.Quantity))
	checkout, error := scalecloud.CreateCheckoutSession(c, token, productModel)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	logger.Info("CreateCheckoutSession", zap.Any("checkout", checkout))
	c.IndentedJSON(http.StatusOK, checkout)
}

func createCheckoutSubscription(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}

	var productModel stripe.ProductModel
	if err := c.BindJSON(&productModel); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if productModel.ProductID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "productID not found"})
		return
	}
	logger.Debug("productID", zap.Any("productID", productModel.ProductID))
	if productModel.Quantity == 0 {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	logger.Debug("quantity", zap.Any("quantity", productModel.Quantity))
	secret, error := scalecloud.CreateCheckoutSubscription(c, token, productModel)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	logger.Info("CreateSubscription", zap.Any("secret", secret))
	c.IndentedJSON(http.StatusOK, secret)
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
