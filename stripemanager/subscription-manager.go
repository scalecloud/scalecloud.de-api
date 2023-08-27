package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/subscription"
	"go.uber.org/zap"
)

func getSubscriptionByID(c context.Context, subscriptionID string) (*stripe.Subscription, error) {
	stripe.Key = getStripeKey()
	return subscription.Get(subscriptionID, nil)
}

func ResumeSubscription(c context.Context, token string, request SubscriptionResumeRequest) (SubscriptionResumeReply, error) {
	if request.ID == "" {
		return SubscriptionResumeReply{}, errors.New("Subscription ID is empty")
	}
	tokenDetails, err := firebasemanager.GetTokenDetails(c, token)
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
	sub, error := getSubscriptionByID(c, request.ID)
	if error != nil {
		logger.Warn("Error getting subscription", zap.Error(error))
		return SubscriptionResumeReply{}, errors.New("Subscription not found")
	}
	if !sub.CancelAtPeriodEnd {
		logger.Info("Subscription is not canceled", zap.String("status", string(sub.Status)))
		return SubscriptionResumeReply{}, errors.New("Subscription is not canceled")
	}
	if sub.Customer.ID != customerID {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.ID))
		return SubscriptionResumeReply{}, errors.New("Subscription not matching customer")
	} else {
		subscriptionParams := &stripe.SubscriptionParams{CancelAtPeriodEnd: stripe.Bool(false)}
		result, err := subscription.Update(request.ID, subscriptionParams)
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
	tokenDetails, err := firebasemanager.GetTokenDetails(c, token)
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
	sub, error := getSubscriptionByID(c, request.ID)
	if error != nil {
		logger.Warn("Error getting subscription", zap.Error(error))
		return SubscriptionCancelReply{}, errors.New("Subscription not found")
	}
	if sub.CancelAtPeriodEnd {
		logger.Info("Subscription is already canceled", zap.String("status", string(sub.Status)))
		return SubscriptionCancelReply{}, errors.New("Subscription is already canceled")
	}
	if sub.Customer.ID != customerID {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.ID))
		return SubscriptionCancelReply{}, errors.New("Subscription not matching customer")
	} else {
		subscriptionParams := &stripe.SubscriptionParams{CancelAtPeriodEnd: stripe.Bool(true)}
		result, err := subscription.Update(request.ID, subscriptionParams)
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
