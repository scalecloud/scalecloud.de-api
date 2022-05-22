package stripe

import (
	"context"
	"errors"
	"strconv"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/sub"
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
		subscriptionOverview, err := mapSubscriptionToSubscriptionOverview(subscription)
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

func mapSubscriptionToSubscriptionOverview(subscription *stripe.Subscription) (subscriptionOverview SubscriptionOverview, err error) {
	subscriptionOverview.ID = subscription.ID
	subscriptionOverview.PlanProductName = subscription.Plan.Product.Name
	metaData := subscription.Plan.Product.Metadata

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
	productName, ok := metaData["productName"]
	if !ok {
		logger.Warn("Product name not found", zap.Any("subscriptionID", subscription.ID))
		return SubscriptionOverview{}, errors.New("Product name not found")
	}
	subscriptionOverview.ProductName = productName

	subscriptionOverview.UserCount = subscription.Quantity
	return subscriptionOverview, nil
}
