package stripe

type SubscriptionDetail struct {
	ID                    string  `json:"id"`
	PlanProductName       string  `json:"planProductName"`
	SubscriptionArticelID string  `json:"subscriptionArticelID"`
	PricePerMonth         float64 `json:"pricePerMonth"`
	Started               string  `json:"started"`
	EndsOn                string  `json:"endsOn"`
}
