package stripe

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"go.uber.org/zap"
)

func getCustomerByID(ctx context.Context, customerID string) (customerDetails *stripe.Customer, err error) {
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

func CreateCustomer(ctx context.Context, email string) (*stripe.Customer, error) {
	if email == "" {
		return nil, errors.New("E-Mail is required")
	}
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
	}
	newCustomer, err := customer.New(params)
	if err != nil {
		logger.Error("Error creating customer", zap.Error(err))
		return nil, err
	}
	logger.Debug("Customer", zap.Any("customer", newCustomer))
	return newCustomer, nil
}
