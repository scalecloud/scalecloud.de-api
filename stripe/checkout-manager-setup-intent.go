package stripe

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebase"
	"github.com/scalecloud/scalecloud.de-api/mongo"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/setupintent"
	"go.uber.org/zap"
)

func CreateCheckoutSetupIntent(c context.Context, token string, checkoutSetupIntentRequest CheckoutSetupIntentRequest) (CheckoutSetupIntentReply, error) {
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return CheckoutSetupIntentReply{}, err
	}
	filter := mongo.User{
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
