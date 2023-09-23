package apimanager

import (
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
)

func (api *Api) createCheckoutSubscription(c *gin.Context) {
	var request stripemanager.CheckoutPaymentIntentRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil &&
		api.hasNoError(c, c.BindJSON(&request)) &&
		api.hasNoError(c, validateStruct(request)) {
		reply, err := api.paymentHandler.CreateCheckoutSubscription(c, tokenDetails, request)
		if api.hasNoError(c, validateStruct(reply)) {
			api.writeReply(c, err, reply)
		}
	}
}

func (api *Api) updateCheckoutSubscription(c *gin.Context) {
	var request stripemanager.CheckoutPaymentIntentUpdateRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil &&
		api.hasNoError(c, c.BindJSON(&request)) &&
		api.hasNoError(c, validateStruct(request)) {
		reply, err := api.paymentHandler.UpdateCheckoutSubscription(c, tokenDetails, request)
		if api.hasNoError(c, validateStruct(reply)) {
			api.writeReply(c, err, reply)
		}
	}
}

func (api *Api) getCheckoutProduct(c *gin.Context) {
	var request stripemanager.CheckoutProductRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil &&
		api.hasNoError(c, c.BindJSON(&request)) &&
		api.hasNoError(c, validateStruct(request)) {
		reply, err := api.paymentHandler.GetCheckoutProduct(c, tokenDetails, request)
		if api.hasNoError(c, validateStruct(reply)) {
			api.writeReply(c, err, reply)
		}
	}
}
