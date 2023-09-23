package apimanager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

func (api *Api) createCheckoutSession(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	var request stripemanager.CheckoutModelPortalRequest
	if err := c.BindJSON(&request); err != nil {
		api.log.Error("Error binding JSON", zap.Error(err))
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if request.ProductID == "" {
		api.log.Error("productID not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "productID not found"})
		return
	}
	api.log.Debug("productID", zap.Any("productID", request.ProductID))
	if request.Quantity == 0 {
		api.log.Error("quantity not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	api.log.Debug("quantity", zap.Any("quantity", request.Quantity))
	checkout, err := api.paymentHandler.CreateCheckoutSession(c, tokenDetails, request)
	if err != nil {
		api.log.Error("Error creating checkout session", zap.Error(err))
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("CreateCheckoutSession", zap.Any("checkout", checkout))
	c.IndentedJSON(http.StatusOK, checkout)
}
