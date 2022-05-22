package stripe

import (
	"context"
	"errors"
)

func GetSubscriptionByID(c context.Context, id, customerID string) (subscriptionDetail SubscriptionDetail, err error) {
	for _, sub := range subscriptionDetailPlaceholder {
		if sub.ID == id {
			return sub, nil
		}
	}
	return SubscriptionDetail{}, errors.New("Subscription not found")
}
