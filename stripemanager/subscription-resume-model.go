package stripemanager

type SubscriptionResumeRequest struct {
	SubscriptionID string `json:"subscriptionID" binding:"required"`
}

type SubscriptionResumeReply struct {
	SubscriptionID    string `json:"subscriptionID" validate:"required"`
	CancelAtPeriodEnd *bool  `json:"cancel_at_period_end" validate:"required"`
}
