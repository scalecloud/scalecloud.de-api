package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/setupintent"
)

func (paymentHandler *PaymentHandler) CreateCheckoutSetupIntent(c context.Context, tokenDetails firebasemanager.TokenDetails, checkoutSetupIntentRequest CheckoutSetupIntentRequest) (CheckoutSetupIntentReply, error) {
	filter := mongomanager.User{
		UID: tokenDetails.UID,
	}
	customerID, err := paymentHandler.searchOrCreateCustomer(c, filter, tokenDetails)
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
