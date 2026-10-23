package mongodbmanager

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func (mongoDBConnection *MongoDBConnection) ensureAuthentikEventIndex(ctx context.Context) error {
	collection, err := mongoDBConnection.getCollection(ctx, databaseAuthentik, collectionAuthentikEvents)
	if err != nil {
		return err
	}
	_, err = collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "action", Value: 1},
			{Key: "context.authorized_application", Value: 1},
			{Key: "user.pk", Value: 1},
			{Key: "created", Value: 1},
		},
		Options: options.Index().SetName("AuthentikActionUserCreated"),
	})
	if err != nil {
		mongoDBConnection.Log.Error("Error creating Authentik event lookup index", zap.Error(err))
		return err
	}
	return nil
}
