package mongomanager

type User struct {
	UID        string `bson:"uid" validate:"required"`
	CustomerID string `bson:"customerID" validate:"required"`
}
