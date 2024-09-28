package stripemanager

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/subscription"
	"go.uber.org/zap"
)

func (paymentHandler *PaymentHandler) CreateCheckoutSubscription(c context.Context, tokenDetails firebasemanager.TokenDetails, checkoutCreateSubscriptionRequest CheckoutCreateSubscriptionRequest) (CheckoutCreateSubscriptionReply, error) {
	stripe.Key = paymentHandler.StripeConnection.Key
	if checkoutCreateSubscriptionRequest.Quantity > 999 {
		return CheckoutCreateSubscriptionReply{}, errors.New("quantity can not be higher than 999")
	}
	price, err := paymentHandler.StripeConnection.GetPrice(c, checkoutCreateSubscriptionRequest.ProductID)
	if err != nil {
		return CheckoutCreateSubscriptionReply{}, err
	}
	cus, err := paymentHandler.GetCustomerByUID(c, tokenDetails.UID)
	if err != nil {
		return CheckoutCreateSubscriptionReply{}, err
	}
	paymentMethod, err := paymentHandler.StripeConnection.GetDefaultPaymentMethod(c, cus)
	if err != nil {
		return CheckoutCreateSubscriptionReply{}, err
	}
	product, err := paymentHandler.StripeConnection.GetProduct(c, checkoutCreateSubscriptionRequest.ProductID)
	if err != nil {
		return CheckoutCreateSubscriptionReply{}, err
	}

	iTrialPeriodDays, err := paymentHandler.getTrialDaysForCustomer(c, checkoutCreateSubscriptionRequest.Quantity, paymentMethod, product, cus)
	if err != nil {
		return CheckoutCreateSubscriptionReply{}, err
	}
	subscriptionParams := &stripe.SubscriptionParams{
		Customer: stripe.String(cus.ID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price:    stripe.String(price.ID),
				Quantity: stripe.Int64(checkoutCreateSubscriptionRequest.Quantity),
			},
		},
	}
	if iTrialPeriodDays > 0 {
		subscriptionParams.TrialPeriodDays = stripe.Int64(iTrialPeriodDays)
	}
	sub, err := subscription.New(subscriptionParams)
	if err != nil {
		paymentHandler.Log.Error("Error creating subscription", zap.Error(err))
		return CheckoutCreateSubscriptionReply{}, err
	}
	paymentHandler.Log.Info("Subscription created.", zap.Any("subscriptionID", sub.ID), zap.Any("status", sub.Status))
	if sub.Status == stripe.SubscriptionStatusActive || sub.Status == stripe.SubscriptionStatusTrialing {
		paymentHandler.Log.Info("Subscription is valid.")
		createSeat(c, sub, tokenDetails, paymentHandler)
	} else if sub.Status == stripe.SubscriptionStatusIncomplete || sub.Status == stripe.SubscriptionStatusIncompleteExpired {
		paymentHandler.Log.Warn("First payment did not work. Subscription is incomplete.", zap.Any("subscriptionID", sub.ID), zap.Any("status", sub.Status))
	} else {
		paymentHandler.Log.Error("Subscription should not get this status. Canceling subscription.", zap.Any("subscriptionID", sub.ID), zap.Any("status", sub.Status))
		sub, err = subscription.Cancel(sub.ID, nil)
		if err != nil {
			paymentHandler.Log.Error("Error canceling subscription", zap.Error(err))
		}
	}
	checkoutSubscriptionModel := CheckoutCreateSubscriptionReply{
		Status:         string(sub.Status),
		SubscriptionID: sub.ID,
		ProductName:    product.Name,
		EMail:          tokenDetails.EMail,
		TrialEnd:       sub.TrialEnd,
	}
	return checkoutSubscriptionModel, nil
}

func createSeat(c context.Context, sub *stripe.Subscription, tokenDetails firebasemanager.TokenDetails, paymentHandler *PaymentHandler) {
	emailVerified := false
	seat := mongomanager.Seat{
		SubscriptionID: sub.ID,
		UID:            tokenDetails.UID,
		EMail:          tokenDetails.EMail,
		EMailVerified:  &emailVerified,
		Roles: []mongomanager.Role{
			mongomanager.RoleOwner,
			mongomanager.RoleAdministrator,
			mongomanager.RoleUser,
			mongomanager.RoleBilling,
		},
	}
	err := paymentHandler.MongoConnection.CreateSeat(c, seat)
	if err != nil {
		paymentHandler.Log.Error("Error creating seat", zap.Error(err))
	}
}

func (paymentHandler *PaymentHandler) GetCheckoutProduct(c context.Context, tokenDetails firebasemanager.TokenDetails, checkoutProductRequest CheckoutProductRequest) (CheckoutProductReply, error) {
	stripe.Key = paymentHandler.StripeConnection.Key
	price, err := paymentHandler.StripeConnection.GetPrice(c, checkoutProductRequest.ProductID)
	if err != nil {
		return CheckoutProductReply{}, err
	}
	currency := strings.ToUpper(string(price.Currency))
	product, err := paymentHandler.StripeConnection.GetProduct(c, checkoutProductRequest.ProductID)
	if err != nil {
		return CheckoutProductReply{}, err
	}
	if product.Metadata == nil {
		return CheckoutProductReply{}, errors.New("product metadata not found")
	}
	cus, err := paymentHandler.GetCustomerByUID(c, tokenDetails.UID)
	if err != nil {
		return CheckoutProductReply{}, err
	}
	paymentMethod, err := paymentHandler.StripeConnection.GetDefaultPaymentMethod(c, cus)
	hasValidPaymentMethod := true
	if err != nil {
		if err == ErrDefaultPaymentMethodNotFound {
			hasValidPaymentMethod = false
		} else {
			return CheckoutProductReply{}, err
		}
	}
	iTrialPeriodDays, err := paymentHandler.getTrialDaysForCustomer(c, 1, paymentMethod, product, cus)
	if err != nil {
		return CheckoutProductReply{}, err
	}
	metaDataProduct := product.Metadata
	if metaDataProduct == nil {
		return CheckoutProductReply{}, errors.New("product metadata not found")
	}
	storageAmount, ok := metaDataProduct["storageAmount"]
	if !ok {
		return CheckoutProductReply{}, errors.New("storageAmount not found for priceID: " + price.ID)
	}
	iStorageAmount, err := strconv.ParseInt(storageAmount, 10, 64)
	if err != nil {
		paymentHandler.Log.Warn("Error converting storageAmount to int", zap.Error(err))
		return CheckoutProductReply{}, errors.New("error converting storageAmount")
	}
	storageUnit, ok := metaDataProduct["storageUnit"]
	if !ok {
		return CheckoutProductReply{}, errors.New("StorageUnit not found for priceID: " + price.ID)
	}
	productName := product.Name
	if productName == "" {
		return CheckoutProductReply{}, errors.New("Product name not found for priceID: " + price.ID)
	}
	checkoutProductReply := CheckoutProductReply{
		ProductID:             checkoutProductRequest.ProductID,
		Name:                  productName,
		StorageAmount:         iStorageAmount,
		StorageUnit:           storageUnit,
		TrialDays:             iTrialPeriodDays,
		PricePerMonth:         price.UnitAmount,
		Currency:              currency,
		HasValidPaymentMethod: &hasValidPaymentMethod,
	}
	return checkoutProductReply, nil
}
