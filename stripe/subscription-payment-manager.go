package stripe

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebase"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/paymentmethod"
	"github.com/stripe/stripe-go/setupintent"
	"go.uber.org/zap"
)

func GetSubscriptionPaymentMethod(c context.Context, token string, request SubscriptionPaymentMethodRequest) (SubscriptionPaymentMethodReply, error) {
	if request.ID == "" {
		return SubscriptionPaymentMethodReply{}, errors.New("Subscription ID is empty")
	}
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return SubscriptionPaymentMethodReply{}, err
	}
	customerID, err := getCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return SubscriptionPaymentMethodReply{}, err
	}
	stripe.Key = getStripeKey()
	subscription, error := getSubscriptionByID(c, request.ID)
	if error != nil {
		logger.Warn("Error getting subscription", zap.Error(error))
		return SubscriptionPaymentMethodReply{}, errors.New("Subscription not found")
	}
	if subscription.Customer.ID != customerID {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.ID))
		return SubscriptionPaymentMethodReply{}, errors.New("Subscription not matching customer")
	}
	paymentMethodID := subscription.DefaultPaymentMethod.ID
	if paymentMethodID == "" {
		logger.Error("No default payment method found", zap.String("subscriptionID", request.ID))
		return SubscriptionPaymentMethodReply{}, errors.New("No default payment method found")
	}
	pm, err := paymentmethod.Get(
		paymentMethodID,
		nil,
	)
	if err != nil {
		logger.Error("Error getting payment method", zap.Error(err))
		return SubscriptionPaymentMethodReply{}, err
	}
	if string(pm.Type) == string(stripe.PaymentMethodType(stripe.PaymentMethodTypeCard)) {
		brand := string(pm.Card.Brand)
		reply := SubscriptionPaymentMethodReply{
			ID:       subscription.ID,
			Type:     string(pm.Type),
			Brand:    brand,
			Last4:    pm.Card.Last4,
			ExpMonth: pm.Card.ExpMonth,
			ExpYear:  pm.Card.ExpYear,
		}
		return reply, nil
	}
	return SubscriptionPaymentMethodReply{}, errors.New("Payment method not found")
}

func ChangeSubscriptionPaymentMethod(c context.Context, token string, request SubscriptionSetupIntentRequest) (SubscriptionSetupIntentReply, error) {
	if request.SubscriptionID == "" {
		return SubscriptionSetupIntentReply{}, errors.New("Subscription ID is empty")
	}
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return SubscriptionSetupIntentReply{}, err
	}
	customerID, err := getCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return SubscriptionSetupIntentReply{}, err
	}
	stripe.Key = getStripeKey()
	subscription, error := getSubscriptionByID(c, request.SubscriptionID)
	if error != nil {
		logger.Warn("Error getting subscription", zap.Error(error))
		return SubscriptionSetupIntentReply{}, errors.New("Subscription not found")
	}
	if subscription.Customer.ID != customerID {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.SubscriptionID))
		return SubscriptionSetupIntentReply{}, errors.New("Subscription not matching customer")
	}

	params := &stripe.SetupIntentParams{
		PaymentMethodTypes: []*string{
			stripe.String("card"),
			stripe.String("sepa_debit"),
			stripe.String("paypal"),
		},
		Customer: stripe.String(customerID),
	}
	si, err := setupintent.New(params)
	if err != nil {
		logger.Error("Error creating setup intent", zap.Error(err))
		return SubscriptionSetupIntentReply{}, err
	}

	reply := SubscriptionSetupIntentReply{
		SetupIntentID: si.ID,
		ClientSecret:  si.ClientSecret,
	}
	return reply, nil
}
