package stripemanager

type SubscriptionDetailReply struct {
	ID                string `json:"id" validate:"required"`
	Active            *bool  `json:"active" validate:"required"`
	ProductName       string `json:"product_name" validate:"required"`
	ProductType       string `json:"product_type" validate:"required"`
	StorageAmount     int    `json:"storage_amount" validate:"required"`
	UserCount         int64  `json:"user_count" validate:"required"`
	PricePerMonth     int64  `json:"price_per_month" validate:"required"`
	Currency          string `json:"currency" validate:"required"`
	CancelAtPeriodEnd *bool  `json:"cancel_at_period_end" validate:"required"`
	CancelAt          int64  `json:"cancel_at"`
	Status            string `json:"status" validate:"required"`
	TrialEnd          int64  `json:"trial_end"`
}
