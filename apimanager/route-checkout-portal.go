package apimanager

import (
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
)

func (api *Api) createCheckoutSession(c *gin.Context) {
	var request stripemanager.CheckoutModelPortalRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil &&
		api.hasNoError(c, c.BindJSON(&request)) &&
		api.hasNoError(c, validateStruct(request)) {
		reply, err := api.paymentHandler.CreateCheckoutSession(c, tokenDetails, request)
		if api.hasNoError(c, validateStruct(reply)) {
			api.writeReply(c, err, reply)
		}
	}

}
