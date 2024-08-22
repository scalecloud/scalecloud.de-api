package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/customer"
	"go.mongodb.org/mongo-driver/mongo"
)

func (paymentHandler *PaymentHandler) GetCustomerByUID(ctx context.Context, uid string) (customerDetails *stripe.Customer, err error) {
	customerID, err := paymentHandler.GetCustomerIDByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key

	return GetCustomerByID(ctx, customerID)
}

func GetCustomerByID(ctx context.Context, customerID string) (customerDetails *stripe.Customer, err error) {
	customer, error := customer.Get(
		customerID,
		nil,
	)
	if error != nil {
		return &stripe.Customer{}, errors.New("customer not found")
	}
	return customer, nil
}

func (paymentHandler *PaymentHandler) existsCustomerByUID(ctx context.Context, uid string) (bool, error) {
	filter := mongomanager.User{
		UID: uid,
	}
	userSearch, err := paymentHandler.MongoConnection.GetUser(ctx, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}
	customerID := userSearch.CustomerID
	if customerID == "" {
		return false, errors.New("customer ID is empty")
	}
	return true, nil
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
		return "", errors.New("customer ID is empty")
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
