package stripe

type CheckoutModelPortalRequest struct {
	ProductID string `json:"productID"`
	Quantity  int64  `json:"quantity"`
}

type CheckoutModelPortalReply struct {
	URL string `json:"url"`
}