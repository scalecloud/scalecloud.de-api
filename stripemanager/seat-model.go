package stripemanager

import "github.com/scalecloud/scalecloud.de-api/mongomanager"

type ListSeatRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	PageIndex      int    `json:"pageIndex" validate:"gte=0"`
	PageSize       int    `json:"pageSize" validate:"gte=1"`
}

type ListSeatReply struct {
	SubscriptionID string   `json:"subscriptionID" validate:"required"`
	MaxSeats       int64    `json:"maxSeats" validate:"required"`
	EMails         []string `json:"emails" validate:"required"`
	PageIndex      int      `json:"pageIndex" validate:"gte=0"`
	TotalResults   int64    `json:"totalResults" validate:"gte=1"`
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
