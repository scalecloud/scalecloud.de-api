package stripe

import (
	"context"
	"errors"
	"strconv"

	"github.com/scalecloud/scalecloud.de-api/firebase"
	"github.com/scalecloud/scalecloud.de-api/mongo"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/sub"
	"go.uber.org/zap"
)

func getSubscriptionByID(c context.Context, subscriptionID string) (*stripe.Subscription, error) {
	stripe.Key = getStripeKey()
	return sub.Get(subscriptionID, nil)
}

func CreateCheckoutSubscription(c context.Context, token string, productmodel ProductModel) (CheckoutSubscriptionModel, error) {
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return CheckoutSubscriptionModel{}, err
	}
	filter := mongo.User{
		UID: tokenDetails.UID,
	}
	customerID, err := searchOrCreateCustomer(c, filter, tokenDetails)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return CheckoutSubscriptionModel{}, err
	}
	if customerID == "" {
		logger.Error("Customer ID is empty")
		return CheckoutSubscriptionModel{}, errors.New("Customer ID is empty")
	}
	stripe.Key = getStripeKey()

	price, err := getPrice(c, productmodel.ProductID)
	if err != nil {
		logger.Error("Error getting price", zap.Error(err))
		return CheckoutSubscriptionModel{}, err
	}
	metaData := price.Metadata
	if err != nil {
		logger.Warn("Error getting price metadata", zap.Error(err))
		return CheckoutSubscriptionModel{}, errors.New("Price metadata not found")
	}
	trialPeriodDays, ok := metaData["trialPeriodDays"]
	if !ok {
		logger.Warn("trialPeriodDays not found", zap.Any("priceID", price.ID))
		return CheckoutSubscriptionModel{}, errors.New("trialPeriodDays not found")
	}
	iTrialPeriodDays, err := strconv.ParseInt(trialPeriodDays, 10, 64)
	logger.Info("trialPeriodDays", zap.Any("trialPeriodDays", iTrialPeriodDays))
	if err != nil {
		logger.Warn("Error converting trialPeriodDays to int", zap.Error(err))
		return CheckoutSubscriptionModel{}, errors.New("Error converting trialPeriodDays to int")
	}

	// Automatically save the payment method to the subscription
	// when the first payment is successful.
	paymentSettings := &stripe.SubscriptionPaymentSettingsParams{
		SaveDefaultPaymentMethod: stripe.String("on_subscription"),
	}

	// Create the subscription. Note we're expanding the Subscription's
	// latest invoice and that invoice's payment_intent
	// so we can pass it to the front end to confirm the payment
	subscriptionParams := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(price.ID),
			},
		},
		PaymentSettings: paymentSettings,
		PaymentBehavior: stripe.String("default_incomplete"),
		TrialPeriodDays: stripe.Int64(iTrialPeriodDays),
	}
	subscriptionParams.AddExpand("latest_invoice.payment_intent")
	subscription, err := sub.New(subscriptionParams)
	if err != nil {
		logger.Error("Error creating subscription", zap.Error(err))
		return CheckoutSubscriptionModel{}, err
	}
	logger.Info("Subscription created", zap.Any("subscriptionID", subscription.ID))
	if subscription.LatestInvoice == nil {
		logger.Error("Latest invoice is nil", zap.Any("subscriptionID", subscription.ID))
		return CheckoutSubscriptionModel{}, errors.New("Latest invoice is nil")
	}
	if subscription.LatestInvoice.PaymentIntent == nil {
		logger.Error("Payment intent is nil", zap.Any("subscriptionID", subscription.ID))
		return CheckoutSubscriptionModel{}, errors.New("Payment intent is nil")
	}

	checkoutSubscriptionModel := CheckoutSubscriptionModel{
		SubscriptionID: subscription.ID,
		ClientSecret:   subscription.LatestInvoice.PaymentIntent.ClientSecret,
	}
	return checkoutSubscriptionModel, nil
}
