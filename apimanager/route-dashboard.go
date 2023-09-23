package apimanager

import (
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
)

func (api *Api) getSubscriptionsOverview(c *gin.Context) {
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil {
		reply, err := api.paymentHandler.GetSubscriptionsOverview(c, tokenDetails)
		if api.hasNoError(c, validateStruct(reply)) {
			api.writeReply(c, err, reply)
		}
	}
}

func (api *Api) getSubscriptionByID(c *gin.Context) {
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil {
		subscriptionID := c.Param("id")
		reply, err := api.paymentHandler.GetSubscriptionDetailByID(c, tokenDetails, subscriptionID)
		if api.hasNoError(c, validateStruct(reply)) {
			api.writeReply(c, err, reply)
		}
	}
}

func (api *Api) handleBillingPortal(c *gin.Context) {
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil {
		reply, err := api.paymentHandler.GetBillingPortal(c, tokenDetails)
		if api.hasNoError(c, validateStruct(reply)) {
			api.writeReply(c, err, reply)
		}
	}
}

func (api *Api) resumeSubscription(c *gin.Context) {
	var request stripemanager.SubscriptionResumeRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil &&
		api.hasNoError(c, c.BindJSON(&request)) &&
		api.hasNoError(c, validateStruct(request)) {
		reply, err := api.paymentHandler.ResumeSubscription(c, tokenDetails, request)
		if api.hasNoError(c, validateStruct(reply)) {
			api.writeReply(c, err, reply)
		}
	}
}

func (api *Api) cancelSubscription(c *gin.Context) {
	var request stripemanager.SubscriptionCancelRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil &&
		api.hasNoError(c, c.BindJSON(&request)) &&
		api.hasNoError(c, validateStruct(request)) {
		reply, err := api.paymentHandler.CancelSubscription(c, tokenDetails, request)
		if api.hasNoError(c, validateStruct(reply)) {
			api.writeReply(c, err, reply)
		}
	}
}

func (api *Api) getSubscriptionPaymentMethod(c *gin.Context) {
	var request stripemanager.SubscriptionPaymentMethodRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil &&
		api.hasNoError(c, c.BindJSON(&request)) &&
		api.hasNoError(c, validateStruct(request)) {
		reply, err := api.paymentHandler.GetSubscriptionPaymentMethod(c, tokenDetails, request)
		if api.hasNoError(c, validateStruct(reply)) {
			api.writeReply(c, err, reply)
		}
	}
}

func (api *Api) getChangePaymentSetupIntent(c *gin.Context) {
	var request stripemanager.ChangePaymentRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil {
		reply, err := api.paymentHandler.GetChangePaymentSetupIntent(c, tokenDetails, request)
		if api.hasNoError(c, validateStruct(reply)) {
			api.writeReply(c, err, reply)
		}
	}
}
