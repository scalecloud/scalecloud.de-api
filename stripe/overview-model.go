package stripe

type SubscriptionOverview struct {
	ID              string `json:"id"`
	PlanProductName string `json:"planProductName"`
	ProductName     string `json:"productName"`
	StorageAmount   int    `json:"storageAmount"`
	UserCount       int64  `json:"userCount"`
}
