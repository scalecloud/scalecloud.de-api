package apimanager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

func (api *Api) createCheckoutSetupIntent(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c, getBearerToken(c))
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
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
	api.log.Debug("productID", zap.Any("productID", checkoutSetupIntentRequest.ProductID))
	if checkoutSetupIntentRequest.Quantity == 0 {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	api.log.Debug("quantity", zap.Any("quantity", checkoutSetupIntentRequest.Quantity))
	secret, error := api.paymentHandler.StripeConnection.CreateCheckoutSetupIntent(c, tokenDetails, checkoutSetupIntentRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("CreateSubscription", zap.Any("secret", secret))
	c.IndentedJSON(http.StatusOK, secret)
}
