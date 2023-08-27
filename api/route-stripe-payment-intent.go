package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

func CreateCheckoutSubscription(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}

	var checkoutIntegrationRequest stripemanager.CheckoutPaymentIntentRequest
	if err := c.BindJSON(&checkoutIntegrationRequest); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if checkoutIntegrationRequest.ProductID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "productID not found"})
		return
	}
	logger.Debug("productID", zap.Any("productID", checkoutIntegrationRequest.ProductID))
	if checkoutIntegrationRequest.Quantity == 0 {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	logger.Debug("quantity", zap.Any("quantity", checkoutIntegrationRequest.Quantity))
	secret, error := stripemanager.CreateCheckoutSubscription(c, token, checkoutIntegrationRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	logger.Info("CreateSubscription", zap.Any("secret", secret))
	c.IndentedJSON(http.StatusOK, secret)
}

func updateCheckoutSubscription(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}

	var checkoutIntegrationUpdateRequest stripemanager.CheckoutPaymentIntentUpdateRequest
	if err := c.BindJSON(&checkoutIntegrationUpdateRequest); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if checkoutIntegrationUpdateRequest.SubscriptionID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "SubscriptionID not found"})
		return
	}
	logger.Debug("subscriptionID", zap.Any("subscriptionID", checkoutIntegrationUpdateRequest.SubscriptionID))
	if checkoutIntegrationUpdateRequest.Quantity == 0 {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	logger.Debug("quantity", zap.Any("quantity", checkoutIntegrationUpdateRequest.Quantity))
	secret, error := stripemanager.UpdateCheckoutSubscription(c, token, checkoutIntegrationUpdateRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	logger.Info("UpdateCheckoutSubscription", zap.Any("secret", secret))
	c.IndentedJSON(http.StatusOK, secret)
}

func getCheckoutProduct(c *gin.Context) {
	logger.Info("getCheckoutProduct")
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}

	var checkoutProductRequest stripemanager.CheckoutProductRequest
	if err := c.BindJSON(&checkoutProductRequest); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if checkoutProductRequest.SubscriptionID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "SubscriptionID not found"})
		return
	}
	logger.Debug("subscriptionID", zap.Any("subscriptionID", checkoutProductRequest.SubscriptionID))
	checkoutProductReply, error := stripemanager.GetCheckoutProduct(c, token, checkoutProductRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	logger.Info("GetCheckoutProduct", zap.Any("checkoutProductReply", checkoutProductReply))
	c.IndentedJSON(http.StatusOK, checkoutProductReply)
}
