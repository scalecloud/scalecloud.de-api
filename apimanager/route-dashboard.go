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
	subscriptionsOverview, err := api.paymentHandler.GetSubscriptionsOverview(c, tokenDetails)
	if err != nil {
		api.log.Error("Error getting subscriptionsOverview", zap.Error(err))
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": err.Error()})
		return
	}
	if subscriptionsOverview != nil {
		api.log.Info("getSubscriptionsOverview", zap.Any("subscriptionsOverview", subscriptionsOverview))
		c.IndentedJSON(http.StatusOK, subscriptionsOverview)
	} else {
		api.log.Error("subscriptionsOverview not found")
		c.SecureJSON(http.StatusNotFound, gin.H{"message": "subscriptionsOverview not found"})
	}
}

func (api *Api) getSubscriptionByID(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}
	subscriptionID := c.Param("id")
	api.log.Debug("getSubscriptionByID", zap.String("subscriptionID", subscriptionID))
	subscriptionDetail, err := api.paymentHandler.GetSubscriptionDetailByID(c, tokenDetails, subscriptionID)
	if err != nil {
		api.log.Error("Error getting subscriptionDetail", zap.Error(err))
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": err.Error()})
		return
	}
	if subscriptionDetail != (stripemanager.SubscriptionDetail{}) {
		api.log.Info("getSubscriptionByID", zap.Any("subscriptionDetail", subscriptionDetail))
		c.IndentedJSON(http.StatusOK, subscriptionDetail)
	} else {
		api.log.Error("subscriptionDetail not found")
		c.SecureJSON(http.StatusNotFound, gin.H{"message": "subscriptionDetail not found"})
	}
}

func (api *Api) handleBillingPortal(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	billingPortal, err := api.paymentHandler.GetBillingPortal(c, tokenDetails)
	if err != nil {
		api.log.Error("Error getting billingPortal", zap.Error(err))
		c.IndentedJSON(http.StatusNoContent, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("getBillingPortal", zap.Any("billingPortal", billingPortal))
	c.IndentedJSON(http.StatusOK, billingPortal)
}

func (api *Api) resumeSubscription(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c)
	if err != nil {
		api.log.Error("Error getting token details", zap.Error(err))
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	var subscriptionResumeRequest stripemanager.SubscriptionResumeRequest
	if err := c.BindJSON(&subscriptionResumeRequest); err != nil {
		api.log.Error("Error binding JSON", zap.Error(err))
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}
	if subscriptionResumeRequest.ID == "" {
		api.log.Error("ID not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "ID not found"})
		return
	}
	reply, err := api.paymentHandler.ResumeSubscription(c, tokenDetails, subscriptionResumeRequest)
	if err != nil {
		api.log.Error("Error resuming subscription", zap.Error(err))
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
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

	var subscriptionCancelRequest stripemanager.SubscriptionCancelRequest
	if err := c.BindJSON(&subscriptionCancelRequest); err != nil {
		api.log.Error("Error binding JSON", zap.Error(err))
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if subscriptionCancelRequest.ID == "" {
		api.log.Error("ID not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "ID not found"})
		return
	}
	reply, err := api.paymentHandler.CancelSubscription(c, tokenDetails, subscriptionCancelRequest)
	if err != nil {
		api.log.Error("Error canceling subscription", zap.Error(err))
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	var subscriptionPaymentMethodRequest stripemanager.SubscriptionPaymentMethodRequest
	if err := c.BindJSON(&subscriptionPaymentMethodRequest); err != nil {
		api.log.Error("Error binding JSON", zap.Error(err))
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if subscriptionPaymentMethodRequest.ID == "" {
		api.log.Error("ID not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "ID not found"})
		return
	}
	reply, err := api.paymentHandler.GetSubscriptionPaymentMethod(c, tokenDetails, subscriptionPaymentMethodRequest)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
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
	if err := c.BindJSON(&request); err != nil {
		api.log.Error("Error binding JSON", zap.Error(err))
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if request.SubscriptionID == "" {
		api.log.Error("SubscriptionID not found")
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "SubscriptionID not found"})
		return
	}
	reply, err := api.paymentHandler.GetChangePaymentSetupIntent(c, tokenDetails, request)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	api.log.Info("getChangePaymentSetupIntent", zap.Any("getChangePaymentSetupIntent", reply))
	c.IndentedJSON(http.StatusOK, reply)
}
