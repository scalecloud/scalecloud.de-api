package apimanager

import (
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/newslettermanager"
)

func (api *Api) newsletterSubscribe(c *gin.Context) {
	var request newslettermanager.NewsletterSubscribeRequest
	if api.handleBind(c, &request) {
		reply, err := api.paymentHandler.NewsletterConnection.NewsletterSubscribe(c, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) newsletterConfirm(c *gin.Context) {
	var request newslettermanager.NewsletterConfirmRequest
	if api.handleBind(c, &request) {
		reply, err := api.paymentHandler.NewsletterConnection.NewsletterConfirm(c, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) newsletterUnsubscribe(c *gin.Context) {
	var request newslettermanager.NewsletterUnsubscribeRequest
	if api.handleBind(c, &request) {
		reply, err := api.paymentHandler.NewsletterConnection.NewsletterUnsubscribe(c, request)
		api.validateAndWriteReply(c, err, reply)
	}
}
