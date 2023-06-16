package stripe

type CheckoutSetupIntentRequest struct {
	ProductID string `json:"productID"`
	Quantity  int64  `json:"quantity"`
}

type CheckoutSetupIntentReply struct {
	SetupIntentID string `json:"setupIntentID"`
	ClientSecret  string `json:"clientSecret"`
}
