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

	databaseOpenCloud           = "opencloud"
	collectionOpenCloudHistory  = "usage-history"
	collectionOpenCloudRegistry = "resource-registry"
	collectionOpenCloudMembers  = "subscription-members"

	databaseNewsletters   = "newsletters"
	collectionSubscribers = "subscribers"
)

var databases = map[string][]string{
	databaseSubscription: {collectionSeats},
	databaseProduct:      {collectionTrial},
	databaseStripe:       {collectionUsers},
	databaseAuthentik:    {collectionAuthentikEvents},
	databaseOpenCloud:    {collectionOpenCloudHistory, collectionOpenCloudRegistry, collectionOpenCloudMembers},
	databaseNewsletters:  {collectionSubscribers},
}
