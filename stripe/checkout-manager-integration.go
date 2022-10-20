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

func CreateCheckoutSubscription(c context.Context, token string, checkoutIntegrationRequest CheckoutIntegrationRequest) (CheckoutIntegrationReply, error) {
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return CheckoutIntegrationReply{}, err
	}
	filter := mongo.User{
		UID: tokenDetails.UID,
	}
	customerID, err := searchOrCreateCustomer(c, filter, tokenDetails)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return CheckoutIntegrationReply{}, err
	}
	if customerID == "" {
		logger.Error("Customer ID is empty")
		return CheckoutIntegrationReply{}, errors.New("Customer ID is empty")
	}
	stripe.Key = getStripeKey()

	price, err := getPrice(c, checkoutIntegrationRequest.ProductID)
	if err != nil {
		logger.Error("Error getting price", zap.Error(err))
		return CheckoutIntegrationReply{}, err
	}
	metaData := price.Metadata
	if err != nil {
		logger.Warn("Error getting price metadata", zap.Error(err))
		return CheckoutIntegrationReply{}, errors.New("Price metadata not found")
	}
	trialPeriodDays, ok := metaData["trialPeriodDays"]
	if !ok {
		logger.Warn("trialPeriodDays not found", zap.Any("priceID", price.ID))
		return CheckoutIntegrationReply{}, errors.New("trialPeriodDays not found")
	}
	iTrialPeriodDays, err := strconv.ParseInt(trialPeriodDays, 10, 64)
	if err != nil {
		logger.Warn("Error converting trialPeriodDays to int", zap.Error(err))
		return CheckoutIntegrationReply{}, errors.New("Error converting trialPeriodDays to int")
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
				Price:    stripe.String(price.ID),
				Quantity: stripe.Int64(checkoutIntegrationRequest.Quantity),
			},
		},
		PaymentSettings: paymentSettings,
		PaymentBehavior: stripe.String("default_incomplete"),
		TrialPeriodDays: stripe.Int64(iTrialPeriodDays),
	}
	subscriptionParams.AddExpand("latest_invoice.payment_intent")
	subscriptionParams.AddExpand("pending_setup_intent")
	subscription, err := sub.New(subscriptionParams)
	if err != nil {
		logger.Error("Error creating subscription", zap.Error(err))
		return CheckoutIntegrationReply{}, err
	}
	logger.Info("Subscription created and waiting for payment.", zap.Any("subscriptionID", subscription.ID))
	if subscription.PendingSetupIntent == nil {
		logger.Error("Pending setup intent is nil")
		return CheckoutIntegrationReply{}, errors.New("Pending setup intent is nil")
	}
	if subscription.PendingSetupIntent.ClientSecret == "" {
		logger.Error("Pending setup intent client secret is nil")
		return CheckoutIntegrationReply{}, errors.New("Pending setup intent client secret is nil")
	}

	checkoutSubscriptionModel := CheckoutIntegrationReply{
		SubscriptionID: subscription.ID,
		ClientSecret:   subscription.PendingSetupIntent.ClientSecret,
		Quantity:       subscription.Quantity,
	}
	return checkoutSubscriptionModel, nil
}

func UpdateCheckoutSubscription(c context.Context, token string, checkoutIntegrationUpdateRequest CheckoutIntegrationUpdateRequest) (CheckoutIntegrationUpdateReply, error) {
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return CheckoutIntegrationUpdateReply{}, err
	}
	customerIDFromUID, err := getCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customerID", zap.Error(err))
		return CheckoutIntegrationUpdateReply{}, err
	}

	stripe.Key = getStripeKey()

	subscription, err := getSubscriptionByID(c, checkoutIntegrationUpdateRequest.SubscriptionID)
	if err != nil {
		logger.Error("Error getting subscription", zap.Error(err))
		return CheckoutIntegrationUpdateReply{}, err
	}
	if subscription.Customer == nil {
		logger.Error("Customer is nil")
		return CheckoutIntegrationUpdateReply{}, errors.New("Customer is nil")
	}
	customerIDFromSubscription := *&subscription.Customer.ID
	if customerIDFromSubscription == "" {
		logger.Error("Customer ID is empty")
		return CheckoutIntegrationUpdateReply{}, errors.New("Customer ID is empty")
	}
	if customerIDFromUID != customerIDFromSubscription {
		logger.Error("Customer ID from UID does not match customer ID from subscription")
		return CheckoutIntegrationUpdateReply{}, errors.New("CustomerID from UID does not match customerID from subscription")
	}

	subscriptionItemID := subscription.Items.Data[0].ID
	if subscriptionItemID == "" {
		logger.Error("Subscription item ID is empty")
		return CheckoutIntegrationUpdateReply{}, errors.New("Subscription item ID is empty")
	}
	params := &stripe.SubscriptionParams{
		Quantity: stripe.Int64(checkoutIntegrationUpdateRequest.Quantity),
	}
	subscriptionUpdated, err := sub.Update(
		checkoutIntegrationUpdateRequest.SubscriptionID,
		params,
	)
	if err != nil {
		logger.Error("Error updating subscription", zap.Error(err))
		return CheckoutIntegrationUpdateReply{}, err
	}
	if checkoutIntegrationUpdateRequest.Quantity != subscriptionUpdated.Quantity {
		logger.Error("Requested Quantity does not match updated qantity")
		return CheckoutIntegrationUpdateReply{}, errors.New("Requested Quantity does not match updated qantity")
	}
	checkoutIntegrationUpdateReturn := CheckoutIntegrationUpdateReply{
		SubscriptionID: subscriptionUpdated.ID,
		ClientSecret:   subscriptionUpdated.PendingSetupIntent.ClientSecret,
		Quantity:       subscriptionUpdated.Quantity,
	}
	return checkoutIntegrationUpdateReturn, nil
}

