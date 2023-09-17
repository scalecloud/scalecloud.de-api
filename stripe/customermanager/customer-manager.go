package customermanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/scalecloud/scalecloud.de-api/stripe/secret"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/customer"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func getCustomerByID(ctx context.Context, customerID string) (customerDetails *stripe.Customer, err error) {
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

func GetCustomerIDByUID(ctx context.Context, uid string) (string, error) {
	filter := mongomanager.User{
		UID: uid,
	}
	userSearch, err := mongomanager.GetUser(ctx, filter)
	if err != nil {
		logger.Error("Error getting user", zap.Error(err))
		return "", err
	}
	customerID := userSearch.CustomerID
	if customerID == "" {
		logger.Error("Customer ID is empty")
		return "", errors.New("Customer ID is empty")
	}
	return customerID, nil
}

func CreateCustomer(ctx context.Context, email string) (*stripe.Customer, error) {
	stripe.Key = secret.GetStripeKey()
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
