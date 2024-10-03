package stripemanager

import (
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/subscriptionitem"
)

func updateSubscriptionItem(subscriptionItemID string, quantity int64) (*stripe.SubscriptionItem, error) {
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
