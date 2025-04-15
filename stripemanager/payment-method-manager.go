package stripemanager

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentmethod"
)

var ErrDefaultPaymentMethodNotFound = errors.New("DefaultPaymentMethod not found")

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

func (stripeConnection *StripeConnection) GetDefaultPaymentMethod(c context.Context, cus *stripe.Customer) (*stripe.PaymentMethod, error) {
	if cus.InvoiceSettings == nil {
		return nil, errors.New("InvoiceSettings not found")
	}
	defaultPaymentMethod := cus.InvoiceSettings.DefaultPaymentMethod
	if defaultPaymentMethod == nil {
		return nil, ErrDefaultPaymentMethodNotFound
	}
	defaultPaymentID := defaultPaymentMethod.ID
	if defaultPaymentID == "" {
		return nil, errors.New("DefaultPaymentMethodID is empty")
	}
	return stripeConnection.GetPaymentMethod(c, defaultPaymentID)
}
