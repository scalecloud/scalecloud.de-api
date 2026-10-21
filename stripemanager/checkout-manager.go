package stripemanager

import (
	"context"

	"go.uber.org/zap"
)

func (paymentHandler *PaymentHandler) searchOrCreateCustomer(c context.Context, eMail, uid string) (string, error) {
	customerID, err := paymentHandler.getOrCreateCustomer(c, eMail, uid)
	if err != nil {
		paymentHandler.Log.Error("Unable to get or create Stripe customer", zap.Error(err))
		return "", err
	}
	return customerID, nil
}
