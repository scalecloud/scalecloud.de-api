package stripechangepayment

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/stripecustomer"
	"github.com/scalecloud/scalecloud.de-api/stripesecret"
	"github.com/scalecloud/scalecloud.de-api/stripesubscription"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/setupintent"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

type SetupIntentMeta string

const SetupIntentMetaKey SetupIntentMeta = "setupIntentMeta"

const (
	CreateSubscription SetupIntentMeta = "createSubscription"
	ChangePayment      SetupIntentMeta = "changePayment"
)

func GetChangePaymentSetupIntent(c context.Context, token string, request ChangePaymentRequest) (ChangePaymentReply, error) {
	if request.SubscriptionID == "" {
		return ChangePaymentReply{}, errors.New("Subscription ID is empty")
	}
	tokenDetails, err := firebasemanager.GetTokenDetails(c, token)
	if err != nil {

		logger.Error("Error getting token details", zap.Error(err))
		return ChangePaymentReply{}, err
	}
	customerID, err := stripecustomer.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return ChangePaymentReply{}, err
	}
	stripe.Key = stripesecret.GetStripeKey()
	subscription, error := stripesubscription.GetSubscriptionByID(c, request.SubscriptionID)
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
