package apimanager

import (
	"github.com/gin-gonic/gin"
	"github.com/scalecloud/scalecloud.de-api/stripemanager"
)

func (api *Api) getProductTiersNextcloud(c *gin.Context) {
	reply, err := api.paymentHandler.GetProductTiers(c, stripemanager.ProductNextcloud)
	api.validateAndWriteReply(c, err, reply)
}

func (api *Api) getProductTiersSynology(c *gin.Context) {
	reply, err := api.paymentHandler.GetProductTiers(c, stripemanager.ProductSynology)
	api.validateAndWriteReply(c, err, reply)
}

func (api *Api) getSubscriptionsOverview(c *gin.Context) {
	tokenDetails, err := api.handleTokenDetails(c)
	if err != nil {
		api.validateAndWriteReply(c, err, nil)
		return
	}

	reply, err := api.paymentHandler.GetSubscriptionsOverview(c, tokenDetails)
	if err != nil {
		api.validateAndWriteReply(c, err, nil)
		return
	}

	for _, s := range reply {
		if !api.validateReply(c, err, s) {
			return
		}
	}
	api.writeReply(c, reply)
}

func (api *Api) getSubscriptionByID(c *gin.Context) {
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil {
		subscriptionID := c.Param("id")
		reply, err := api.paymentHandler.GetSubscriptionDetailByID(c, tokenDetails, subscriptionID)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) getCancelState(c *gin.Context) {
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil {
		subscriptionID := c.Param("id")
		reply, err := api.paymentHandler.GetCancelState(c, tokenDetails, subscriptionID)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) GetMyPermission(c *gin.Context) {
	var request stripemanager.PermissionRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.GetMyPermission(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) getSubscriptionListSeats(c *gin.Context) {
	var request stripemanager.ListSeatRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.GetSubscriptionListSeats(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) getSubscriptionSeatDetail(c *gin.Context) {
	var request stripemanager.SeatDetailRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.GetSubscriptionSeatDetail(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) getSubscriptionUpdateSeat(c *gin.Context) {
	var request stripemanager.UpdateSeatDetailRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.GetSubscriptionUpdateSeat(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) getSubscriptionAddSeat(c *gin.Context) {
	var request stripemanager.AddSeatRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.GetSubscriptionAddSeat(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) getSubscriptionRemoveSeat(c *gin.Context) {
	var request stripemanager.DeleteSeatRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.GetSubscriptionRemoveSeat(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) getSubscriptionInvoices(c *gin.Context) {
	var request stripemanager.ListInvoicesRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.GetSubscriptionInvoices(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) getBillingAddress(c *gin.Context) {
	var request stripemanager.BillingAddressRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.GetBillingAddress(c, tokenDetails, request)
		api.validateAndWriteReply(c, err, reply)
	}
}

func (api *Api) updateBillingAddress(c *gin.Context) {
	var request stripemanager.UpdateBillingAddressRequest
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil &&
		api.handleBind(c, &request) {
		reply, err := api.paymentHandler.UpdateBillingAddress(c, tokenDetails, request)
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
	tokenDetails, err := api.handleTokenDetails(c)
	if err == nil {
		reply, err := api.paymentHandler.GetChangePaymentSetupIntent(c, tokenDetails)
		api.validateAndWriteReply(c, err, reply)
	}
}
