package stripemanager

type SubscriptionResumeRequest struct {
	ID string `json:"id"`
}

type SubscriptionResumeReply struct {
	ID                string `json:"id"`
	CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
}
