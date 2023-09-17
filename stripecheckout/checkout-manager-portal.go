package stripecheckout

import (
	"context"
	"errors"
	"strconv"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/scalecloud/scalecloud.de-api/stripeprice"
	"github.com/scalecloud/scalecloud.de-api/stripesecret"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/checkout/session"
	"go.uber.org/zap"
)

func CreateCheckoutSession(c context.Context, token string, checkoutModelPortalRequest CheckoutModelPortalRequest) (CheckoutModelPortalReply, error) {
	tokenDetails, err := firebasemanager.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return CheckoutModelPortalReply{}, err
	}
	filter := mongomanager.User{
		UID: tokenDetails.UID,
	}
	customerID, err := searchOrCreateCustomer(c, filter, tokenDetails)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return CheckoutModelPortalReply{}, err
	}
	if customerID == "" {
		logger.Error("Customer ID is empty")
		return CheckoutModelPortalReply{}, errors.New("Customer ID is empty")
	}
	stripe.Key = stripesecret.GetStripeKey()

	price, err := stripeprice.GetPrice(c, checkoutModelPortalRequest.ProductID)
	if err != nil {
		logger.Error("Error getting price", zap.Error(err))
		return CheckoutModelPortalReply{}, err
	}
	metaData := price.Metadata
	if err != nil {
		logger.Warn("Error getting price metadata", zap.Error(err))
		return CheckoutModelPortalReply{}, errors.New("Price metadata not found")
	}
	trialPeriodDays, ok := metaData["trialPeriodDays"]
	if !ok {
		logger.Warn("trialPeriodDays not found", zap.Any("priceID", price.ID))
		return CheckoutModelPortalReply{}, errors.New("trialPeriodDays not found")
	}
	iTrialPeriodDays, err := strconv.ParseInt(trialPeriodDays, 10, 64)
	if err != nil {
		logger.Warn("Error converting trialPeriodDays to int", zap.Error(err))
		return CheckoutModelPortalReply{}, errors.New("Error converting trialPeriodDays to int")
	}

	domain := "https://scalecloud.de/checkout"
	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(price.ID),
				Quantity: stripe.Int64(checkoutModelPortalRequest.Quantity),
				AdjustableQuantity: &stripe.CheckoutSessionLineItemAdjustableQuantityParams{
					Enabled: stripe.Bool(true),
				},
			},
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			TrialPeriodDays: stripe.Int64(iTrialPeriodDays),
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(domain + "/success.html"),
		CancelURL:  stripe.String(domain + "/cancel.html"),
		Customer:   stripe.String(customerID),
	}
	session, err := session.New(params)
	if err != nil {
		logger.Error("Error creating session", zap.Error(err))
		return CheckoutModelPortalReply{}, err
	}

	checkoutModel := CheckoutModelPortalReply{
		URL: session.URL,
	}
	return checkoutModel, nil
}