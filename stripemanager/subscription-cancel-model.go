package stripemanager

type SubscriptionCancelRequest struct {
	ID string `json:"id" binding:"required"`
}

type SubscriptionCancelReply struct {
	ID                string `json:"id" validate:"required"`
	CancelAtPeriodEnd *bool  `json:"cancel_at_period_end" validate:"required"`
	CancelAt          int64  `json:"cancel_at"`
}
