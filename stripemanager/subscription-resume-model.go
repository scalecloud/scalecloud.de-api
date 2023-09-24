package stripemanager

type SubscriptionResumeRequest struct {
	ID string `json:"id" binding:"required"`
}

type SubscriptionResumeReply struct {
	ID                string `json:"id" validate:"required"`
	CancelAtPeriodEnd *bool  `json:"cancel_at_period_end" validate:"required"`
}
