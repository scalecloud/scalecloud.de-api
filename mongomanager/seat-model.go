package mongomanager

import "time"

type Role string

const (
	RoleOwner         Role = "Owner"
	RoleAdministrator Role = "Administrator"
	RoleUser          Role = "User"
	RoleBilling       Role = "Billing"
)

type SeatStatus string

const (
	SeatStatusPending  SeatStatus = "pending"
	SeatStatusClaiming SeatStatus = "claiming"
	SeatStatusActive   SeatStatus = "active"
)

type Seat struct {
	SubscriptionID      string     `bson:"subscriptionID" json:"subscriptionID" validate:"required"`
	UID                 string     `bson:"uid" json:"uid"`
	EMail               string     `bson:"email" json:"email" validate:"required"`
	EMailVerified       *bool      `bson:"emailVerified" json:"emailVerified"`
	Roles               []Role     `bson:"roles" json:"roles" validate:"required"`
	Status              SeatStatus `bson:"status,omitempty" json:"status,omitempty"`
	InvitationTokenHash string     `bson:"invitationTokenHash,omitempty" json:"-"`
	InvitationExpiresAt *time.Time `bson:"invitationExpiresAt,omitempty" json:"invitationExpiresAt,omitempty"`
	AuthentikUserPK     int64      `bson:"authentikUserPk,omitempty" json:"-"`
}
