package mongomanager

type Seat struct {
	SubscriptionID string `bson:"subscriptionID"`
	EMail          string `bson:"email" validate:"required"`
}
