package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/paymentmethod"
	"github.com/stripe/stripe-go/v75/setupintent"
	"go.uber.org/zap"
)

type SetupIntentMeta string

const SetupIntentMetaKey SetupIntentMeta = "setupIntentMeta"

const (
	CreateSubscription SetupIntentMeta = "createSubscription"
	ChangePayment      SetupIntentMeta = "changePayment"
)

func GetSubscriptionPaymentMethod(c context.Context, token string, request SubscriptionPaymentMethodRequest) (SubscriptionPaymentMethodReply, error) {
	if request.ID == "" {
		return SubscriptionPaymentMethodReply{}, errors.New("Subscription ID is empty")
	}
	tokenDetails, err := firebasemanager.GetTokenDetails(c, token)
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
	if subscription.DefaultPaymentMethod == nil {
		logger.Error("No default payment method found", zap.String("subscriptionID", request.ID))
		return SubscriptionPaymentMethodReply{}, errors.New("No default payment method found")
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
			ExpMonth: uint64(pm.Card.ExpMonth),
			ExpYear:  uint64(pm.Card.ExpYear),
		}
		return reply, nil
	}
	return SubscriptionPaymentMethodReply{}, errors.New("Payment method not found")
}

func GetChangePaymentSetupIntent(c context.Context, token string, request ChangePaymentRequest) (ChangePaymentReply, error) {
	if request.SubscriptionID == "" {
		return ChangePaymentReply{}, errors.New("Subscription ID is empty")
	}
	tokenDetails, err := firebasemanager.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return ChangePaymentReply{}, err
	}
	customerID, err := getCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return ChangePaymentReply{}, err
	}
	stripe.Key = getStripeKey()
	subscription, error := getSubscriptionByID(c, request.SubscriptionID)
	if error != nil {
		logger.Warn("Error getting subscription", zap.Error(error))
		return ChangePaymentReply{}, errors.New("Subscription not found")
	}
	if subscription.Customer.ID != customerID {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.SubscriptionID))
		return ChangePaymentReply{}, errors.New("Subscription not matching customer")
	}

	params := &stripe.SetupIntentParams{
		Customer: stripe.String(customerID),
	}

	params.AddMetadata(string(SetupIntentMetaKey), string(ChangePayment))

	si, err := setupintent.New(params)
	if err != nil {
		logger.Error("Error creating setup intent", zap.Error(err))
		return ChangePaymentReply{}, err
	}

	reply := ChangePaymentReply{
		SetupIntentID: si.ID,
		ClientSecret:  si.ClientSecret,
	}
	return reply, nil
}
