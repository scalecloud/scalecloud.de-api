package stripe

import (
	"context"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/subitem"
)

func updateSubscriptionItem(c context.Context, subscriptionItemID string, quantity int64) (*stripe.SubscriptionItem, error) {
	params := &stripe.SubscriptionItemParams{
		Quantity: stripe.Int64(quantity),
	}
	si, err := subitem.Update(
		subscriptionItemID,
		params,
	)
	if err != nil {
		return nil, err
	}
	return si, nil
}
