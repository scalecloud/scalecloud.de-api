package stripe

import (
	"context"
	"errors"
	"strconv"

	"github.com/scalecloud/scalecloud.de-api/firebase"
	"github.com/stripe/stripe-go/v75"
	"go.uber.org/zap"
)

func GetSubscriptionByID(c context.Context, token, subscriptionID string) (subscriptionDetail SubscriptionDetail, err error) {
	if subscriptionID == "" {
		logger.Error("Subscription ID is empty")
		return SubscriptionDetail{}, errors.New("Subscription ID is empty")
	}
	tokenDetails, err := firebase.GetTokenDetails(c, token)
	if err != nil {
		logger.Error("Error getting token details", zap.Error(err))
		return SubscriptionDetail{}, err
	}
	customerID, err := getCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		logger.Error("Error getting customer ID", zap.Error(err))
		return SubscriptionDetail{}, err
	}
	stripe.Key = getStripeKey()
	subscription, error := getSubscriptionByID(c, subscriptionID)
	if error != nil {
		logger.Warn("Error getting subscription", zap.Error(error))
		return SubscriptionDetail{}, errors.New("Subscription not found")
	}
	logger.Debug("subscription", zap.Any("subscription", subscription))
	if subscription.Customer.ID != customerID {
		logger.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", subscriptionID))
		return SubscriptionDetail{}, errors.New("Subscription not matching customer")
	} else {
		subscriptionDetail, err = mapSubscriptionItemToSubscriptionDetail(c, subscription)
		if err != nil {
			logger.Error("Error mapping subscription to subscription detail", zap.Error(err))
			return SubscriptionDetail{}, errors.New("Subscription not found")
		}
	}
	return subscriptionDetail, nil
}

func mapSubscriptionItemToSubscriptionDetail(c context.Context, subscription *stripe.Subscription) (subscriptionDetail SubscriptionDetail, err error) {
	subscriptionDetail.ID = subscription.ID
	productID := subscription.Items.Data[0].Price.Product.ID
	logger.Debug("Product ID", zap.String("productID", productID))

	product, err := getProduct(c, productID)
	if err != nil {
		logger.Warn("Error getting product", zap.Error(err))
		return SubscriptionDetail{}, errors.New("Product not found")
	}
	subscriptionDetail.ProductName = product.Name

	subscriptionDetail.Active = product.Active

	metaData := product.Metadata
	if err != nil {
		logger.Warn("Error getting product metadata", zap.Error(err))
		return SubscriptionDetail{}, errors.New("Product metadata not found")
	}
	storageAmount, ok := metaData["storageAmount"]
	if !ok {
		logger.Warn("Storage amount not found", zap.Any("subscriptionID", subscription.ID))
		return SubscriptionDetail{}, errors.New("Storage amount not found")
	}
	iStorageAmount, err := strconv.Atoi(storageAmount)
	if err != nil {
		logger.Warn("Error converting storage amount to int", zap.Error(err))
		return SubscriptionDetail{}, errors.New("Error converting storage amount to int")
	}
	subscriptionDetail.StorageAmount = iStorageAmount
	productType, ok := metaData["productType"]
	if !ok {
		logger.Warn("ProductType not found", zap.Any("subscriptionID", subscription.ID))
		return SubscriptionDetail{}, errors.New("ProductType not found")
	}
	subscriptionDetail.ProductType = productType

	subscriptionDetail.UserCount = subscription.Items.Data[0].Quantity

	price, err := getPrice(c, subscription.Items.Data[0].Price.Product.ID)
	if err != nil {
		logger.Warn("Error getting price", zap.Error(err))
		return SubscriptionDetail{}, errors.New("Price not found")
	}

	subscriptionDetail.PricePerMonth = price.UnitAmount

	subscriptionDetail.Currency = string(price.Currency)

	subscriptionDetail.CancelAtPeriodEnd = subscription.CancelAtPeriodEnd

	subscriptionDetail.CancelAt = subscription.CancelAt

	subscriptionDetail.Status = string(subscription.Status)

	subscriptionDetail.TrialEnd = subscription.TrialEnd

	return subscriptionDetail, nil
}
