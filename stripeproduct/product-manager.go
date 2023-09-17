package stripeproduct

import (
	"context"
	"errors"

	"github.com/scalecloud/scalecloud.de-api/stripesecret"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/product"
	"go.uber.org/zap"
)

var logger, _ = zap.NewProduction()

func GetProduct(c context.Context, productID string) (*stripe.Product, error) {
	stripe.Key = stripesecret.GetStripeKey()
	params := &stripe.ProductParams{}
	product, err := product.Get(productID, params)
	if err != nil {
		logger.Warn("Error getting product", zap.Error(err))
		return nil, errors.New("Product not found")
	}
	logger.Debug("Product", zap.Any("productID", product.ID))
	return product, nil
}
