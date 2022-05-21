package stripe

type SubscriptionOverview struct {
	ID                    string  `json:"id"`
	Title                 string  `json:"title"`
	SubscriptionArticelID string  `json:"subscriptionArticelID"`
	PricePerMonth         float64 `json:"pricePerMonth"`
	Started               string  `json:"started"`
	EndsOn                string  `json:"endsOn"`
}
