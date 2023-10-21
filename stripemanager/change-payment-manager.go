package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/setupintent"
	"go.uber.org/zap"
)

type SetupIntentMeta string

const SetupIntentMetaKey SetupIntentMeta = "setupIntentMeta"

const (
	CreateSubscription SetupIntentMeta = "createSubscription"
	ChangePayment      SetupIntentMeta = "changePayment"
)

func (paymentHandler *PaymentHandler) GetChangePaymentSetupIntent(c context.Context, tokenDetails firebasemanager.TokenDetails, request ChangePaymentRequest) (ChangePaymentReply, error) {
	if request.SubscriptionID == "" {
		return ChangePaymentReply{}, errors.New("Subscription ID is empty")
	}
	customerID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return ChangePaymentReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key
	subscription, err := paymentHandler.StripeConnection.GetSubscriptionByID(c, request.SubscriptionID)
	if err != nil {
		return ChangePaymentReply{}, err
	}
	if subscription.Customer.ID != customerID {
		paymentHandler.Log.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.SubscriptionID))
		return ChangePaymentReply{}, errors.New("Subscription not matching customer")
	}

	params := &stripe.SetupIntentParams{
		Customer: stripe.String(customerID),
	}

	params.AddMetadata(string(SetupIntentMetaKey), string(ChangePayment))

	si, err := setupintent.New(params)
	if err != nil {
		return ChangePaymentReply{}, err
	}

	reply := ChangePaymentReply{
		SetupIntentID: si.ID,
		ClientSecret:  si.ClientSecret,
		EMail:         tokenDetails.EMail,
	}
	return reply, nil
}
