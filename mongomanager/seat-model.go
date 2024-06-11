package mongomanager

type Role string

const (
	RoleOwner         Role = "Owner"
	RoleAdministrator Role = "Administrator"
	RoleUser          Role = "User"
	RoleBilling       Role = "Billing"
)

type Seat struct {
	SubscriptionID string `bson:"subscriptionID"`
	EMail          string `bson:"email" validate:"required"`
	Roles          []Role `bson:"roles" validate:"required"`
}
