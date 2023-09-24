package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/subscription"
	"go.uber.org/zap"
)

func (stripeConnection *StripeConnection) GetSubscriptionByID(c context.Context, subscriptionID string) (*stripe.Subscription, error) {
	stripe.Key = stripeConnection.Key
	return subscription.Get(subscriptionID, nil)
}

func (paymentHandler *PaymentHandler) ResumeSubscription(c context.Context, tokenDetails firebasemanager.TokenDetails, request SubscriptionResumeRequest) (SubscriptionResumeReply, error) {
	if request.ID == "" {
		return SubscriptionResumeReply{}, errors.New("Subscription ID is empty")
	}
	customerID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return SubscriptionResumeReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key
	sub, error := paymentHandler.StripeConnection.GetSubscriptionByID(c, request.ID)
	if error != nil {
		return SubscriptionResumeReply{}, errors.New("Subscription not found")
	}
	if !sub.CancelAtPeriodEnd {
		return SubscriptionResumeReply{}, errors.New("Subscription is not canceled")
	}
	if sub.Customer.ID != customerID {
		paymentHandler.Log.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.ID))
		return SubscriptionResumeReply{}, errors.New("Subscription not matching customer")
	} else {
		subscriptionParams := &stripe.SubscriptionParams{CancelAtPeriodEnd: stripe.Bool(false)}
		result, err := subscription.Update(request.ID, subscriptionParams)
		if err != nil {
			return SubscriptionResumeReply{}, err
		}

		reply := SubscriptionResumeReply{
			ID:                result.ID,
			CancelAtPeriodEnd: &result.CancelAtPeriodEnd,
		}
		return reply, nil
	}
}

func (paymentHandler *PaymentHandler) CancelSubscription(c context.Context, tokenDetails firebasemanager.TokenDetails, request SubscriptionCancelRequest) (SubscriptionCancelReply, error) {
	if request.ID == "" {
		return SubscriptionCancelReply{}, errors.New("Subscription ID is empty")
	}
	customerID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return SubscriptionCancelReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key
	sub, error := paymentHandler.StripeConnection.GetSubscriptionByID(c, request.ID)
	if error != nil {
		return SubscriptionCancelReply{}, errors.New("Subscription not found")
	}
	if sub.CancelAtPeriodEnd {
		paymentHandler.Log.Info("Subscription is already canceled", zap.String("status", string(sub.Status)))
		return SubscriptionCancelReply{}, errors.New("Subscription is already canceled")
	}
	if sub.Customer.ID != customerID {
		paymentHandler.Log.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.ID))
		return SubscriptionCancelReply{}, errors.New("Subscription not matching customer")
	} else {
		subscriptionParams := &stripe.SubscriptionParams{CancelAtPeriodEnd: stripe.Bool(true)}
		result, err := subscription.Update(request.ID, subscriptionParams)
		if err != nil {
			return SubscriptionCancelReply{}, err
		}

		reply := SubscriptionCancelReply{
			ID:                result.ID,
			CancelAtPeriodEnd: &result.CancelAtPeriodEnd,
			CancelAt:          result.CancelAt,
		}
		return reply, nil
	}
}
