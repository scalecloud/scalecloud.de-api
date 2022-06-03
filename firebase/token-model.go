package firebase

type TokenDetails struct {
	UID   string `bson:"uid,omitempty"`
	Email string `bson:"email,omitempty"`
}
