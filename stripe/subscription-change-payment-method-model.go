package stripe

type ChangeSubscriptionPaymentMethodRequest struct {
	SubscriptionID string `json:"subscriptionid"`
}

type ChangeSubscriptionPaymentMethodReply struct {
	SetupIntentID string `json:"setupintentid"`
	Secret        string `json:"secret"`
}
