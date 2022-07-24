package stripe

type CheckoutModelPortalRequest struct {
	ProductID string `json:"productID"`
	Quantity  int64  `json:"quantity"`
}

type CheckoutModelPortalReturn struct {
	URL string `json:"url"`
}
