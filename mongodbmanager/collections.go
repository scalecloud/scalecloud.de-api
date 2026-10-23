package mongodbmanager

const (
	databaseSubscription = "subscription"
	collectionSeats      = "seats"
	collectionSeatLocks  = "seat-reservation-locks"

	databaseProduct = "product"
	collectionTrial = "trial"

	databaseStripe  = "stripe"
	collectionUsers = "users"

	databaseAuthentik         = "authentik"
	collectionAuthentikEvents = "events"

	databaseNewsletters   = "newsletters"
	collectionSubscribers = "subscribers"
)

var databases = map[string][]string{
	databaseSubscription: {collectionSeats},
	databaseProduct:      {collectionTrial},
	databaseStripe:       {collectionUsers},
	databaseAuthentik:    {collectionAuthentikEvents},
	databaseNewsletters:  {collectionSubscribers},
}
