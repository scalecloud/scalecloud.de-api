package apimanager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

func (api *Api) createCheckoutSubscription(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	var checkoutIntegrationRequest stripemanager.CheckoutPaymentIntentRequest
	if err := c.BindJSON(&checkoutIntegrationRequest); err != nil {
		api.log.Error("Error binding JSON", zap.Error(err))
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if checkoutIntegrationRequest.ProductID == "" {
		api.log.Error("productID not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "productID not found"})
		return
	}
	api.log.Debug("productID", zap.Any("productID", checkoutIntegrationRequest.ProductID))
	if checkoutIntegrationRequest.Quantity == 0 {
		api.log.Error("quantity not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	api.log.Debug("quantity", zap.Any("quantity", checkoutIntegrationRequest.Quantity))
	secret, err := api.paymentHandler.CreateCheckoutSubscription(c, tokenDetails, checkoutIntegrationRequest)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("CreateSubscription", zap.Any("secret", secret))
	c.IndentedJSON(http.StatusOK, secret)
}

func (api *Api) updateCheckoutSubscription(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	var checkoutIntegrationUpdateRequest stripemanager.CheckoutPaymentIntentUpdateRequest
	if err := c.BindJSON(&checkoutIntegrationUpdateRequest); err != nil {
		api.log.Error("Error binding JSON", zap.Error(err))
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if checkoutIntegrationUpdateRequest.SubscriptionID == "" {
		api.log.Error("SubscriptionID not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "SubscriptionID not found"})
		return
	}
	api.log.Debug("subscriptionID", zap.Any("subscriptionID", checkoutIntegrationUpdateRequest.SubscriptionID))
	if checkoutIntegrationUpdateRequest.Quantity == 0 {
		api.log.Error("quantity not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	api.log.Debug("quantity", zap.Any("quantity", checkoutIntegrationUpdateRequest.Quantity))
	secret, err := api.paymentHandler.UpdateCheckoutSubscription(c, tokenDetails, checkoutIntegrationUpdateRequest)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("UpdateCheckoutSubscription", zap.Any("secret", secret))
	c.IndentedJSON(http.StatusOK, secret)
}

func (api *Api) getCheckoutProduct(c *gin.Context) {
	api.log.Info("getCheckoutProduct")
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}

	var checkoutProductRequest stripemanager.CheckoutProductRequest
	if err := c.BindJSON(&checkoutProductRequest); err != nil {
		api.log.Error("Error binding JSON", zap.Error(err))
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if checkoutProductRequest.SubscriptionID == "" {
		api.log.Error("SubscriptionID not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "SubscriptionID not found"})
		return
	}
	api.log.Debug("subscriptionID", zap.Any("subscriptionID", checkoutProductRequest.SubscriptionID))
	checkoutProductReply, err := api.paymentHandler.GetCheckoutProduct(c, tokenDetails, checkoutProductRequest)
	if err != nil {
		api.log.Error("Error getting checkout product", zap.Error(err))
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	api.log.Info("GetCheckoutProduct", zap.Any("checkoutProductReply", checkoutProductReply))
	c.IndentedJSON(http.StatusOK, checkoutProductReply)
}
