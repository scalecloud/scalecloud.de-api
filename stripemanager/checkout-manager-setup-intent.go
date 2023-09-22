package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/setupintent"
	"go.uber.org/zap"
)

func (stripeConnection *StripeConnection) CreateCheckoutSetupIntent(c context.Context, tokenDetails firebasemanager.TokenDetails, checkoutSetupIntentRequest CheckoutSetupIntentRequest) (CheckoutSetupIntentReply, error) {
	filter := mongomanager.User{
		UID: tokenDetails.UID,
	}
	customerID, err := stripeConnection.searchOrCreateCustomer(c, filter, tokenDetails)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return CheckoutSetupIntentReply{}, err
	}
	if customerID == "" {
		logger.Error("Customer ID is empty")
		return CheckoutSetupIntentReply{}, errors.New("Customer ID is empty")
	}
	stripe.Key = stripeConnection.Key

	setupIntentParam := &stripe.SetupIntentParams{
		Customer: stripe.String(customerID),
	}
	setupIntent, err := setupintent.New(setupIntentParam)
	if err != nil {
		logger.Error("Error creating setup intent", zap.Error(err))
		return CheckoutSetupIntentReply{}, err
	}

	checkoutSetupIntentReplyModel := CheckoutSetupIntentReply{
		SetupIntentID: setupIntent.ID,
		ClientSecret:  setupIntent.ClientSecret,
	}
	return checkoutSetupIntentReplyModel, nil

}
