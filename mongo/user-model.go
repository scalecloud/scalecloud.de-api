package mongo

type User struct {
	UID        string `bson:"uid,omitempty"`
	CustomerID string `bson:"customerID,omitempty"`
}
