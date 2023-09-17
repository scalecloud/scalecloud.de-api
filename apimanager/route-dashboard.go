package apimanager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripebillingportal"
	"github.com/scalecloud/scalecloud.de-api/stripechangepayment"
	"github.com/scalecloud/scalecloud.de-api/stripepaymentintent"
	"github.com/scalecloud/scalecloud.de-api/stripesubscription"
	"go.uber.org/zap"
)

func getSubscriptionsOverview(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}
	subscriptionsOverview, error := stripesubscription.GetSubscriptionsOverview(c, token)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	logger.Info("getSubscriptionsOverview", zap.Any("subscriptionsOverview", subscriptionsOverview))
	if subscriptionsOverview != nil {
		c.IndentedJSON(http.StatusOK, subscriptionsOverview)
	} else {
		c.SecureJSON(http.StatusNotFound, gin.H{"message": "subscriptionsOverview not found"})
	}
}

func getSubscriptionByID(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}
	subscriptionID := c.Param("id")
	logger.Debug("getSubscriptionByID", zap.String("subscriptionID", subscriptionID))
	subscriptionDetail, error := stripesubscription.GetSubscriptionDetailByID(c, token, subscriptionID)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	logger.Info("Found subscriptionDetail", zap.Any("subscriptionDetail", subscriptionDetail))
	if subscriptionDetail != (stripesubscription.SubscriptionDetail{}) {
		c.IndentedJSON(http.StatusOK, subscriptionDetail)
	} else {
		c.SecureJSON(http.StatusNotFound, gin.H{"message": "subscriptionDetail not found"})
	}
}

func getBillingPortal(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}
	billingPortal, error := stripebillingportal.GetBillingPortal(c, token)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	logger.Info("getBillingPortal", zap.Any("billingPortal", billingPortal))
	c.IndentedJSON(http.StatusOK, billingPortal)
}

func resumeSubscription(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}

	var subscriptionResumeRequest stripesubscription.SubscriptionResumeRequest
	if err := c.BindJSON(&subscriptionResumeRequest); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if subscriptionResumeRequest.ID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "ID not found"})
		return
	}
	reply, error := stripesubscription.ResumeSubscription(c, token, subscriptionResumeRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	logger.Info("resumeSubscription", zap.Any("Resume", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func cancelSubscription(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}

	var subscriptionCancelRequest stripesubscription.SubscriptionCancelRequest
	if err := c.BindJSON(&subscriptionCancelRequest); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if subscriptionCancelRequest.ID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "ID not found"})
		return
	}
	reply, error := stripesubscription.CancelSubscription(c, token, subscriptionCancelRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	logger.Info("CancelSubscription", zap.Any("Cancel", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func getSubscriptionPaymentMethod(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}

	var subscriptionPaymentMethodRequest stripepaymentintent.SubscriptionPaymentMethodRequest
	if err := c.BindJSON(&subscriptionPaymentMethodRequest); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if subscriptionPaymentMethodRequest.ID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "ID not found"})
		return
	}
	reply, error := stripepaymentintent.GetSubscriptionPaymentMethod(c, token, subscriptionPaymentMethodRequest)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	logger.Info("getSubscriptionPaymentMethod", zap.Any("SubscriptionPaymentMethodRequest", reply))
	c.IndentedJSON(http.StatusOK, reply)
}

func getChangePaymentSetupIntent(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}

	var request stripechangepayment.ChangePaymentRequest
	if err := c.BindJSON(&request); err != nil {
		c.SecureJSON(http.StatusUnsupportedMediaType, gin.H{"message": "Invalid JSON"})
		return
	}

	if request.SubscriptionID == "" {
		c.SecureJSON(http.StatusBadRequest, gin.H{"message": "SubscriptionID not found"})
		return
	}
	reply, error := stripechangepayment.GetChangePaymentSetupIntent(c, token, request)
	if error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": error.Error()})
		return
	}
	logger.Info("getChangePaymentSetupIntent", zap.Any("getChangePaymentSetupIntent", reply))
	c.IndentedJSON(http.StatusOK, reply)
}
