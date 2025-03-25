package mongomanager

const (
	databaseSubscription = "subscription"
	collectionSeats      = "seats"

	databaseProduct = "product"
	collectionTrial = "trial"

	databaseStripe  = "stripe"
	collectionUsers = "users"

	databaseNewsletters   = "newsletters"
	collectionSubscribers = "subscribers"
)

var databases = map[string][]string{
	databaseSubscription: {collectionSeats},
	databaseProduct:      {collectionTrial},
	databaseStripe:       {collectionUsers},
	databaseNewsletters:  {collectionSubscribers},
}
