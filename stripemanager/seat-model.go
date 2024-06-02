package stripemanager

type ListSeatRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
}

type ListSeatReply struct {
	SubscriptionID string   `json:"subscriptionID" validate:"required"`
	ProductName    string   `json:"product_name" validate:"required"`
	ProductType    string   `json:"product_type" validate:"required"`
	MaxSeats       int64    `json:"max_seats" validate:"required"`
	EMails         []string `json:"emails" validate:"required"`
}

type AddSeatRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	Email          string `json:"email" validate:"required"`
}

type AddSeatReply struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	Success        bool   `json:"success" validate:"required"`
	Email          string `json:"email" validate:"required"`
}

type RemoveSeatRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	Email          string `json:"email" validate:"required"`
}

type RemoveSeatReply struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	Success        bool   `json:"success" validate:"required"`
	Email          string `json:"email" validate:"required"`
}
