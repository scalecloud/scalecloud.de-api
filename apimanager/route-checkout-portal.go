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
	err = c.BindJSON(&request)
	if err != nil {
		api.log.Warn("Error binding JSON", zap.Error(err))
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": err.Error()})
		return
	}
	err = validateStruct(request)
	if err != nil {
		api.log.Warn("Error validating struct", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	reply, err := api.paymentHandler.CreateCheckoutSession(c, tokenDetails, request)
	if err != nil {
		api.log.Error("Error creating checkout session", zap.Error(err))
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	err = validateStruct(reply)
	if err != nil {
		api.log.Warn("Reply", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("CreateCheckoutSession", zap.Any("checkout", reply))
	c.IndentedJSON(http.StatusOK, "checkout")
}
