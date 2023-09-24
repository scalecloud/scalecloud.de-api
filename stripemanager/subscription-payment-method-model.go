package stripemanager

type SubscriptionPaymentMethodRequest struct {
	ID string `json:"id" binding:"required"`
}

type SubscriptionPaymentMethodReply struct {
	ID       string `json:"id" validate:"required"`
	Type     string `json:"type" validate:"required"`
	Brand    string `json:"brand" validate:"required"`
	Last4    string `json:"last4" validate:"required"`
	ExpMonth uint64 `json:"exp_month" validate:"required"`
	ExpYear  uint64 `json:"exp_year" validate:"required"`
}
