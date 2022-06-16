package stripe

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/price"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

func getPrice(c context.Context, productID string) (*stripe.Price, error) {
	ret := &stripe.Price{}
	stripe.Key = getStripeKey()
	params := &stripe.PriceListParams{
		Product: stripe.String(productID),
		Active:  stripe.Bool(true),
	}
	iter := price.List(params)
	for {
		if iter.Next() {
			if ret.ID != "" {
				logger.Warn("More than one active price for productID"+productID, zap.Error(iter.Err()))
			} else {
				ret = iter.Price()
				logger.Info("Price", zap.Any("priceID", ret.ID))
			}
		} else {
			if iter.Err() != nil {
				if iter.Err() == iterator.Done {
					logger.Debug("Iteration done")
					break
				} else {
					logger.Error("Error getting price", zap.Error(iter.Err()))
					break
				}
			} else {
				logger.Debug("Iteration done with no error.")
				break
			}
		}
	}
	if ret.ID == "" {
		logger.Error("No active price for productID" + productID)
		return nil, errors.New("No active price for productID" + productID)
	}
	return ret, nil
}
