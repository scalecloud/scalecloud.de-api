package stripemanager

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

func (paymentHandler *PaymentHandler) searchOrCreateCustomer(c context.Context, eMail, uid string) (string, error) {
	customerID, err := paymentHandler.getOrCreateCustomer(c, eMail, uid)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			paymentHandler.Log.Error("Unable to resolve Stripe customer mapping", zap.Error(err))
		} else {
			paymentHandler.Log.Error("Unable to get or create Stripe customer", zap.Error(err))
		}
		return "", err
	}
	return customerID, nil
}
