package stripemanager

import (
	"context"
	"errors"
	"strconv"

	"github.com/scalecloud/scalecloud.de-api/firebasemanager"
	"github.com/stripe/stripe-go/v75"
	"go.uber.org/zap"
)

func (stripeConnection *StripeConnection) GetSubscriptionDetailByID(c context.Context, tokenDetails firebasemanager.TokenDetails, subscriptionID string) (subscriptionDetail SubscriptionDetail, err error) {
	if subscriptionID == "" {
		return SubscriptionDetail{}, errors.New("Subscription ID is empty")
	}
	customerID, err := GetCustomerIDByUID(c, tokenDetails.UID)
	if err != nil {
		return SubscriptionDetail{}, err
	}
	stripe.Key = stripeConnection.Key
	subscription, err := stripeConnection.GetSubscriptionByID(c, subscriptionID)
	if err != nil {
		return SubscriptionDetail{}, errors.New("Subscription not found")
	}
	stripeConnection.Log.Debug("subscription", zap.Any("subscription", subscription))
	if subscription.Customer.ID != customerID {
		stripeConnection.Log.Error("Tried to request subscription for wrong customer", zap.String("customerID", customerID), zap.String("subscriptionID", subscriptionID))
		return SubscriptionDetail{}, errors.New("Subscription not matching customer")
	} else {
		subscriptionDetail, err = stripeConnection.mapSubscriptionItemToSubscriptionDetail(c, subscription)
		if err != nil {
			return SubscriptionDetail{}, err
		}
	}
	return subscriptionDetail, nil
}

func (stripeConnection *StripeConnection) mapSubscriptionItemToSubscriptionDetail(c context.Context, subscription *stripe.Subscription) (subscriptionDetail SubscriptionDetail, err error) {
	subscriptionDetail.ID = subscription.ID
	productID := subscription.Items.Data[0].Price.Product.ID
	stripeConnection.Log.Debug("Product ID", zap.String("productID", productID))

	prod, err := stripeConnection.GetProduct(c, productID)
	if err != nil {
		return SubscriptionDetail{}, errors.New("Product not found")
	}
	subscriptionDetail.ProductName = prod.Name

	subscriptionDetail.Active = prod.Active

	metaData := prod.Metadata
	if err != nil {
		return SubscriptionDetail{}, errors.New("Product metadata not found")
	}
	storageAmount, ok := metaData["storageAmount"]
	if !ok {
		return SubscriptionDetail{}, errors.New("Storage amount not found")
	}
	iStorageAmount, err := strconv.Atoi(storageAmount)
	if err != nil {
		return SubscriptionDetail{}, errors.New("Error converting storage amount to int")
	}
	subscriptionDetail.StorageAmount = iStorageAmount
	productType, ok := metaData["productType"]
	if !ok {
		return SubscriptionDetail{}, errors.New("ProductType not found")
	}
	subscriptionDetail.ProductType = productType

	subscriptionDetail.UserCount = subscription.Items.Data[0].Quantity

	pri, err := stripeConnection.GetPrice(c, subscription.Items.Data[0].Price.Product.ID)
	if err != nil {
		return SubscriptionDetail{}, errors.New("Price not found")
	}

	subscriptionDetail.PricePerMonth = pri.UnitAmount

	subscriptionDetail.Currency = string(pri.Currency)

	subscriptionDetail.CancelAtPeriodEnd = subscription.CancelAtPeriodEnd

	subscriptionDetail.CancelAt = subscription.CancelAt

	subscriptionDetail.Status = string(subscription.Status)

	subscriptionDetail.TrialEnd = subscription.TrialEnd

	return subscriptionDetail, nil
}
