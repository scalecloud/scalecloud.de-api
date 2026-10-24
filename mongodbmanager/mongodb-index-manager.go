package mongodbmanager

import "context"

func (mongoDBConnection *MongoDBConnection) EnsureIndexes(ctx context.Context) error {
	err := mongoDBConnection.ensureSeatIndex(ctx)
	if err != nil {
		return err
	}
	err = mongoDBConnection.ensureSeatInvitationTokenIndex(ctx)
	if err != nil {
		return err
	}
	err = mongoDBConnection.ensureNewsletterIndex(ctx)
	if err != nil {
		return err
	}
	err = mongoDBConnection.ensureStripeUserUIDIndex(ctx)
	if err != nil {
		return err
	}
	err = mongoDBConnection.ensureAuthentikEventIndex(ctx)
	if err != nil {
		return err
	}
	err = mongoDBConnection.ensureOpenCloudUsageIndexes(ctx)
	if err != nil {
		return err
	}
	err = mongoDBConnection.ensureOpenCloudRegistryIndexes(ctx)
	if err != nil {
		return err
	}
	mongoDBConnection.Log.Info("all required indexes are present")
	return nil
}
