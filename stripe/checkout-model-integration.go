package stripe

type CheckoutIntegrationRequest struct {
	ProductID string `json:"productID"`
	Quantity  int64  `json:"quantity"`
}

type CheckoutIntegrationReturn struct {
	SubscriptionID string `json:"subscriptionId"`
	ClientSecret   string `json:"clientSecret"`
	Quantity       int64  `json:"quantity"`
}

type CheckoutIntegrationUpdateRequest struct {
	SubscriptionID string `json:"subscriptionId"`
	Quantity       int64  `json:"quantity"`
}

type CheckoutIntegrationUpdateReturn struct {
	SubscriptionID string `json:"subscriptionId"`
	ClientSecret   string `json:"clientSecret"`
	Quantity       int64  `json:"quantity"`
}
