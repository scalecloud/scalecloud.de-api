package stripemanager

type ChangePaymentRequest struct {
	SubscriptionID string `json:"subscriptionid" binding:"required"`
}

type ChangePaymentReply struct {
	SetupIntentID string `json:"setupintentid" validate:"required"`
	ClientSecret  string `json:"clientsecret" validate:"required"`
	EMail         string `json:"email" validate:"required"`
}
