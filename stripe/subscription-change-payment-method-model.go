package stripe

type SubscriptionSetupIntentRequest struct {
	SubscriptionID string `json:"subscriptionid"`
}

type SubscriptionSetupIntentReply struct {
	SetupIntentID string `json:"setupintentid"`
	Secret        string `json:"secret"`
}
