package stripemanager

import (
	"context"
	"errors"
	"strconv"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/subscription"
	"go.uber.org/zap"
)

func (paymentHandler *PaymentHandler) GetSubscriptionsOverview(c context.Context, tokenDetails firebasemanager.TokenDetails) (subscriptionOverview []SubscriptionOverviewReply, err error) {
	customerID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return []SubscriptionOverviewReply{}, err
	}
	subscriptions := []SubscriptionOverviewReply{}
	stripe.Key = paymentHandler.StripeConnection.Key
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
	}
	iter := subscription.List(params)
	for iter.Next() {
		subscription := iter.Subscription()
		paymentHandler.Log.Debug("Subscription", zap.Any("subscription", subscription.Customer.ID))
		subscriptionOverview, err := paymentHandler.StripeConnection.mapSubscriptionToSubscriptionOverview(c, subscription)
		if err != nil {
			return []SubscriptionOverviewReply{}, errors.New("Subscription not found")
		}
		subscriptions = append(subscriptions, subscriptionOverview)
	}
	if len(subscriptions) == 0 {
		return []SubscriptionOverviewReply{}, errors.New("No subscriptions found")
	}
	return subscriptions, nil
}

func (stripeConnection *StripeConnection) mapSubscriptionToSubscriptionOverview(c context.Context, subscription *stripe.Subscription) (reply SubscriptionOverviewReply, err error) {
	reply.ID = subscription.ID
	productID := subscription.Items.Data[0].Price.Product.ID
	stripeConnection.Log.Debug("Product ID", zap.String("productID", productID))

	product, err := stripeConnection.GetProduct(c, productID)
	if err != nil {
		return SubscriptionOverviewReply{}, errors.New("Product not found")
	}
	reply.ProductName = product.Name

	reply.Acive = &product.Active

	metaData := product.Metadata
	if err != nil {
		return SubscriptionOverviewReply{}, errors.New("Product metadata not found")
	}
	storageAmount, ok := metaData["storageAmount"]
	if !ok {
		return SubscriptionOverviewReply{}, errors.New("Storage amount not found")
	}
	iStorageAmount, err := strconv.Atoi(storageAmount)
	if err != nil {
		stripeConnection.Log.Warn("Error converting storage amount to int", zap.Error(err))
		return SubscriptionOverviewReply{}, errors.New("Error converting storage amount")
	}
	reply.StorageAmount = iStorageAmount
	productType, ok := metaData["productType"]
	if !ok {
		return SubscriptionOverviewReply{}, errors.New("ProductType not found")
	}
	reply.ProductType = productType

	reply.UserCount = subscription.Items.Data[0].Quantity
	return reply, nil
}
