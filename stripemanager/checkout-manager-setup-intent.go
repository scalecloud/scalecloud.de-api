package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/setupintent"
)

func (paymentHandler *PaymentHandler) CreateCheckoutSetupIntent(c context.Context, tokenDetails firebasemanager.TokenDetails, checkoutSetupIntentRequest CheckoutSetupIntentRequest) (CheckoutSetupIntentReply, error) {
	customerID, err := paymentHandler.searchOrCreateCustomer(c, tokenDetails)
	if err != nil {
		return CheckoutSetupIntentReply{}, err
	}
	if customerID == "" {
		return CheckoutSetupIntentReply{}, errors.New("customer ID is empty")
	}
	stripe.Key = paymentHandler.StripeConnection.Key

	setupIntentParam := &stripe.SetupIntentParams{
		Customer: stripe.String(customerID),
	}
	setupIntent, err := setupintent.New(setupIntentParam)
	if err != nil {
		return CheckoutSetupIntentReply{}, err
	}

	checkoutSetupIntentReplyModel := CheckoutSetupIntentReply{
		SetupIntentID: setupIntent.ID,
		ClientSecret:  setupIntent.ClientSecret,
	}
	return checkoutSetupIntentReplyModel, nil

}
