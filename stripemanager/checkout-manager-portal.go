package stripemanager

import (
	"context"
	"errors"
	"strconv"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/checkout/session"
	"go.uber.org/zap"
)

func (stripeConnection *StripeConnection) CreateCheckoutSession(c context.Context, tokenDetails firebasemanager.TokenDetails, checkoutModelPortalRequest CheckoutModelPortalRequest) (CheckoutModelPortalReply, error) {
	filter := mongomanager.User{
		UID: tokenDetails.UID,
	}
	customerID, err := stripeConnection.searchOrCreateCustomer(c, filter, tokenDetails)
	if err != nil {
		return CheckoutModelPortalReply{}, err
	}
	if customerID == "" {
		return CheckoutModelPortalReply{}, errors.New("Customer ID is empty")
	}
	stripe.Key = stripeConnection.Key

	price, err := stripeConnection.GetPrice(c, checkoutModelPortalRequest.ProductID)
	if err != nil {
		return CheckoutModelPortalReply{}, err
	}
	metaData := price.Metadata
	if metaData == nil {
		return CheckoutModelPortalReply{}, errors.New("Price metadata not found")
	}
	trialPeriodDays, ok := metaData["trialPeriodDays"]
	if !ok {
		return CheckoutModelPortalReply{}, errors.New("trialPeriodDays not found for priceID: " + price.ID)
	}
	iTrialPeriodDays, err := strconv.ParseInt(trialPeriodDays, 10, 64)
	if err != nil {
		stripeConnection.Log.Warn("Error converting trialPeriodDays to int", zap.Error(err))
		return CheckoutModelPortalReply{}, errors.New("Error converting trialPeriodDays")
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
		return CheckoutModelPortalReply{}, err
	}

	checkoutModel := CheckoutModelPortalReply{
		URL: session.URL,
	}
	return checkoutModel, nil
}
