package stripemanager

type CheckoutCreateSubscriptionRequest struct {
	ProductID string `json:"productID" binding:"required"`
	Quantity  int64  `json:"quantity" binding:"required"`
}

type CheckoutCreateSubscriptionReply struct {
	Status         string `json:"status" validate:"required"`
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	ProductName    string `json:"productName" validate:"required"`
	EMail          string `json:"email" validate:"required"`
	TrialEnd       int64  `json:"trialEnd" validate:"gte=0"`
}

type CheckoutProductRequest struct {
	ProductID string `json:"productID" binding:"required"`
}

type CheckoutProductReply struct {
	ProductID             string `json:"productID" validate:"required"`
	Name                  string `json:"name" validate:"required"`
	StorageAmount         int64  `json:"storageAmount" validate:"required"`
	StorageUnit           string `json:"storageUnit" validate:"required"`
	TrialDays             int64  `json:"trialDays" validate:"required"`
	PricePerMonth         int64  `json:"pricePerMonth" validate:"required"`
	Currency              string `json:"currency" validate:"required"`
	HasValidPaymentMethod *bool  `json:"has_valid_payment_method" validate:"required"`
}
