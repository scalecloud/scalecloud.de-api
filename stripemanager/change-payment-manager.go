package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/paymentmethod"
	"github.com/stripe/stripe-go/v81/setupintent"
	"go.uber.org/zap"
)

type SetupIntentMeta string

const SetupIntentMetaKey SetupIntentMeta = "setupIntentMeta"

const (
	CreateSubscription SetupIntentMeta = "createSubscription"
	ChangePayment      SetupIntentMeta = "changePayment"
)

func (paymentHandler *PaymentHandler) GetChangePaymentSetupIntent(c context.Context, tokenDetails firebasemanager.TokenDetails) (ChangePaymentReply, error) {
	customerID, err := paymentHandler.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return ChangePaymentReply{}, err
	}
	stripe.Key = paymentHandler.StripeConnection.Key

	params := &stripe.SetupIntentParams{
		Customer: stripe.String(customerID),
	}

	params.AddMetadata(string(SetupIntentMetaKey), string(ChangePayment))

	si, err := setupintent.New(params)
	if err != nil {
		return ChangePaymentReply{}, err
	}

	reply := ChangePaymentReply{
		SetupIntentID: si.ID,
		ClientSecret:  si.ClientSecret,
		EMail:         tokenDetails.EMail,
	}
	return reply, nil
}

func (paymentHandler *PaymentHandler) ChangePaymentDefault(c context.Context, setupIntent stripe.SetupIntent) error {
	stripe.Key = paymentHandler.StripeConnection.Key
	cus := setupIntent.Customer
	if cus == nil {
		return errors.New("customer not set")
	}
	if cus.ID == "" {
		return errors.New("customer ID not set")
	}
	params := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(setupIntent.PaymentMethod.ID),
		},
	}
	result, err := customer.Update(cus.ID, params)
	if err != nil {
		return err
	}
	paymentHandler.Log.Info("Customer updated", zap.Any("Customer", result.ID))
	err = paymentHandler.detachPaymentMethodsButDefault(setupIntent)
	if err != nil {
		return err
	}
	return nil
}

func (paymentHandler *PaymentHandler) detachPaymentMethodsButDefault(setupIntent stripe.SetupIntent) error {
	stripe.Key = paymentHandler.StripeConnection.Key
	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(setupIntent.Customer.ID),
	}
	i := paymentmethod.List(params)
	for i.Next() {
		pm := i.PaymentMethod()
		if pm.ID != setupIntent.PaymentMethod.ID {
			pmDetached, err := paymentmethod.Detach(
				pm.ID,
				nil,
			)
			if err != nil {
				return err
			}
			paymentHandler.Log.Info("PaymentMethod detached", zap.Any("PaymentMethod", pmDetached.ID))
		}
	}
	return nil
}

func (paymentHandler *PaymentHandler) ChangeCustomerAddress(c context.Context, setupIntent stripe.SetupIntent) error {
	stripe.Key = paymentHandler.StripeConnection.Key
	cus := setupIntent.Customer
	if cus == nil {
		return errors.New("customer not set")
	}
	if cus.ID == "" {
		return errors.New("customer ID not set")
	}
	paymentMethod, err := paymentHandler.StripeConnection.GetPaymentMethod(c, setupIntent.PaymentMethod.ID)
	if err != nil {
		return errors.New("payment method not found")
	}
	address := paymentMethod.BillingDetails.Address
	if address == nil {
		return errors.New("billing address not set")
	}
	if address.Line1 == "" {
		return errors.New("billing address line1 not set")
	}
	params := &stripe.CustomerParams{
		Name:  stripe.String(paymentMethod.BillingDetails.Name),
		Phone: stripe.String(paymentMethod.BillingDetails.Phone),
		Address: &stripe.AddressParams{
			Line1:      stripe.String(address.Line1),
			Line2:      stripe.String(address.Line2),
			City:       stripe.String(address.City),
			PostalCode: stripe.String(address.PostalCode),
			Country:    stripe.String(address.Country),
		},
	}
	updatedCustomer, err := customer.Update(cus.ID, params)
	if err != nil {
		return err
	}
	paymentHandler.Log.Info("Customer address updated", zap.Any("Customer", updatedCustomer.ID))
	return nil
}
