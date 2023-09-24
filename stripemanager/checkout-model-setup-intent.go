package stripemanager

type CheckoutSetupIntentRequest struct {
	ProductID string `json:"productID" binding:"required"`
	Quantity  int64  `json:"quantity" binding:"required"`
}

type CheckoutSetupIntentReply struct {
	SetupIntentID string `json:"setupIntentID" validate:"required"`
	ClientSecret  string `json:"clientSecret" validate:"required"`
}
