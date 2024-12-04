package stripemanager

type Address struct {
	City       string  `json:"city" validate:"required"`
	Country    string  `json:"country" validate:"required"`
	Line1      string  `json:"line1" validate:"required"`
	Line2      *string `json:"line2" validate:"required"`
	PostalCode string  `json:"postal_code" validate:"required"`
	State      string  `json:"state" validate:"required"`
}

type BillingAddressRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
}

type BillingAddressReply struct {
	SubscriptionID string  `json:"subscriptionID" validate:"required"`
	Name           string  `json:"name" validate:"required"`
	Address        Address `json:"address" validate:"required"`
	Phone          string  `json:"phone" validate:"required"`
}

type UpdateBillingAddressRequest struct {
	SubscriptionID string  `json:"subscriptionID" validate:"required"`
	Name           string  `json:"name" validate:"required"`
	Address        Address `json:"address" validate:"required"`
	Phone          string  `json:"phone" validate:"required"`
}

type UpdateBillingAddressReply struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
}
