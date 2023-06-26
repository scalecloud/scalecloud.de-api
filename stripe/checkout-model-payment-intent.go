package stripe

type CheckoutPaymentIntentRequest struct {
	ProductID string `json:"productID"`
	Quantity  int64  `json:"quantity"`
}

type CheckoutPaymentIntentReply struct {
	SubscriptionID string `json:"subscriptionID"`
	ClientSecret   string `json:"clientSecret"`
	Quantity       int64  `json:"quantity"`
	EMail          string `json:"email"`
}

type CheckoutPaymentIntentUpdateRequest struct {
	SubscriptionID string `json:"subscriptionID"`
	Quantity       int64  `json:"quantity"`
}

type CheckoutPaymentIntentUpdateReply struct {
	SubscriptionID string `json:"subscriptionID"`
	ClientSecret   string `json:"clientSecret"`
	Quantity       int64  `json:"quantity"`
}

type CheckoutProductRequest struct {
	SubscriptionID string `json:"subscriptionID"`
}

type CheckoutProductReply struct {
	SubscriptionID string `json:"subscriptionID"`
	ProductID      string `json:"productID"`
	Name           string `json:"name"`
	StorageAmount  int64  `json:"storageAmount"`
	StorageUnit    string `json:"storageUnit"`
	TrialDays      int64  `json:"trialDays"`
	PricePerMonth  int64  `json:"pricePerMonth"`
	Currency       string `json:"currency"`
}
