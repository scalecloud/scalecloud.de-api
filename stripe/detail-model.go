package stripe

type SubscriptionDetail struct {
	ID                string `json:"id"`
	Active            bool   `json:"active"`
	ProductName       string `json:"productName"`
	ProductType       string `json:"productType"`
	StorageAmount     int    `json:"storageAmount"`
	UserCount         int64  `json:"userCount"`
	PricePerMonth     int64  `json:"pricePerMonth"`
	Currency          string `json:"currency"`
	CancelAtPeriodEnd bool   `json:"cancelAtPeriodEnd"`
	CancelAt          int64  `json:"cancelAt"`
}
