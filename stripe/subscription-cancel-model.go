package stripe

type SubscriptionCancelRequest struct {
	ID string `json:"id"`
}

type SubscriptionCancelReply struct {
	ID                string `json:"id"`
	CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
	CancelAt          int64  `json:"cancel_at"`
}
