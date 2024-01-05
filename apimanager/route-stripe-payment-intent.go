package apimanager

import (
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
)

func (api *Api) createCheckoutSubscription(c *gin.Context) {
	var request stripemanager.CheckoutCreateSubscriptionRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.CreateCheckoutSubscription(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) getCheckoutProduct(c *gin.Context) {
	var request stripemanager.CheckoutProductRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.GetCheckoutProduct(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}
