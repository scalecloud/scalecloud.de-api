package stripe

type SubscriptionPaymentMethodRequest struct {
	ID string `json:"id"`
}

type SubscriptionPaymentMethodReply struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth uint64 `json:"exp_month"`
	ExpYear  uint64 `json:"exp_year"`
}
