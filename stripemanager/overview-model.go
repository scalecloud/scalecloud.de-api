package stripemanager

type SubscriptionOverviewReply struct {
	ID            string `json:"id" validate:"required"`
	Acive         *bool  `json:"active" validate:"required"`
	ProductName   string `json:"productName" validate:"required"`
	ProductType   string `json:"productType" validate:"required"`
	StorageAmount int    `json:"storageAmount" validate:"required"`
	UserCount     int64  `json:"userCount" validate:"required"`
}
