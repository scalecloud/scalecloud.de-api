package stripemanager

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/subscription"
	"go.uber.org/zap"
)

func (stripeConnection *StripeConnection) CreateCheckoutSubscription(c context.Context, tokenDetails firebasemanager.TokenDetails, checkoutIntegrationRequest CheckoutPaymentIntentRequest) (CheckoutPaymentIntentReply, error) {
	filter := mongomanager.User{
		UID: tokenDetails.UID,
	}
	customerID, err := stripeConnection.searchOrCreateCustomer(c, filter, tokenDetails)
	if err != nil {
		return CheckoutPaymentIntentReply{}, err
	}
	if customerID == "" {
		return CheckoutPaymentIntentReply{}, errors.New("Customer ID is empty")
	}
	stripe.Key = stripeConnection.Key

	price, err := stripeConnection.GetPrice(c, checkoutIntegrationRequest.ProductID)
	if err != nil {
		return CheckoutPaymentIntentReply{}, err
	}
	metaData := price.Metadata
	if metaData == nil {
		return CheckoutPaymentIntentReply{}, errors.New("Price metadata not found")
	}
	trialPeriodDays, ok := metaData["trialPeriodDays"]
	if !ok {
		return CheckoutPaymentIntentReply{}, errors.New("trialPeriodDays not found for priceID: " + price.ID)
	}
	iTrialPeriodDays, err := strconv.ParseInt(trialPeriodDays, 10, 64)
	if err != nil {
		stripeConnection.Log.Error("Error converting trialPeriodDays to int", zap.Error(err))
		return CheckoutPaymentIntentReply{}, errors.New("Error converting trialPeriodDays")
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
	sub, err := subscription.New(subscriptionParams)
	if err != nil {
		stripeConnection.Log.Error("Error creating subscription", zap.Error(err))
		return CheckoutPaymentIntentReply{}, err
	}
	if sub.PendingSetupIntent == nil {
		return CheckoutPaymentIntentReply{}, errors.New("Pending setup intent is nil")
	}
	if sub.PendingSetupIntent.ClientSecret == "" {
		return CheckoutPaymentIntentReply{}, errors.New("Pending setup intent client secret is nil")
	}
	stripeConnection.Log.Info("Subscription created and waiting for payment.", zap.Any("subscriptionID", sub.ID))

	checkoutSubscriptionModel := CheckoutPaymentIntentReply{
		SubscriptionID: sub.ID,
		ClientSecret:   sub.PendingSetupIntent.ClientSecret,
		Quantity:       sub.Items.Data[0].Quantity,
		EMail:          tokenDetails.EMail,
	}
	return checkoutSubscriptionModel, nil
}

func (stripeConnection *StripeConnection) UpdateCheckoutSubscription(c context.Context, tokenDetails firebasemanager.TokenDetails, checkoutIntegrationUpdateRequest CheckoutPaymentIntentUpdateRequest) (CheckoutPaymentIntentUpdateReply, error) {
	customerIDFromUID, err := GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return CheckoutPaymentIntentUpdateReply{}, err
	}

	stripe.Key = stripeConnection.Key

	sub, err := stripeConnection.GetSubscriptionByID(c, checkoutIntegrationUpdateRequest.SubscriptionID)
	if err != nil {
		return CheckoutPaymentIntentUpdateReply{}, err
	}
	if sub.Customer == nil {
		return CheckoutPaymentIntentUpdateReply{}, errors.New("Customer is nil")
	}
	customerIDFromSubscription := *&sub.Customer.ID
	if customerIDFromSubscription == "" {
		return CheckoutPaymentIntentUpdateReply{}, errors.New("Customer ID is empty")
	}
	if customerIDFromUID != customerIDFromSubscription {
		return CheckoutPaymentIntentUpdateReply{}, errors.New("CustomerID from UID does not match customerID from subscription")
	}

	subscriptionItemID := sub.Items.Data[0].ID
	if subscriptionItemID == "" {
		return CheckoutPaymentIntentUpdateReply{}, errors.New("Subscription item ID is empty")
	}
	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:       stripe.String(subscriptionItemID),
				Quantity: stripe.Int64(checkoutIntegrationUpdateRequest.Quantity),
			},
		},
	}

	subscriptionUpdated, err := subscription.Update(
		checkoutIntegrationUpdateRequest.SubscriptionID,
		params,
	)
	if err != nil {
		return CheckoutPaymentIntentUpdateReply{}, err
	}
	if checkoutIntegrationUpdateRequest.Quantity != subscriptionUpdated.Items.Data[0].Quantity {
		return CheckoutPaymentIntentUpdateReply{}, errors.New("Requested Quantity does not match updated qantity")
	}
	checkoutIntegrationUpdateReturn := CheckoutPaymentIntentUpdateReply{
		SubscriptionID: subscriptionUpdated.ID,
		ClientSecret:   subscriptionUpdated.PendingSetupIntent.ClientSecret,
		Quantity:       subscriptionUpdated.Items.Data[0].Quantity,
	}
	return checkoutIntegrationUpdateReturn, nil
}

