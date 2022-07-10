package stripe

type CheckoutSubscriptionModel struct {
	SubscriptionID string `json:"subscriptionId"`
	ClientSecret   string `json:"clientSecret"`
}
