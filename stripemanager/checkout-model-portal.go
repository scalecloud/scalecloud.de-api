package stripemanager

type CheckoutModelPortalRequest struct {
	ProductID string `validate:"required"`
	Quantity  int64  `validate:"required"`
}

type CheckoutModelPortalReply struct {
	URL string `json:"url"`
}
