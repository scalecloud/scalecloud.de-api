package mongodbmanager

import (
	"errors"
	"time"
)

var ErrOpenCloudStorageCapacityReached = errors.New("OpenCloud storage allocation exceeds the purchased capacity")

type OpenCloudRole string

const (
	OpenCloudRoleOwner         OpenCloudRole = "Owner"
	OpenCloudRoleAdministrator OpenCloudRole = "Administrator"
	OpenCloudRoleBilling       OpenCloudRole = "Billing"
)

type OpenCloudMemberStatus string

const (
	OpenCloudMemberPending  OpenCloudMemberStatus = "pending"
	OpenCloudMemberClaiming OpenCloudMemberStatus = "claiming"
	OpenCloudMemberActive   OpenCloudMemberStatus = "active"
)

type OpenCloudMember struct {
	SubscriptionID        string                `bson:"subscriptionID" json:"subscriptionID" validate:"required"`
	UID                   string                `bson:"uid" json:"uid"`
	EMail                 string                `bson:"email" json:"email" validate:"required"`
	EMailVerified         *bool                 `bson:"emailVerified" json:"emailVerified"`
	Roles                 []OpenCloudRole       `bson:"roles" json:"roles" validate:"required"`
	Status                OpenCloudMemberStatus `bson:"status,omitempty" json:"status,omitempty"`
	InvitationTokenHash   string                `bson:"invitationTokenHash,omitempty" json:"-"`
	InvitationExpiresAt   *time.Time            `bson:"invitationExpiresAt,omitempty" json:"invitationExpiresAt,omitempty"`
	AuthentikUserPK       int64                 `bson:"authentikUserPk,omitempty" json:"-"`
	OpenCloudDriveID      string                `bson:"openCloudDriveId,omitempty" json:"-"`
	OpenCloudUserID       string                `bson:"openCloudUserId,omitempty" json:"-"`
	OpenCloudStorageBytes int64                 `bson:"openCloudStorageBytes,omitempty" json:"-"`
}
