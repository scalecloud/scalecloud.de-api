package stripe

type Subscription struct {
	ID                    string  `json:"id"`
	Title                 string  `json:"title"`
	SubscriptionArticelID string  `json:"artist"`
	PricePerMonth         float64 `json:"pricepermonth"`
	Started               string  `json:"Started"`
	EndsOn                string  `json:"EndsOn"`
}
