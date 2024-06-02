package mongomanager

type Seat struct {
	SubscriptionID string `bson:"subscriptionID"`
	EMail          string `bson:"email" validate:"required"`
}

type SeatFilter struct {
	SubscriptionID string `bson:"subscriptionID" index:"unique"`
}
