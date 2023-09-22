package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/setupintent"
	"go.uber.org/zap"
)

type SetupIntentMeta string

const SetupIntentMetaKey SetupIntentMeta = "setupIntentMeta"

const (
	CreateSubscription SetupIntentMeta = "createSubscription"
	ChangePayment      SetupIntentMeta = "changePayment"
)

func (stripeConnection *StripeConnection) GetChangePaymentSetupIntent(c context.Context, tokenDetails firebasemanager.TokenDetails, request ChangePaymentRequest) (ChangePaymentReply, error) {
	if request.SubscriptionID == "" {
		return ChangePaymentReply{}, errors.New("Subscription ID is empty")
	}
	customerID, err := GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return ChangePaymentReply{}, err
	}
	stripe.Key = stripeConnection.Key
	subscription, error := stripeConnection.GetSubscriptionByID(c, request.SubscriptionID)
	if error != nil {
		logger.Warn("Error getting subscription", zap.Error(error))
		return ChangePaymentReply{}, errors.New("Subscription not found")
	}
	if subscription.Customer.ID != customerID {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.SubscriptionID))
		return ChangePaymentReply{}, errors.New("Subscription not matching customer")
	}

	params := &stripe.SetupIntentParams{
		Customer: stripe.String(customerID),
	}

	params.AddMetadata(string(SetupIntentMetaKey), string(ChangePayment))

	si, err := setupintent.New(params)
	if err != nil {
		logger.Error("Error creating setup intent", zap.Error(err))
		return ChangePaymentReply{}, err
	}

	reply := ChangePaymentReply{
		SetupIntentID: si.ID,
		ClientSecret:  si.ClientSecret,
	}
	return reply, nil
}
