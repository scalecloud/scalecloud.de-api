package stripe

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/sub"
	"go.uber.org/zap"
)

func getSubscriptionByIDTest() {
	stripe.Key = getStripeKey()
	subscriptionID := "sub_1L21VfA86yrbtIQrsM51apgo"

	subscription, error := sub.Get(subscriptionID, nil)
	if error != nil {
		logger.Error("Error getting subscription item", zap.Error(error))
	}
	logger.Info("subscription", zap.Any("subscription", subscription.Created))
}

func GetSubscriptionByIDTwo(c context.Context, customerID, subscriptionID string) (subscriptionDetail SubscriptionDetail, err error) {
	stripe.Key = getStripeKey()
	//	subscriptionID := "sub_INYwS5uFiirGNs"
	subscription, err := sub.Get(subscriptionID, nil)
	if err != nil {
		logger.Error("Error getting subscription", zap.Error(err))
		return SubscriptionDetail{}, errors.New("Subscription not found")
	}
	logger.Info("Subscription", zap.Any("subscription", subscription))
	if subscription.Customer.ID == customerID {
		//	subscriptionDetail, err := mapSubscriptionItemToSubscriptionDetail(subscription)
		if err != nil {
			logger.Error("Error mapping subscription to subscription detail", zap.Error(err))
			return SubscriptionDetail{}, errors.New("Subscription not found")
		}
		return subscriptionDetail, nil
	} else {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", subscriptionID))
		return SubscriptionDetail{}, errors.New("Subscription not matching customer")
	}
}
