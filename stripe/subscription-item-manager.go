package stripe

import (
	"github.com/stripe/stripe-go/v72/subitem"
	"go.uber.org/zap"
)

func getSubscriptionArtikelID() {

	subscriptionArtikelID := "si_INYwzY0bSrDTHX"

	subscriptionItem, error := subitem.Get(subscriptionArtikelID, nil)
	if error != nil {
		logger.Error("Error getting subscription item", zap.Error(error))
	}
	logger.Info("subscriptionItem", zap.Any("subscriptionItem", subscriptionItem))
}
