package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/paymentmethod"
	"github.com/stripe/stripe-go/v76/setupintent"
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
		return errors.New("Customer not set")
	}
	if cus.ID == "" {
		return errors.New("Customer ID not set")
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
	paymentHandler.detachPaymentMethodsButDefault(c, setupIntent)
	return nil
}

func (paymentHandler *PaymentHandler) detachPaymentMethodsButDefault(c context.Context, setupIntent stripe.SetupIntent) error {
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
