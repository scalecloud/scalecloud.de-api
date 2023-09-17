package subscriptionmanager

type SubscriptionDetail struct {
	ID                string `json:"id"`
	Active            bool   `json:"active"`
	ProductName       string `json:"product_name"`
	ProductType       string `json:"product_type"`
	StorageAmount     int    `json:"storage_amount"`
	UserCount         int64  `json:"user_count"`
	PricePerMonth     int64  `json:"price_per_month"`
	Currency          string `json:"currency"`
	CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
	CancelAt          int64  `json:"cancel_at"`
	Status            string `json:"status"`
	TrialEnd          int64  `json:"trial_end"`
}
