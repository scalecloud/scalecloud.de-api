package apimanager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

func CreateCheckoutSetupIntent(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}

	var checkoutSetupIntentRequest stripemanager.CheckoutSetupIntentRequest
	if err := c.BindJSON(&checkoutSetupIntentRequest); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if checkoutSetupIntentRequest.ProductID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "productID not found"})
		return
	}
	logger.Debug("productID", zap.Any("productID", checkoutSetupIntentRequest.ProductID))
	if checkoutSetupIntentRequest.Quantity == 0 {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	logger.Debug("quantity", zap.Any("quantity", checkoutSetupIntentRequest.Quantity))
	secret, error := stripemanager.CreateCheckoutSetupIntent(c, token, checkoutSetupIntentRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	logger.Info("CreateSubscription", zap.Any("secret", secret))
	c.IndentedJSON(http.StatusOK, secret)
}
