package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/customer"
)

func (paymentHandler *PaymentHandler) GetBillingAddress(c context.Context, tokenDetails firebasemanager.TokenDetails, request BillingAddressRequest) (BillingAddressReply, error) {
	err := paymentHandler.MongoConnection.HasPermission(c, tokenDetails, request.SubscriptionID, []mongomanager.Role{mongomanager.RoleBilling})
	if err != nil {
		return BillingAddressReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key

	subscription, err := paymentHandler.StripeConnection.GetSubscriptionByID(c, request.SubscriptionID)
	if err != nil {
		return BillingAddressReply{}, err
	}

	if subscription.Customer == nil {
		return BillingAddressReply{}, errors.New("subscription does not have a customer")
	}
	if subscription.Customer.ID == "" {
		return BillingAddressReply{}, errors.New("subscription customer ID is empty")
	}
	customer, err := GetCustomerByID(c, subscription.Customer.ID)
	if err != nil {
		return BillingAddressReply{}, err
	}

	address := Address{
		City:       customer.Address.City,
		Country:    customer.Address.Country,
		Line1:      customer.Address.Line1,
		Line2:      &customer.Address.Line2,
		PostalCode: customer.Address.PostalCode,
	}

	reply := BillingAddressReply{
		SubscriptionID: request.SubscriptionID,
		Name:           customer.Name,
		Address:        address,
		Phone:          customer.Phone,
	}

	return reply, nil

}

func (paymentHandler *PaymentHandler) UpdateBillingAddress(c context.Context, tokenDetails firebasemanager.TokenDetails, request UpdateBillingAddressRequest) (UpdateBillingAddressReply, error) {
	err := paymentHandler.MongoConnection.HasPermission(c, tokenDetails, request.SubscriptionID, []mongomanager.Role{mongomanager.RoleBilling})
	if err != nil {
		return UpdateBillingAddressReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key

	subscription, err := paymentHandler.StripeConnection.GetSubscriptionByID(c, request.SubscriptionID)
	if err != nil {
		return UpdateBillingAddressReply{}, err
	}

	if subscription.Customer == nil {
		return UpdateBillingAddressReply{}, errors.New("subscription does not have a customer")
	}
	if subscription.Customer.ID == "" {
		return UpdateBillingAddressReply{}, errors.New("subscription customer ID is empty")
	}

	params := &stripe.CustomerParams{
		Name: stripe.String(request.Name),
		Address: &stripe.AddressParams{
			City:       stripe.String(request.Address.City),
			Country:    stripe.String(request.Address.Country),
			Line1:      stripe.String(request.Address.Line1),
			Line2:      stripe.String(*request.Address.Line2),
			PostalCode: stripe.String(request.Address.PostalCode),
		},
		Phone: stripe.String(request.Phone),
	}

	_, err = customer.Update(subscription.Customer.ID, params)
	if err != nil {
		return UpdateBillingAddressReply{}, err
	}

	reply := UpdateBillingAddressReply{
		SubscriptionID: request.SubscriptionID,
	}

	return reply, nil

}