func (stripeConnection *StripeConnection) GetCheckoutProduct(c context.Context, tokenDetails firebasemanager.TokenDetails, checkoutProductRequest CheckoutProductRequest) (CheckoutProductReply, error) {
	customerIDFromUID, err := GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return CheckoutProductReply{}, err
	}

	stripe.Key = stripeConnection.Key

	subscription, err := stripeConnection.GetSubscriptionByID(c, checkoutProductRequest.SubscriptionID)
	if err != nil {
		return CheckoutProductReply{}, err
	}
	if subscription.Customer == nil {
		return CheckoutProductReply{}, errors.New("Customer is nil")
	}
	customerIDFromSubscription := *&subscription.Customer.ID
	if customerIDFromSubscription == "" {
		return CheckoutProductReply{}, errors.New("Customer ID is empty")
	}
	if customerIDFromUID != customerIDFromSubscription {
		return CheckoutProductReply{}, errors.New("CustomerID from UID does not match customerID from subscription")
	}
	subscriptionItem := subscription.Items.Data[0]
	if subscriptionItem == nil {
		return CheckoutProductReply{}, errors.New("Subscription item is nill")
	}
	productID := subscriptionItem.Price.Product.ID

	price, err := stripeConnection.GetPrice(c, productID)
	if err != nil {
		return CheckoutProductReply{}, err
	}

	currency := strings.ToUpper(string(price.Currency))

	metaDataPrice := price.Metadata
	if metaDataPrice == nil {
		return CheckoutProductReply{}, errors.New("Price metadata not found")
	}
	trialPeriodDays, ok := metaDataPrice["trialPeriodDays"]
	if !ok {
		return CheckoutProductReply{}, errors.New("trialPeriodDays not found for priceID: " + price.ID)
	}
	iTrialPeriodDays, err := strconv.ParseInt(trialPeriodDays, 10, 64)
	if err != nil {
		stripeConnection.Log.Warn("Error converting trialPeriodDays to int", zap.Error(err))
		return CheckoutProductReply{}, errors.New("Error converting trialPeriodDays")
	}

	product, err := stripeConnection.GetProduct(c, productID)
	if err != nil {
		return CheckoutProductReply{}, err
	}

	productName := product.Name

	metaDataProduct := product.Metadata

	storageAmount, ok := metaDataProduct["storageAmount"]
	if !ok {
		return CheckoutProductReply{}, errors.New("storageAmount not found for priceID: " + price.ID)
	}
	iStorageAmount, err := strconv.ParseInt(storageAmount, 10, 64)
	if err != nil {
		stripeConnection.Log.Warn("Error converting storageAmount to int", zap.Error(err))
		return CheckoutProductReply{}, errors.New("Error converting storageAmount")
	}

	storageUnit, ok := metaDataProduct["storageUnit"]
	if !ok {
		return CheckoutProductReply{}, errors.New("StorageUnit not found for priceID: " + price.ID)
	}

	checkoutProductReply := CheckoutProductReply{
		SubscriptionID: subscription.ID,
		ProductID:      productID,
		Name:           productName,
		StorageAmount:  iStorageAmount,
		StorageUnit:    storageUnit,
		TrialDays:      iTrialPeriodDays,
		PricePerMonth:  price.UnitAmount,
		Currency:       currency,
	}
	return checkoutProductReply, nil
}
