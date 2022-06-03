package stripe

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebase"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"go.uber.org/zap"
)

func GetCheckout(c context.Context, tokenDetails firebase.TokenDetails) (CheckoutModel, error) {
	if tokenDetails.UID == "" {
		logger.Error("Customer ID is empty")
		return CheckoutModel{}, errors.New("Customer ID is empty")
	}
	if tokenDetails.Email == "" {
		logger.Error("Email is empty")
		return CheckoutModel{}, errors.New("Email is empty")
	}

	domain := "https://scalecloud.de/checkout"
	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				// Provide the exact Price ID (for example, pr_1234) of the product you want to sell
				Price:    stripe.String("price_1Gv4wsA86yrbtIQrnW9dilsQ"),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(domain + "/success.html"),
		CancelURL:  stripe.String(domain + "/cancel.html"),
		Customer:   stripe.String("cus_IJNox8VXgkX2gU"),
	}

	session, err := session.New(params)
	if err != nil {
		logger.Error("Error creating session", zap.Error(err))
		return CheckoutModel{}, err
	}

	checkoutModel := CheckoutModel{
		URL: session.URL,
	}
	return checkoutModel, nil
}
