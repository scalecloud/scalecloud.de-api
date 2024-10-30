package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/subscription"
	"go.uber.org/zap"
)

func (stripeConnection *StripeConnection) GetSubscriptionByID(c context.Context, subscriptionID string) (*stripe.Subscription, error) {
	stripe.Key = stripeConnection.Key
	return subscription.Get(subscriptionID, nil)
}

func (paymentHandler *PaymentHandler) ResumeSubscription(c context.Context, tokenDetails firebasemanager.TokenDetails, request SubscriptionResumeRequest) (SubscriptionResumeReply, error) {
	err := paymentHandler.MongoConnection.HasPermission(c, tokenDetails, request.SubscriptionID, []mongomanager.Role{mongomanager.RoleAdministrator})
	if err != nil {
		return SubscriptionResumeReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key
	sub, error := paymentHandler.StripeConnection.GetSubscriptionByID(c, request.SubscriptionID)
	if error != nil {
		return SubscriptionResumeReply{}, errors.New("subscription not found")
	}
	if !sub.CancelAtPeriodEnd {
		return SubscriptionResumeReply{}, errors.New("subscription is not canceled")
	}
	subscriptionParams := &stripe.SubscriptionParams{CancelAtPeriodEnd: stripe.Bool(false)}
	result, err := subscription.Update(request.SubscriptionID, subscriptionParams)
	if err != nil {
		return SubscriptionResumeReply{}, err
	}
	reply := SubscriptionResumeReply{
		SubscriptionID:    result.ID,
		CancelAtPeriodEnd: &result.CancelAtPeriodEnd,
	}
	return reply, nil
}

func (paymentHandler *PaymentHandler) CancelSubscription(c context.Context, tokenDetails firebasemanager.TokenDetails, request SubscriptionCancelRequest) (SubscriptionCancelReply, error) {
	err := paymentHandler.MongoConnection.HasPermission(c, tokenDetails, request.SubscriptionID, []mongomanager.Role{mongomanager.RoleAdministrator})
	if err != nil {
		return SubscriptionCancelReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key
	sub, error := paymentHandler.StripeConnection.GetSubscriptionByID(c, request.SubscriptionID)
	if error != nil {
		return SubscriptionCancelReply{}, errors.New("subscription not found")
	}
	if sub.CancelAtPeriodEnd {
		paymentHandler.Log.Info("Subscription is already canceled", zap.String("status", string(sub.Status)))
		return SubscriptionCancelReply{}, errors.New("subscription is already canceled")
	}
	subscriptionParams := &stripe.SubscriptionParams{CancelAtPeriodEnd: stripe.Bool(true)}
	result, err := subscription.Update(request.SubscriptionID, subscriptionParams)
	if err != nil {
		return SubscriptionCancelReply{}, err
	}
	reply := SubscriptionCancelReply{
		SubscriptionID:    result.ID,
		CancelAtPeriodEnd: &result.CancelAtPeriodEnd,
		CancelAt:          result.CancelAt,
	}
	return reply, nil
}
