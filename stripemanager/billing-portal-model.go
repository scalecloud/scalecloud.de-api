package stripemanager

type BillingPortalReply struct {
	URL string `json:"url" validate:"required"`
}
