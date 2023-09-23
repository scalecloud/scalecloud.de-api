package stripemanager

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/price"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

func (stripeConnection *StripeConnection) GetPrice(c context.Context, productID string) (*stripe.Price, error) {
	priceSearch := &stripe.Price{}
	stripe.Key = stripeConnection.Key
	params := &stripe.PriceListParams{
		Product: stripe.String(productID),
		Active:  stripe.Bool(true),
	}
	iter := price.List(params)
	priceSearch = stripeConnection.searchPrice(iter, priceSearch, productID)
	if priceSearch.ID == "" {
		return nil, errors.New("No active price for productID" + productID)
	}
	return priceSearch, nil
}

func (stripeConnection *StripeConnection) searchPrice(iter *price.Iter, ret *stripe.Price, productID string) *stripe.Price {
	for {
		if iter.Next() {
			if ret.ID != "" {
				stripeConnection.Log.Warn("More than one active price for productID"+productID, zap.Error(iter.Err()))
			} else {
				ret = iter.Price()
				stripeConnection.Log.Info("Price", zap.Any("priceID", ret.ID))
			}
		} else {
			stripeConnection.checkIterErrors(iter)
			break
		}
	}
	return ret
}

func (stripeConnection *StripeConnection) checkIterErrors(iter *price.Iter) {
	if iter.Err() != nil {
		if iter.Err() == iterator.Done {
			stripeConnection.Log.Debug("Iteration done")
		} else {
			stripeConnection.Log.Error("Error getting price", zap.Error(iter.Err()))
		}
	} else {
		stripeConnection.Log.Debug("Iteration done with no error.")
	}
}
