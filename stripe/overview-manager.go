package stripe

import (
	"context"
	"errors"
	"strconv"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/sub"
	"go.uber.org/zap"
)

func GetSubscriptionsOverview(c context.Context, customerID string) (subscriptionOverview []SubscriptionOverview, err error) {
	if customerID == "" {
		logger.Error("Customer ID is empty")
		return []SubscriptionOverview{}, errors.New("Customer ID is empty")
	}
	subscriptions := []SubscriptionOverview{}
	stripe.Key = getStripeKey()
	params := &stripe.SubscriptionListParams{
		Customer: customerID,
	}
	iter := sub.List(params)
	for iter.Next() {
		subscription := iter.Subscription()
		logger.Debug("Subscription", zap.Any("subscription", subscription.Customer.ID))
		subscriptionOverview, err := mapSubscriptionToSubscriptionOverview(c, subscription)
		if err != nil {
			logger.Warn("Error mapping subscription to subscription detail", zap.Error(err))
			return []SubscriptionOverview{}, errors.New("Subscription not found")
		}
		subscriptions = append(subscriptions, subscriptionOverview)
	}
	if len(subscriptions) == 0 {
		logger.Error("No subscriptions found", zap.String("customerID", customerID))
		return []SubscriptionOverview{}, errors.New("No subscriptions found")
	}
	return subscriptions, nil
}

func mapSubscriptionToSubscriptionOverview(c context.Context, subscription *stripe.Subscription) (subscriptionOverview SubscriptionOverview, err error) {
	subscriptionOverview.ID = subscription.ID
	productID := subscription.Plan.Product.ID
	logger.Debug("Product ID", zap.String("productID", productID))

	product, err := getProduct(c, productID)
	if err != nil {
		logger.Warn("Error getting product", zap.Error(err))
		return SubscriptionOverview{}, errors.New("Product not found")
	}
	subscriptionOverview.ProductName = product.Name

	subscriptionOverview.Acive = product.Active

	metaData := product.Metadata
	if err != nil {
		logger.Warn("Error getting product metadata", zap.Error(err))
		return SubscriptionOverview{}, errors.New("Product metadata not found")
	}
	storageAmount, ok := metaData["storageAmount"]
	if !ok {
		logger.Warn("Storage amount not found", zap.Any("subscriptionID", subscription.ID))
		return SubscriptionOverview{}, errors.New("Storage amount not found")
	}
	iStorageAmount, err := strconv.Atoi(storageAmount)
	if err != nil {
		logger.Warn("Error converting storage amount to int", zap.Error(err))
		return SubscriptionOverview{}, errors.New("Error converting storage amount to int")
	}
	subscriptionOverview.StorageAmount = iStorageAmount
	productType, ok := metaData["productType"]
	if !ok {
		logger.Warn("ProductType not found", zap.Any("subscriptionID", subscription.ID))
		return SubscriptionOverview{}, errors.New("ProductType not found")
	}
	subscriptionOverview.ProductType = productType

	subscriptionOverview.UserCount = subscription.Quantity
	return subscriptionOverview, nil
}
