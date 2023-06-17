package stripe

type SubscriptionCancelRequest struct {
	ID string `json:"id"`
}

type SubscriptionCancelReply struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	CancelAt int64  `json:"cancel_at"`
}
