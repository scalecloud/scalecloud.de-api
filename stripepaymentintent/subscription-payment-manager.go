package stripepaymentintent

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/stripecustomer"
	"github.com/scalecloud/scalecloud.de-api/stripesecret"
	"github.com/scalecloud/scalecloud.de-api/stripesubscription"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/paymentmethod"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func GetSubscriptionPaymentMethod(c context.Context, token string, request SubscriptionPaymentMethodRequest) (SubscriptionPaymentMethodReply, error) {
	if request.ID == "" {
		return SubscriptionPaymentMethodReply{}, errors.New("Subscription ID is empty")
	}
	tokenDetails, err := firebasemanager.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return SubscriptionPaymentMethodReply{}, err
	}
	customerID, err := stripecustomer.GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return SubscriptionPaymentMethodReply{}, err
	}
	stripe.Key = stripesecret.GetStripeKey()
	subscription, error := stripesubscription.GetSubscriptionByID(c, request.ID)
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
