package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/scalecloud.de-api"
	"github.com/scalecloud/scalecloud.de-api/stripe"
	"go.uber.org/zap"
)

func getSubscriptionsOverview(c *gin.Context) {
	token, ok := getBearerToken(c)
	if !ok {
		c.SecureJSON(http.StatusUnauthorized, gin.H{"message": messageBearer})
		return
	}
	subscriptionsOverview, error := scalecloud.GetSubscriptionsOverview(c, token)
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
	subscriptionDetail, error := scalecloud.GetSubscriptionByID(c, token, subscriptionID)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	logger.Info("Found subscriptionDetail", zap.Any("subscriptionDetail", subscriptionDetail))
	if subscriptionDetail != (stripe.SubscriptionDetail{}) {
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
	billingPortal, error := scalecloud.GetBillingPortal(c, token)
	if error != nil {
		c.IndentedJSON(http.StatusNoContent, gin.H{"error": error.Error()})
		return
	}
	logger.Info("getBillingPortal", zap.Any("billingPortal", billingPortal))
	c.IndentedJSON(http.StatusOK, billingPortal)
}
