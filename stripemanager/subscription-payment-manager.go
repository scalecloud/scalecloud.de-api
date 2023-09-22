package stripemanager

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/paymentmethod"
	"go.uber.org/zap"
)

func (stripeConnection *StripeConnection) GetSubscriptionPaymentMethod(c context.Context, tokenDetails firebasemanager.TokenDetails, request SubscriptionPaymentMethodRequest) (SubscriptionPaymentMethodReply, error) {
	if request.ID == "" {
		return SubscriptionPaymentMethodReply{}, errors.New("Subscription ID is empty")
	}
	customerID, err := GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return SubscriptionPaymentMethodReply{}, err
	}
	stripe.Key = stripeConnection.Key
	sub, error := stripeConnection.GetSubscriptionByID(c, request.ID)
	if error != nil {
		logger.Warn("Error getting subscription", zap.Error(error))
		return SubscriptionPaymentMethodReply{}, errors.New("Subscription not found")
	}
	if sub.Customer.ID != customerID {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", request.ID))
		return SubscriptionPaymentMethodReply{}, errors.New("Subscription not matching customer")
	}
	if sub.DefaultPaymentMethod == nil {
		logger.Error("No default payment method found", zap.String("subscriptionID", request.ID))
		return SubscriptionPaymentMethodReply{}, errors.New("No default payment method found")
	}
	paymentMethodID := sub.DefaultPaymentMethod.ID
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
			ID:       sub.ID,
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
