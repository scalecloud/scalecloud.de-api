package stripe

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"go.uber.org/zap"
)

func getCustomerByID(c context.Context, customerID string) (customerDetails *stripe.Customer, err error) {
	stripe.Key = getStripeKey()
	//customerID := "cus_IJNox8VXgkX2gU"
	customer, error := customer.Get(
		customerID,
		nil,
	)
	if error != nil {
		logger.Error("Error getting customer", zap.Error(error))
		return &stripe.Customer{}, errors.New("Customer not found")
	}
	logger.Debug("Customer", zap.Any("customer", customer))
	return customer, nil
}
