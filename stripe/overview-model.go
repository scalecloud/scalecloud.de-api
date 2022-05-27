package stripe

type SubscriptionOverview struct {
	ID            string `json:"id"`
	ProductName   string `json:"productName"`
	ProductType   string `json:"productType"`
	StorageAmount int    `json:"storageAmount"`
	UserCount     int64  `json:"userCount"`
}
