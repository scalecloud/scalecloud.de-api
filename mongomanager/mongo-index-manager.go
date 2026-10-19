package mongomanager

import "context"

func (mongoConnection *MongoConnection) EnsureIndexes(ctx context.Context) error {
	err := mongoConnection.ensureSeatIndex(ctx)
	if err != nil {
		return err
	}
	err = mongoConnection.ensureSeatInvitationTokenIndex(ctx)
	if err != nil {
		return err
	}
	err = mongoConnection.ensureNewsletterIndex(ctx)
	if err != nil {
		return err
	}
	err = mongoConnection.ensureStripeUserUIDIndex(ctx)
	if err != nil {
		return err
	}
	mongoConnection.Log.Info("all required indexes are present")
	return nil
}
