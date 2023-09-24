package stripemanager

type CheckoutModelPortalRequest struct {
	ProductID string `json:"productID" binding:"required"`
	Quantity  int64  `json:"quantity" binding:"required"`
}

type CheckoutModelPortalReply struct {
	URL string `json:"url" validate:"required"`
}
