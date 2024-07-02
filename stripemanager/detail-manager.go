package stripemanager

import (
	"context"
	"errors"
	"strconv"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v78"
	"go.uber.org/zap"
)

func (paymentHandler *PaymentHandler) GetSubscriptionDetailByID(c context.Context, tokenDetails firebasemanager.TokenDetails, subscriptionID string) (SubscriptionDetailReply, error) {
	err := paymentHandler.MongoConnection.HasPermission(c, tokenDetails, subscriptionID, []mongomanager.Role{mongomanager.RoleBilling, mongomanager.RoleUser, mongomanager.RoleAdministrator})
	if err != nil {
		return SubscriptionDetailReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key
	subscription, err := paymentHandler.StripeConnection.GetSubscriptionByID(c, subscriptionID)
	if err != nil {
		return SubscriptionDetailReply{}, errors.New("subscription not found")
	}
	paymentHandler.Log.Debug("subscription", zap.Any("subscription", subscription))
	subscriptionDetailReply, err := paymentHandler.StripeConnection.mapSubscriptionItemToSubscriptionDetail(c, subscription)
	if err != nil {
		return SubscriptionDetailReply{}, err
	}
	return subscriptionDetailReply, nil
}

func (stripeConnection *StripeConnection) mapSubscriptionItemToSubscriptionDetail(c context.Context, subscription *stripe.Subscription) (reply SubscriptionDetailReply, err error) {
	reply.ID = subscription.ID
	productID := subscription.Items.Data[0].Price.Product.ID
	stripeConnection.Log.Debug("Product ID", zap.String("productID", productID))

	prod, err := stripeConnection.GetProduct(c, productID)
	if err != nil {
		return SubscriptionDetailReply{}, errors.New("product not found")
	}
	reply.ProductName = prod.Name

	reply.Active = &prod.Active

	metaData := prod.Metadata
	if metaData == nil {
		return SubscriptionDetailReply{}, errors.New("product metadata not found")
	}
	storageAmount, ok := metaData["storageAmount"]
	if !ok {
		return SubscriptionDetailReply{}, errors.New("storage amount not found")
	}
	iStorageAmount, err := strconv.Atoi(storageAmount)
	if err != nil {
		return SubscriptionDetailReply{}, errors.New("error converting storage amount to int")
	}
	reply.StorageAmount = iStorageAmount
	productType, ok := metaData["productType"]
	if !ok {
		return SubscriptionDetailReply{}, errors.New("ProductType not found")
	}
	reply.ProductType = productType

	reply.UserCount = subscription.Items.Data[0].Quantity

	pri, err := stripeConnection.GetPrice(c, subscription.Items.Data[0].Price.Product.ID)
	if err != nil {
		return SubscriptionDetailReply{}, errors.New("price not found")
	}

	reply.PricePerMonth = pri.UnitAmount

	reply.Currency = string(pri.Currency)

	reply.CancelAtPeriodEnd = &subscription.CancelAtPeriodEnd

	reply.CancelAt = subscription.CancelAt

	reply.Status = string(subscription.Status)

	reply.TrialEnd = subscription.TrialEnd

	return reply, nil
}
