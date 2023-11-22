package stripemanager

import (
	"context"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentmethod"
)

func (stripeConnection *StripeConnection) GetPaymentMethod(c context.Context, paymentMethodID string) (*stripe.PaymentMethod, error) {
	stripe.Key = stripeConnection.Key
	pm, err := paymentmethod.Get(
		paymentMethodID,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return pm, nil
}