func GetCheckoutProduct(c context.Context, token string, checkoutProductRequest CheckoutProductRequest) (CheckoutProductReply, error) {
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return CheckoutProductReply{}, err
	}
	customerIDFromUID, err := getCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customerID", zap.Error(err))
		return CheckoutProductReply{}, err
	}

	stripe.Key = getStripeKey()

	subscription, err := getSubscriptionByID(c, checkoutProductRequest.SubscriptionID)
	if err != nil {
		logger.Error("Error getting subscription", zap.Error(err))
		return CheckoutProductReply{}, err
	}
	if subscription.Customer == nil {
		logger.Error("Customer is nil")
		return CheckoutProductReply{}, errors.New("Customer is nil")
	}
	customerIDFromSubscription := *&subscription.Customer.ID
	if customerIDFromSubscription == "" {
		logger.Error("Customer ID is empty")
		return CheckoutProductReply{}, errors.New("Customer ID is empty")
	}
	if customerIDFromUID != customerIDFromSubscription {
		logger.Error("Customer ID from UID does not match customer ID from subscription")
		return CheckoutProductReply{}, errors.New("CustomerID from UID does not match customerID from subscription")
	}
	subscriptionItem := subscription.Items.Data[0]
	if subscriptionItem == nil {
		logger.Error("Subscription item is nill")
		return CheckoutProductReply{}, errors.New("Subscription item is nill")
	}
	price, err := getPrice(c, subscriptionItem.Plan.Product.ID)
	if err != nil {
		logger.Error("Error getting price", zap.Error(err))
		return CheckoutProductReply{}, err
	}
	metaData := price.Metadata
	if err != nil {
		logger.Warn("Error getting price metadata", zap.Error(err))
		return CheckoutProductReply{}, errors.New("Price metadata not found")
	}
	trialPeriodDays, ok := metaData["trialPeriodDays"]
	if !ok {
		logger.Warn("trialPeriodDays not found", zap.Any("priceID", price.ID))
		return CheckoutProductReply{}, errors.New("trialPeriodDays not found")
	}
	iTrialPeriodDays, err := strconv.ParseInt(trialPeriodDays, 10, 64)
	if err != nil {
		logger.Warn("Error converting trialPeriodDays to int", zap.Error(err))
		return CheckoutProductReply{}, errors.New("Error converting trialPeriodDays to int")
	}

	storageAmount, ok := metaData["storageAmount"]
	if !ok {
		logger.Warn("storageAmount not found", zap.Any("priceID", price.ID))
		return CheckoutProductReply{}, errors.New("storageAmount not found")
	}
	iStorageAmount, err := strconv.ParseInt(storageAmount, 10, 64)
	if err != nil {
		logger.Warn("Error converting storageAmount to int", zap.Error(err))
		return CheckoutProductReply{}, errors.New("Error converting storageAmount to int")
	}

	storageUnit, ok := metaData["storageUnit"]
	if !ok {
		logger.Warn("StorageUnit not found", zap.Any("priceID", price.ID))
		return CheckoutProductReply{}, errors.New("StorageUnit not found")
	}

	checkoutProductReply := CheckoutProductReply{
		SubscriptionID: subscription.ID,
		ProductID:      price.Product.ID,
		Name:           price.Product.Name,
		StorageAmount:  iStorageAmount,
		StorageUnit:    storageUnit,
		TrialDays:      iTrialPeriodDays,
		PricePerMonth:  price.UnitAmount,
	}
	return checkoutProductReply, nil
}
