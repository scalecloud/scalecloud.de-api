package stripe

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go/v72"
	"go.uber.org/zap"
)

func checkParamsValid(customerID, subscriptionID string) (err error) {
	if customerID == "" {
		logger.Error("Customer ID is empty")
		return errors.New("Customer ID is empty")
	} else if subscriptionID == "" {
		logger.Error("Subscription ID is empty")
		return errors.New("SubscriptionID is empty")
	} else {
		return nil
	}
}

func GetSubscriptionByID(c context.Context, customerID, subscriptionID string) (subscriptionDetail SubscriptionDetail, err error) {
	err = checkParamsValid(customerID, subscriptionID)
	if err != nil {
		return subscriptionDetail, err
	}
	stripe.Key = getStripeKey()
	subscription, error := getSubscriptionByID(c, subscriptionID)
	if error != nil {
		logger.Warn("Error getting subscription", zap.Error(error))
		return SubscriptionDetail{}, errors.New("Subscription not found")
	}
	logger.Debug("subscription", zap.Any("subscription", subscription))
	if subscription.Customer.ID != customerID {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", subscriptionID))
		return SubscriptionDetail{}, errors.New("Subscription not matching customer")
	} else {
		subscriptionDetail, err = mapSubscriptionItemToSubscriptionDetail(subscription)
		if err != nil {
			logger.Error("Error mapping subscription to subscription detail", zap.Error(err))
			return SubscriptionDetail{}, errors.New("Subscription not found")
		}
	}
	return subscriptionDetail, nil
}

func mapSubscriptionItemToSubscriptionDetail(subscription *stripe.Subscription) (subscriptionDetail SubscriptionDetail, err error) {
	subscriptionDetail.ID = subscription.ID
	subscriptionDetail.PlanProductName = subscription.Plan.Product.Name
	subscriptionDetail.SubscriptionArticelID = subscription.Items.Data[0].ID
	//	subscriptionDetail.PricePerMonth = subscription.Items.Data[0].Plan.Amount / 100
	//	subscriptionDetail.Started = subscription.Start
	//	subscriptionDetail.EndsOn = subscription.End
	return subscriptionDetail, nil
}
