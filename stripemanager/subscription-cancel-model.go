package stripemanager

type SubscriptionCancelRequest struct {
	SubscriptionID string `json:"subscriptionID" binding:"required"`
}

type SubscriptionCancelReply struct {
	SubscriptionID    string `json:"subscriptionID" validate:"required"`
	CancelAtPeriodEnd *bool  `json:"cancel_at_period_end" validate:"required"`
	CancelAt          int64  `json:"cancel_at" validate:"gte=0"`
}
