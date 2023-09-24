package firebasemanager

type TokenDetails struct {
	UID   string `json:"uid" validate:"required"`
	EMail string `json:"email" validate:"required"`
}
