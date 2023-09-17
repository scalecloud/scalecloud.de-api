package stripechangepayment

type ChangePaymentRequest struct {
	SubscriptionID string `json:"subscriptionid"`
}

type ChangePaymentReply struct {
	SetupIntentID string `json:"setupintentid"`
	ClientSecret  string `json:"clientsecret"`
	EMail         string `json:"email"`
}
