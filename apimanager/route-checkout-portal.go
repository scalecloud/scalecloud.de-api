package apimanager

import (
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
)

func (api *Api) createCheckoutSession(c *gin.Context) {
	var request stripemanager.CheckoutModelPortalRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil &&
		api.checkBind(c, c.BindJSON(&request)) &&
		api.checkValidate(c, validateStruct(request)) {
		reply, err := api.paymentHandler.CreateCheckoutSession(c, tokenDetails, request)
		api.writeReply(err, c, reply)
	}

}
