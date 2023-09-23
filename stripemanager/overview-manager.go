package stripemanager

import (
	"context"
	"errors"
	"strconv"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/subscription"
	"go.uber.org/zap"
)

func (paymentHandler *PaymentHandler) GetSubscriptionsOverview(c context.Context, tokenDetails firebasemanager.TokenDetails) (subscriptionOverview []SubscriptionOverview, err error) {
	customerID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return []SubscriptionOverview{}, err
	}
	subscriptions := []SubscriptionOverview{}
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
			return []SubscriptionOverview{}, errors.New("Subscription not found")
		}
		subscriptions = append(subscriptions, subscriptionOverview)
	}
	if len(subscriptions) == 0 {
		return []SubscriptionOverview{}, errors.New("No subscriptions found")
	}
	return subscriptions, nil
}

func (stripeConnection *StripeConnection) mapSubscriptionToSubscriptionOverview(c context.Context, subscription *stripe.Subscription) (subscriptionOverview SubscriptionOverview, err error) {
	subscriptionOverview.ID = subscription.ID
	productID := subscription.Items.Data[0].Price.Product.ID
	stripeConnection.Log.Debug("Product ID", zap.String("productID", productID))

	product, err := stripeConnection.GetProduct(c, productID)
	if err != nil {
		return SubscriptionOverview{}, errors.New("Product not found")
	}
	subscriptionOverview.ProductName = product.Name

	subscriptionOverview.Acive = product.Active

	metaData := product.Metadata
	if err != nil {
		return SubscriptionOverview{}, errors.New("Product metadata not found")
	}
	storageAmount, ok := metaData["storageAmount"]
	if !ok {
		return SubscriptionOverview{}, errors.New("Storage amount not found")
	}
	iStorageAmount, err := strconv.Atoi(storageAmount)
	if err != nil {
		stripeConnection.Log.Warn("Error converting storage amount to int", zap.Error(err))
		return SubscriptionOverview{}, errors.New("Error converting storage amount")
	}
	subscriptionOverview.StorageAmount = iStorageAmount
	productType, ok := metaData["productType"]
	if !ok {
		return SubscriptionOverview{}, errors.New("ProductType not found")
	}
	subscriptionOverview.ProductType = productType

	subscriptionOverview.UserCount = subscription.Items.Data[0].Quantity
	return subscriptionOverview, nil
}
