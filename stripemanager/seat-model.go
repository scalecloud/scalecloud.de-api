package stripemanager

import "github.com/scalecloud/scalecloud.de-api/mongomanager"

type ListSeatRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	PageIndex      int    `json:"pageIndex" validate:"gte=0"`
	PageSize       int    `json:"pageSize" validate:"gte=1"`
}

type ListSeatReply struct {
	SubscriptionID string              `json:"subscriptionID" validate:"required"`
	MaxSeats       int64               `json:"maxSeats" validate:"required"`
	Seats          []mongomanager.Seat `json:"seats" validate:"required"`
	PageIndex      int                 `json:"pageIndex" validate:"gte=0"`
	TotalResults   int64               `json:"totalResults" validate:"gte=1"`
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

type DeleteSeatRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	EMail          string `json:"email" validate:"required"`
}

type DeleteSeatReply struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	Success        bool   `json:"success" validate:"required"`
	EMail          string `json:"email" validate:"required"`
}

type SeatDetailRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	UID            string `json:"uid" validate:"required"`
}

type SeatDetailReply struct {
	SelectedSeat mongomanager.Seat `json:"selectedSeat" validate:"required"`
	MySeat       mongomanager.Seat `json:"mySeat" validate:"required"`
}

type UpdateSeatDetailRequest struct {
	SeatUpdated mongomanager.Seat `json:"seatUpdated" validate:"required"`
}

type UpdateSeatDetailReply struct {
	Seat mongomanager.Seat `json:"seat" validate:"required"`
}

type PermissionRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
}

type PermissionReply struct {
	MySeat mongomanager.Seat `json:"mySeat" validate:"required"`
}
