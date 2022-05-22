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
		logger.Info("Subscription", zap.Any("subscription", subscription.Customer.ID))
		subscriptionDetail, err := mapSubscriptionToSubscriptionOverview(subscription)
		if err != nil {
			logger.Error("Error mapping subscription to subscription detail", zap.Error(err))
			return []SubscriptionOverview{}, errors.New("Subscription not found")
		}
		subscriptions = append(subscriptions, subscriptionDetail)
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
	metaData := subscription.Metadata
	if val, ok := metaData["storageAmount"]; ok {
		i, err := strconv.Atoi(val)
		if err != nil {
			logger.Error("Error converting subscriptionArticelID to int", zap.Error(err))
			return SubscriptionOverview{}, errors.New("SubscriptionArticelID not found")
		}
		subscriptionOverview.StorageAmount = i
	}
	subscriptionOverview.UserCount = subscription.Quantity
	return subscriptionOverview, nil
}

func mapSubscriptionItemToSubscriptionDetail(subscription stripe.Subscription, subscriptionItem stripe.SubscriptionItem) (subscriptionDetail SubscriptionDetail, err error) {
	subscriptionDetail.ID = subscriptionItem.ID
	subscriptionDetail.PlanProductName = subscription.Plan.Product.Name
	subscriptionDetail.SubscriptionArticelID = subscription.Items.Data[0].ID
	//	subscriptionDetail.PricePerMonth = subscription.Items.Data[0].Plan.Amount / 100
	//	subscriptionDetail.Started = subscription.Start
	//	subscriptionDetail.EndsOn = subscription.End
	return subscriptionDetail, nil
}
