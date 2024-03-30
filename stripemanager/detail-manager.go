package stripemanager

import (
	"context"
	"errors"
	"strconv"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v76"
	"go.uber.org/zap"
)

func (paymentHandler *PaymentHandler) GetSubscriptionDetailByID(c context.Context, tokenDetails firebasemanager.TokenDetails, subscriptionID string) (subscriptionDetailReply SubscriptionDetailReply, err error) {
	if subscriptionID == "" {
		return SubscriptionDetailReply{}, errors.New("Subscription ID is empty")
	}
	customerID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return SubscriptionDetailReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key
	subscription, err := paymentHandler.StripeConnection.GetSubscriptionByID(c, subscriptionID)
	if err != nil {
		return SubscriptionDetailReply{}, errors.New("Subscription not found")
	}
	paymentHandler.Log.Debug("subscription", zap.Any("subscription", subscription))
	if subscription.Customer.ID != customerID {
		paymentHandler.Log.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", subscriptionID))
		return SubscriptionDetailReply{}, errors.New("Subscription not matching customer")
	} else {
		subscriptionDetailReply, err = paymentHandler.StripeConnection.mapSubscriptionItemToSubscriptionDetail(c, subscription)
		if err != nil {
			return SubscriptionDetailReply{}, err
		}
	}
	return subscriptionDetailReply, nil
}

func (stripeConnection *StripeConnection) mapSubscriptionItemToSubscriptionDetail(c context.Context, subscription *stripe.Subscription) (reply SubscriptionDetailReply, err error) {
	reply.ID = subscription.ID
	productID := subscription.Items.Data[0].Price.Product.ID
	stripeConnection.Log.Debug("Product ID", zap.String("productID", productID))

	prod, err := stripeConnection.GetProduct(c, productID)
	if err != nil {
		return SubscriptionDetailReply{}, errors.New("Product not found")
	}
	reply.ProductName = prod.Name

	reply.Active = &prod.Active

	metaData := prod.Metadata
	if err != nil {
		return SubscriptionDetailReply{}, errors.New("Product metadata not found")
	}
	storageAmount, ok := metaData["storageAmount"]
	if !ok {
		return SubscriptionDetailReply{}, errors.New("Storage amount not found")
	}
	iStorageAmount, err := strconv.Atoi(storageAmount)
	if err != nil {
		return SubscriptionDetailReply{}, errors.New("Error converting storage amount to int")
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
		return SubscriptionDetailReply{}, errors.New("Price not found")
	}

	reply.PricePerMonth = pri.UnitAmount

	reply.Currency = string(pri.Currency)

	reply.CancelAtPeriodEnd = &subscription.CancelAtPeriodEnd

	reply.CancelAt = subscription.CancelAt

	reply.Status = string(subscription.Status)

	reply.TrialEnd = subscription.TrialEnd

	return reply, nil
}
