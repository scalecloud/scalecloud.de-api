package apimanager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
	"go.uber.org/zap"
)

func (api *Api) getSubscriptionsOverview(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c, getBearerToken(c))
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}
	subscriptionsOverview, error := api.paymentHandler.StripeConnection.GetSubscriptionsOverview(c, tokenDetails)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("getSubscriptionsOverview", zap.Any("subscriptionsOverview", subscriptionsOverview))
	if subscriptionsOverview != nil {
		c.IndentedJSON(http.StatusOK, subscriptionsOverview)
	} else {
		c.SecureJSON(http.StatusNotFound, gin.H{"message": "subscriptionsOverview not found"})
	}
}

func (api *Api) getSubscriptionByID(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c, getBearerToken(c))
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}
	subscriptionID := c.Param("id")
	api.log.Debug("getSubscriptionByID", zap.String("subscriptionID", subscriptionID))
	subscriptionDetail, error := api.paymentHandler.StripeConnection.GetSubscriptionDetailByID(c, tokenDetails, subscriptionID)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("Found subscriptionDetail", zap.Any("subscriptionDetail", subscriptionDetail))
	if subscriptionDetail != (stripemanager.SubscriptionDetail{}) {
		c.IndentedJSON(http.StatusOK, subscriptionDetail)
	} else {
		c.SecureJSON(http.StatusNotFound, gin.H{"message": "subscriptionDetail not found"})
	}
}

func (api *Api) handleBillingPortal(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c, getBearerToken(c))
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}
	billingPortal, error := api.paymentHandler.GetBillingPortal(c, tokenDetails)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("getBillingPortal", zap.Any("billingPortal", billingPortal))
	c.IndentedJSON(http.StatusOK, billingPortal)
}

func (api *Api) resumeSubscription(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c, getBearerToken(c))
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}

	var subscriptionResumeRequest stripemanager.SubscriptionResumeRequest
	if err := c.BindJSON(&subscriptionResumeRequest); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if subscriptionResumeRequest.ID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "ID not found"})
		return
	}
	reply, error := api.paymentHandler.ResumeSubscription(c, tokenDetails, subscriptionResumeRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("resumeSubscription", zap.Any("Resume", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func (api *Api) cancelSubscription(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c, getBearerToken(c))
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}

	var subscriptionCancelRequest stripemanager.SubscriptionCancelRequest
	if err := c.BindJSON(&subscriptionCancelRequest); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if subscriptionCancelRequest.ID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "ID not found"})
		return
	}
	reply, error := api.paymentHandler.CancelSubscription(c, tokenDetails, subscriptionCancelRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("CancelSubscription", zap.Any("Cancel", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func (api *Api) getSubscriptionPaymentMethod(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c, getBearerToken(c))
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}

	var subscriptionPaymentMethodRequest stripemanager.SubscriptionPaymentMethodRequest
	if err := c.BindJSON(&subscriptionPaymentMethodRequest); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if subscriptionPaymentMethodRequest.ID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "ID not found"})
		return
	}
	reply, error := api.paymentHandler.StripeConnection.GetSubscriptionPaymentMethod(c, tokenDetails, subscriptionPaymentMethodRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("getSubscriptionPaymentMethod", zap.Any("SubscriptionPaymentMethodRequest", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func (api *Api) getChangePaymentSetupIntent(c *gin.Context) {
	tokenDetails, err := api.paymentHandler.FirebaseConnection.GetTokenDetails(c, getBearerToken(c))
	if err != nil {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": "Error getting token details"})
		return
	}

	var request stripemanager.ChangePaymentRequest
	if err := c.BindJSON(&request); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if request.SubscriptionID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "SubscriptionID not found"})
		return
	}
	reply, error := api.paymentHandler.StripeConnection.GetChangePaymentSetupIntent(c, tokenDetails, request)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	api.log.Info("getChangePaymentSetupIntent", zap.Any("getChangePaymentSetupIntent", reply))
	c.IndentedJSON(http.StatusOK, reply)
}
