package stripeprice

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/stripesecret"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/price"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

var logger, _ = zap.NewProduction()

func GetPrice(c context.Context, productID string) (*stripe.Price, error) {
	priceSearch := &stripe.Price{}
	stripe.Key = stripesecret.GetStripeKey()
	params := &stripe.PriceListParams{
		Product: stripe.String(productID),
		Active:  stripe.Bool(true),
	}
	iter := price.List(params)
	priceSearch = searchPrice(iter, priceSearch, productID)
	if priceSearch.ID == "" {
		logger.Error("No active price for productID" + productID)
		return nil, errors.New("No active price for productID" + productID)
	}
	return priceSearch, nil
}

func searchPrice(iter *price.Iter, ret *stripe.Price, productID string) *stripe.Price {
	for {
		if iter.Next() {
			if ret.ID != "" {
				logger.Warn("More than one active price for productID"+productID, zap.Error(iter.Err()))
			} else {
				ret = iter.Price()
				logger.Info("Price", zap.Any("priceID", ret.ID))
			}
		} else {
			checkIterErrors(iter)
			break
		}
	}
	return ret
}

func checkIterErrors(iter *price.Iter) {
	if iter.Err() != nil {
		if iter.Err() == iterator.Done {
			logger.Debug("Iteration done")
		} else {
			logger.Error("Error getting price", zap.Error(iter.Err()))
		}
	} else {
		logger.Debug("Iteration done with no error.")
	}
}
