package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/customer"
)

func getCustomerByID(ctx context.Context, customerID string) (customerDetails *stripe.Customer, err error) {
	customer, error := customer.Get(
		customerID,
		nil,
	)
	if error != nil {
		return &stripe.Customer{}, errors.New("Customer not found")
	}
	return customer, nil
}

func (paymentHandler *PaymentHandler) GetCustomerIDByUID(ctx context.Context, uid string) (string, error) {
	filter := mongomanager.User{
		UID: uid,
	}
	userSearch, err := paymentHandler.MongoConnection.GetUser(ctx, filter)
	if err != nil {
		return "", err
	}
	customerID := userSearch.CustomerID
	if customerID == "" {
		return "", errors.New("Customer ID is empty")
	}
	return customerID, nil
}

func (stripeConnection *StripeConnection) CreateCustomer(ctx context.Context, email string) (*stripe.Customer, error) {
	stripe.Key = stripeConnection.Key
	if email == "" {
		return nil, errors.New("E-Mail is required")
	}
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
	}
	newCustomer, err := customer.New(params)
	if err != nil {
		return nil, err
	}
	return newCustomer, nil
}
