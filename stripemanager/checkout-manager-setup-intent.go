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

func CreateCheckoutSetupIntent(c context.Context, token string, checkoutSetupIntentRequest CheckoutSetupIntentRequest) (CheckoutSetupIntentReply, error) {
	tokenDetails, err := firebasemanager.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return CheckoutSetupIntentReply{}, err
	}
	filter := mongomanager.User{
		UID: tokenDetails.UID,
	}
	customerID, err := searchOrCreateCustomer(c, filter, tokenDetails)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return CheckoutSetupIntentReply{}, err
	}
	if customerID == "" {
		logger.Error("Customer ID is empty")
		return CheckoutSetupIntentReply{}, errors.New("Customer ID is empty")
	}
	stripe.Key = getStripeKey()

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
