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
	var request stripemanager.CheckoutPaymentIntentRequest
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
	reply, err := api.paymentHandler.CreateCheckoutSubscription(c, tokenDetails, request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	err = validateStruct(reply)
	if err != nil {
		api.log.Warn("Reply", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("CreateSubscription", zap.Any("secret", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func (api *Api) updateCheckoutSubscription(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	var request stripemanager.CheckoutPaymentIntentUpdateRequest
	err = c.BindJSON(&request)
	if err != nil {
		api.log.Warn("Error binding JSON", zap.Error(err))
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": err.Error()})
		return
	}
	reply, err := api.paymentHandler.UpdateCheckoutSubscription(c, tokenDetails, request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	err = validateStruct(reply)
	if err != nil {
		api.log.Warn("Reply", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("UpdateCheckoutSubscription", zap.Any("secret", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func (api *Api) getCheckoutProduct(c *gin.Context) {
	api.log.Info("getCheckoutProduct")
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}
	var request stripemanager.CheckoutProductRequest
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
	reply, err := api.paymentHandler.GetCheckoutProduct(c, tokenDetails, request)
	if err != nil {
		api.log.Error("Error getting checkout product", zap.Error(err))
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	err = validateStruct(reply)
	if err != nil {
		api.log.Warn("Reply", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("GetCheckoutProduct", zap.Any("checkoutProductReply", reply))
	c.IndentedJSON(http.StatusOK, reply)
}
