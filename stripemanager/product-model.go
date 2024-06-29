package stripemanager

type ProductType string

const (
	ProductNextcloud ProductType = "Nextcloud"
	ProductSynology  ProductType = "Synology"
)

type ProductTiersRequest struct {
	ProductType ProductType `json:"productType" validate:"required"`
}

type ProductTiersReply struct {
	ProductType  ProductType   `json:"productType" validate:"required"`
	ProductTiers []ProductTier `json:"productTiers" validate:"required"`
}

type ProductTier struct {
	ProductType   ProductType `json:"productType" validate:"required"`
	ProductID     string      `json:"productID" validate:"required"`
	Name          string      `json:"name" validate:"required"`
	StorageAmount int         `json:"storageAmount" validate:"required"`
	StorageUnit   string      `json:"storageUnit" validate:"required"`
	TrialDays     int64       `json:"trialDays" validate:"required"`
	PricePerMonth int64       `json:"pricePerMonth" validate:"required"`
}
