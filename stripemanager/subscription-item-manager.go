package stripemanager

import (
	"context"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/subscriptionitem"
)

func updateSubscriptionItem(c context.Context, subscriptionItemID string, quantity int64) (*stripe.SubscriptionItem, error) {
	params := &stripe.SubscriptionItemParams{
		Quantity: stripe.Int64(quantity),
	}
	si, err := subscriptionitem.Update(
		subscriptionItemID,
		params,
	)
	if err != nil {
		return nil, err
	}
	return si, nil
}
