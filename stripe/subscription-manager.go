package stripe

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebase"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/sub"
	"go.uber.org/zap"
)

func getSubscriptionByID(c context.Context, subscriptionID string) (*stripe.Subscription, error) {
	stripe.Key = getStripeKey()
	return sub.Get(subscriptionID, nil)
}

func ResumeSubscription(c context.Context, token string, request SubscriptionResumeRequest) (SubscriptionResumeReply, error) {
	if request.ID == "" {
		return SubscriptionResumeReply{}, errors.New("Subscription ID is empty")
	}
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return SubscriptionResumeReply{}, err
	}
	customerID, err := getCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return SubscriptionResumeReply{}, err
	}
	stripe.Key = getStripeKey()
	subscription, error := getSubscriptionByID(c, request.ID)
	if error != nil {
		logger.Warn("Error getting subscription", zap.Error(error))
		return SubscriptionResumeReply{}, errors.New("Subscription not found")
	}
	logger.Debug("subscription", zap.Any("subscription", subscription))
	if subscription.Customer.ID != customerID {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.ID))
		return SubscriptionResumeReply{}, errors.New("Subscription not matching customer")
	} else {
		subscriptionParams := &stripe.SubscriptionParams{CancelAtPeriodEnd: stripe.Bool(false)}
		result, err := sub.Update(request.ID, subscriptionParams)
		if err != nil {
			logger.Error("Error updating subscription", zap.Error(err))
			return SubscriptionResumeReply{}, err
		}

		reply := SubscriptionResumeReply{
			ID:                result.ID,
			CancelAtPeriodEnd: result.CancelAtPeriodEnd,
		}
		return reply, nil
	}

}

func CancelSubscription(c context.Context, token string, request SubscriptionCancelRequest) (SubscriptionCancelReply, error) {
	if request.ID == "" {
		return SubscriptionCancelReply{}, errors.New("Subscription ID is empty")
	}
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return SubscriptionCancelReply{}, err
	}
	customerID, err := getCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return SubscriptionCancelReply{}, err
	}
	stripe.Key = getStripeKey()
	subscription, error := getSubscriptionByID(c, request.ID)
	if error != nil {
		logger.Warn("Error getting subscription", zap.Error(error))
		return SubscriptionCancelReply{}, errors.New("Subscription not found")
	}
	logger.Debug("subscription", zap.Any("subscription", subscription))
	if subscription.Customer.ID != customerID {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.ID))
		return SubscriptionCancelReply{}, errors.New("Subscription not matching customer")
	} else {
		subscriptionParams := &stripe.SubscriptionParams{CancelAtPeriodEnd: stripe.Bool(true)}
		result, err := sub.Update(request.ID, subscriptionParams)
		if err != nil {
			logger.Error("Error updating subscription", zap.Error(err))
			return SubscriptionCancelReply{}, err
		}

		reply := SubscriptionCancelReply{
			ID:                result.ID,
			CancelAtPeriodEnd: result.CancelAtPeriodEnd,
			CancelAt:          result.CancelAt,
		}
		return reply, nil
	}

}
