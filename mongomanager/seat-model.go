package mongomanager

type Role string

const (
	RoleOwner         Role = "Owner"
	RoleAdministrator Role = "Administrator"
	RoleUser          Role = "User"
	RoleBilling       Role = "Billing"
)

type Seat struct {
	SubscriptionID string `bson:"subscriptionID" json:"subscriptionID" validate:"required"`
	UID            string `bson:"uid" json:"uid" validate:"required"`
	EMail          string `bson:"email" json:"email" validate:"required"`
	EMailVerified  *bool  `bson:"emailVerified" json:"emailVerified" validate:"required"`
	Roles          []Role `bson:"roles" json:"roles" validate:"required"`
}
