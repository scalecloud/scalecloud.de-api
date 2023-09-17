package subscriptionmanager

type SubscriptionOverview struct {
	ID            string `json:"id"`
	Acive         bool   `json:"active"`
	ProductName   string `json:"productName"`
	ProductType   string `json:"productType"`
	StorageAmount int    `json:"storageAmount"`
	UserCount     int64  `json:"userCount"`
}
