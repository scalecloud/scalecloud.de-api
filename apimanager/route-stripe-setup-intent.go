package apimanager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

func (api *Api) createCheckoutSetupIntent(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	var checkoutSetupIntentRequest stripemanager.CheckoutSetupIntentRequest
	if err := c.BindJSON(&checkoutSetupIntentRequest); err != nil {
		api.log.Error("Error binding JSON", zap.Error(err))
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if checkoutSetupIntentRequest.ProductID == "" {
		api.log.Error("productID not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "productID not found"})
		return
	}
	api.log.Debug("productID", zap.Any("productID", checkoutSetupIntentRequest.ProductID))
	if checkoutSetupIntentRequest.Quantity == 0 {
		api.log.Error("quantity not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	api.log.Debug("quantity", zap.Any("quantity", checkoutSetupIntentRequest.Quantity))
	secret, error := api.paymentHandler.CreateCheckoutSetupIntent(c, tokenDetails, checkoutSetupIntentRequest)
	if error != nil {
		api.log.Error("Error creating checkout setup intent", zap.Error(error))
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("CreateSubscription", zap.Any("secret", secret))
	c.IndentedJSON(http.StatusOK, secret)
}
