package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

func createCheckoutSession(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
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
	logger.Debug("productID", zap.Any("productID", checkoutModelPortalRequest.ProductID))
	if checkoutModelPortalRequest.Quantity == 0 {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "quantity not found"})
		return
	}
	logger.Debug("quantity", zap.Any("quantity", checkoutModelPortalRequest.Quantity))
	checkout, error := stripemanager.CreateCheckoutSession(c, token, checkoutModelPortalRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	logger.Info("CreateCheckoutSession", zap.Any("checkout", checkout))
	c.IndentedJSON(http.StatusOK, checkout)
}
