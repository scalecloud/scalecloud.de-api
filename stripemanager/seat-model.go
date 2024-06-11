package stripemanager

import "github.com/scalecloud/scalecloud.de-api/mongomanager"

type ListSeatRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
}

type ListSeatReply struct {
	SubscriptionID string   `json:"subscriptionID" validate:"required"`
	MaxSeats       int64    `json:"max_seats" validate:"required"`
	EMails         []string `json:"emails" validate:"required"`
}

type AddSeatRequest struct {
	SubscriptionID string              `json:"subscriptionID" validate:"required"`
	EMail          string              `json:"email" validate:"required"`
	Roles          []mongomanager.Role `json:"roles" validate:"required"`
}

type AddSeatReply struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	Success        bool   `json:"success" validate:"required"`
	EMail          string `json:"email" validate:"required"`
}

type RemoveSeatRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	EMail          string `json:"email" validate:"required"`
}

type RemoveSeatReply struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	Success        bool   `json:"success" validate:"required"`
	EMail          string `json:"email" validate:"required"`
}
