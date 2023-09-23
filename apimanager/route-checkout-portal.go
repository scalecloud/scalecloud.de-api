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
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}
	var checkoutModelPortalRequest stripemanager.CheckoutModelPortalRequest
	if err := c.BindJSON(&checkoutModelPortalRequest); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if checkoutModelPortalRequest.ProductID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "productID not found"})
		return
	}
	api.log.Debug("productID", zap.Any("productID", checkoutModelPortalRequest.ProductID))
	if checkoutModelPortalRequest.Quantity == 0 {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	api.log.Debug("quantity", zap.Any("quantity", checkoutModelPortalRequest.Quantity))
	checkout, error := api.paymentHandler.CreateCheckoutSession(c, tokenDetails, checkoutModelPortalRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("CreateCheckoutSession", zap.Any("checkout", checkout))
	c.IndentedJSON(http.StatusOK, checkout)
}
