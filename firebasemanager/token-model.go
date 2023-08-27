package firebasemanager

type TokenDetails struct {
	UID   string `bson:"uid,omitempty"`
	EMail string `bson:"email,omitempty"`
}
