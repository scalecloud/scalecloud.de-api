package stripemanager

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/product"
)

func (stripeConnection *StripeConnection) GetProduct(c context.Context, productID string) (*stripe.Product, error) {
	stripe.Key = stripeConnection.Key
	params := &stripe.ProductParams{}
	product, err := product.Get(productID, params)
	if err != nil {
		return nil, errors.New("Product not found")
	}
	return product, nil
}
