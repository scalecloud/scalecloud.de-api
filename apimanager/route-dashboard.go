package apimanager

import (
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
)

func (api *Api) getSubscriptionsOverview(c *gin.Context) {
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil {
		reply, err := api.paymentHandler.GetSubscriptionsOverview(c, tokenDetails)
		for _, s := range reply {
			if !api.validateReply(c, err, s) {
				return
			}
		}
		api.writeReply(c, reply)
	}
}

func (api *Api) getSubscriptionByID(c *gin.Context) {
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil {
		subscriptionID := c.Param("id")
		reply, err := api.paymentHandler.GetSubscriptionDetailByID(c, tokenDetails, subscriptionID)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) handleBillingPortal(c *gin.Context) {
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil {
		reply, err := api.paymentHandler.GetBillingPortal(c, tokenDetails)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) resumeSubscription(c *gin.Context) {
	var request stripemanager.SubscriptionResumeRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.ResumeSubscription(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) cancelSubscription(c *gin.Context) {
	var request stripemanager.SubscriptionCancelRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.CancelSubscription(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) getPaymentMethodOverview(c *gin.Context) {
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil {
		reply, err := api.paymentHandler.GetPaymentMethodOverview(c, tokenDetails)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) getChangePaymentSetupIntent(c *gin.Context) {
	var request stripemanager.ChangePaymentRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.GetChangePaymentSetupIntent(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}
