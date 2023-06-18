package stripe

type ChangeSubscriptionPaymentMethodRequest struct {
	SubscriptionID string `json:"id"`
}

type ChangeSubscriptionPaymentMethodReply struct {
	SetupIntentID string `json:"id"`
	Secret        string `json:"secret"`
}
