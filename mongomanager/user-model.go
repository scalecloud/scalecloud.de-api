package mongomanager

import "time"

type User struct {
	UID                        string     `bson:"uid" validate:"required"`
	CustomerID                 string     `bson:"customerID,omitempty"`
	ProvisioningToken          string     `bson:"provisioningToken,omitempty"`
	ProvisioningLeaseExpiresAt *time.Time `bson:"provisioningLeaseExpiresAt,omitempty"`
}
