package apimanager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

func (api *Api) getSubscriptionsOverview(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}
	reply, err := api.paymentHandler.GetSubscriptionsOverview(c, tokenDetails)
	if err != nil {
		api.log.Error("Error getting subscriptionsOverview", zap.Error(err))
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": err.Error()})
		return
	}
	err = validateStruct(reply)
	if err != nil {
		api.log.Warn("Reply", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("getSubscriptionsOverview", zap.Any("subscriptionsOverview", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func (api *Api) getSubscriptionByID(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}
	subscriptionID := c.Param("id")
	reply, err := api.paymentHandler.GetSubscriptionDetailByID(c, tokenDetails, subscriptionID)
	if err != nil {
		api.log.Error("Error getting subscriptionDetail", zap.Error(err))
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": err.Error()})
		return
	}
	err = validateStruct(reply)
	if err != nil {
		api.log.Warn("Reply", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("getSubscriptionByID", zap.Any("subscriptionDetail", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func (api *Api) handleBillingPortal(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	reply, err := api.paymentHandler.GetBillingPortal(c, tokenDetails)
	if err != nil {
		api.log.Error("Error getting billingPortal", zap.Error(err))
		c.IndentedJSON(http.StatusNoContent, gin.H{"message": err.Error()})
		return
	}
	err = validateStruct(reply)
	if err != nil {
		api.log.Warn("Reply", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("getBillingPortal", zap.Any("billingPortal", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func (api *Api) resumeSubscription(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	var request stripemanager.SubscriptionResumeRequest
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
	reply, err := api.paymentHandler.ResumeSubscription(c, tokenDetails, request)
	if err != nil {
		api.log.Error("Error resuming subscription", zap.Error(err))
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	err = validateStruct(reply)
	if err != nil {
		api.log.Warn("Reply", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("resumeSubscription", zap.Any("Resume", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func (api *Api) cancelSubscription(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	var request stripemanager.SubscriptionCancelRequest
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
	reply, err := api.paymentHandler.CancelSubscription(c, tokenDetails, request)
	if err != nil {
		api.log.Error("Error canceling subscription", zap.Error(err))
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	err = validateStruct(reply)
	if err != nil {
		api.log.Warn("Reply", zap.Error(err))
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("CancelSubscription", zap.Any("Cancel", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func (api *Api) getSubscriptionPaymentMethod(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	var request stripemanager.SubscriptionPaymentMethodRequest
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
	reply, err := api.paymentHandler.GetSubscriptionPaymentMethod(c, tokenDetails, request)
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
	api.log.Info("getSubscriptionPaymentMethod", zap.Any("SubscriptionPaymentMethodRequest", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func (api *Api) getChangePaymentSetupIntent(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	var request stripemanager.ChangePaymentRequest
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
	reply, err := api.paymentHandler.GetChangePaymentSetupIntent(c, tokenDetails, request)
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
	api.log.Info("getChangePaymentSetupIntent", zap.Any("getChangePaymentSetupIntent", reply))
	c.IndentedJSON(http.StatusOK, reply)
}
