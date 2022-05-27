package stripe

import (
	"context"
	"errors"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/plan"
	"go.uber.org/zap"
)

func getPlan(c context.Context, planID string) (*stripe.Plan, error) {
	stripe.Key = getStripeKey()
	params := &stripe.PlanParams{}
	plan, err := plan.Get(planID, params)
	if err != nil {
		logger.Warn("Error getting Plan", zap.Error(err))
		return nil, errors.New("Plan not found")
	}
	logger.Debug("Plan", zap.Any("planID", plan.ID))
	return plan, nil
}
