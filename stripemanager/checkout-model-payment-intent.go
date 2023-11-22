package stripemanager

type CheckoutPaymentIntentRequest struct {
	ProductID string `json:"productID" binding:"required"`
	Quantity  int64  `json:"quantity" binding:"required"`
}

type CheckoutPaymentIntentReply struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	ClientSecret   string `json:"clientSecret" validate:"required"`
	Quantity       int64  `json:"quantity" validate:"required"`
	EMail          string `json:"email" validate:"required"`
}

type CheckoutPaymentIntentUpdateRequest struct {
	SubscriptionID string `json:"subscriptionID" binding:"required"`
	Quantity       int64  `json:"quantity" binding:"required"`
}

type CheckoutPaymentIntentUpdateReply struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	ClientSecret   string `json:"clientSecret" validate:"required"`
	Quantity       int64  `json:"quantity" validate:"required"`
}

type CheckoutProductRequest struct {
	ProductID string `json:"productID" binding:"required"`
}

type CheckoutProductReply struct {
	ProductID     string `json:"productID" validate:"required"`
	Name          string `json:"name" validate:"required"`
	StorageAmount int64  `json:"storageAmount" validate:"required"`
	StorageUnit   string `json:"storageUnit" validate:"required"`
	TrialDays     int64  `json:"trialDays" validate:"required"`
	PricePerMonth int64  `json:"pricePerMonth" validate:"required"`
	Currency      string `json:"currency" validate:"required"`
}
