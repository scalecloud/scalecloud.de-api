package stripemanager

import "github.com/scalecloud/scalecloud.de-api/mongodbmanager"

type ListSeatRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	PageIndex      int    `json:"pageIndex" validate:"gte=0"`
	PageSize       int    `json:"pageSize" validate:"gte=1"`
}

type ListSeatReply struct {
	SubscriptionID string              `json:"subscriptionID" validate:"required"`
	MaxSeats       int64               `json:"maxSeats" validate:"required"`
	Seats          []mongodbmanager.Seat `json:"seats" validate:"required"`
	PageIndex      int                 `json:"pageIndex" validate:"gte=0"`
	TotalResults   int64               `json:"totalResults" validate:"gte=1"`
}

type AddSeatRequest struct {
	SubscriptionID string              `json:"subscriptionID" validate:"required"`
	EMail          string              `json:"email" validate:"required"`
	Roles          []mongodbmanager.Role `json:"roles" validate:"required"`
}

type AddSeatReply struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	Success        bool   `json:"success" validate:"required"`
	EMail          string `json:"email" validate:"required"`
}

type SeatInvitationPreviewReply struct {
	ProductName   string              `json:"productName"`
	StorageAmount int64               `json:"storageAmount"`
	StorageUnit   string              `json:"storageUnit"`
	EMail         string              `json:"email"`
	Roles         []mongodbmanager.Role `json:"roles"`
	ExpiresAt     int64               `json:"expiresAt"`
}

type SeatInvitationClaimReply struct {
	SubscriptionID string              `json:"subscriptionID"`
	ProductName    string              `json:"productName"`
	StorageAmount  int64               `json:"storageAmount"`
	StorageUnit    string              `json:"storageUnit"`
	Roles          []mongodbmanager.Role `json:"roles"`
	Success        bool                `json:"success"`
}

type DeleteSeatRequest struct {
	SeatToDelete mongodbmanager.Seat `json:"seatToDelete" validate:"required"`
}

type DeleteSeatReply struct {
	DeletedSeat mongodbmanager.Seat `json:"deletedSeat" validate:"required"`
	Success     bool              `json:"success" validate:"required"`
}

type SeatDetailRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
	UID            string `json:"uid" validate:"required"`
}

type SeatDetailReply struct {
	SelectedSeat mongodbmanager.Seat `json:"selectedSeat" validate:"required"`
	MySeat       mongodbmanager.Seat `json:"mySeat" validate:"required"`
}

type UpdateSeatDetailRequest struct {
	SeatUpdated mongodbmanager.Seat `json:"seatUpdated" validate:"required"`
}

type UpdateSeatDetailReply struct {
	Seat mongodbmanager.Seat `json:"seat" validate:"required"`
}

type PermissionRequest struct {
	SubscriptionID string `json:"subscriptionID" validate:"required"`
}

type PermissionReply struct {
	MySeat mongodbmanager.Seat `json:"mySeat" validate:"required"`
}
