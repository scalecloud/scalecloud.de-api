package apimanager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

func (api *Api) createCheckoutSubscription(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c, getBearerToken(c))
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
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
	api.log.Debug("productID", zap.Any("productID", checkoutIntegrationRequest.ProductID))
	if checkoutIntegrationRequest.Quantity == 0 {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	api.log.Debug("quantity", zap.Any("quantity", checkoutIntegrationRequest.Quantity))
	secret, error := api.paymentHandler.StripeConnection.CreateCheckoutSubscription(c, tokenDetails, checkoutIntegrationRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("CreateSubscription", zap.Any("secret", secret))
	c.IndentedJSON(http.StatusOK, secret)
}

func (api *Api) updateCheckoutSubscription(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c, getBearerToken(c))
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
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
	api.log.Debug("subscriptionID", zap.Any("subscriptionID", checkoutIntegrationUpdateRequest.SubscriptionID))
	if checkoutIntegrationUpdateRequest.Quantity == 0 {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	api.log.Debug("quantity", zap.Any("quantity", checkoutIntegrationUpdateRequest.Quantity))
	secret, error := api.paymentHandler.StripeConnection.UpdateCheckoutSubscription(c, tokenDetails, checkoutIntegrationUpdateRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("UpdateCheckoutSubscription", zap.Any("secret", secret))
	c.IndentedJSON(http.StatusOK, secret)
}

func (api *Api) getCheckoutProduct(c *gin.Context) {
	api.log.Info("getCheckoutProduct")
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c, getBearerToken(c))
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
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
	api.log.Debug("subscriptionID", zap.Any("subscriptionID", checkoutProductRequest.SubscriptionID))
	checkoutProductReply, error := api.paymentHandler.StripeConnection.GetCheckoutProduct(c, tokenDetails, checkoutProductRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("GetCheckoutProduct", zap.Any("checkoutProductReply", checkoutProductReply))
	c.IndentedJSON(http.StatusOK, checkoutProductReply)
}
