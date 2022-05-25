package stripe

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/product"
	"go.uber.org/zap"
)

func getProductMetadata(c context.Context, productID string) (metadata map[string]string, err error) {
	stripe.Key = getStripeKey()
	params := &stripe.ProductParams{}
	product, err := product.Get(productID, params)
	if err != nil {
		logger.Warn("Error getting product", zap.Error(err))
		return nil, errors.New("Product not found")
	}
	logger.Debug("Meta", zap.Any("metaData", product.Metadata))
	return product.Metadata, nil
}
