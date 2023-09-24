package apimanager

import (
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
)

func (api *Api) createCheckoutSession(c *gin.Context) {
	var request stripemanager.CheckoutModelPortalRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.CreateCheckoutSession(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}

}
