package apimanager

import (
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
)

func (api *Api) createCheckoutSetupIntent(c *gin.Context) {
	var request stripemanager.CheckoutSetupIntentRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.CreateCheckoutSetupIntent(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}
