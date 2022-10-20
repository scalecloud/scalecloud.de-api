package stripe

type CheckoutIntegrationRequest struct {
	ProductID string `json:"productID"`
	Quantity  int64  `json:"quantity"`
}

type CheckoutIntegrationReply struct {
	SubscriptionID string `json:"subscriptionId"`
	ClientSecret   string `json:"clientSecret"`
	Quantity       int64  `json:"quantity"`
}

type CheckoutIntegrationUpdateRequest struct {
	SubscriptionID string `json:"subscriptionId"`
	Quantity       int64  `json:"quantity"`
}

type CheckoutIntegrationUpdateReply struct {
	SubscriptionID string `json:"subscriptionId"`
	ClientSecret   string `json:"clientSecret"`
	Quantity       int64  `json:"quantity"`
}

type CheckoutProductRequest struct {
	SubscriptionID string `json:"subscriptionId"`
}

type CheckoutProductReply struct {
	SubscriptionID string `json:"subscriptionId"`
	ProductID      string `json:"productID"`
	Name           string `json:"name"`
	StorageAmount  int64  `json:"storageAmount"`
	StorageUnit    string `json:"storageUnit"`
	TrialDays      int64  `json:"trialDays"`
	PricePerMonth  int64  `json:"pricePerMonth"`
}
