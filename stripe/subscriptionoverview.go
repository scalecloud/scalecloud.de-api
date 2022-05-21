package stripe

type SubscriptionOverview struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	ProductName   string `json:"productName"`
	StorageAmount int    `json:"storageAmount"`
	UserCount     int    `json:"userCount"`
}
