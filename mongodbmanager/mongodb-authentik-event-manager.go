package mongodbmanager

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

func (mongoDBConnection *MongoDBConnection) StoreAuthentikEvent(ctx context.Context, payload bson.M) (bool, error) {
	eventUUID, ok := payload["event_uuid"].(string)
	if !ok || eventUUID == "" {
		return false, fmt.Errorf("Authentik event UUID is required")
	}
	action, ok := payload["action"].(string)
	if !ok || action == "" {
		return false, fmt.Errorf("Authentik event action is required")
	}

	collection, err := mongoDBConnection.getCollection(ctx, databaseAuthentik, collectionAuthentikEvents)
	if err != nil {
		return false, err
	}
	document := bson.M{
		"_id":         eventUUID,
		"event_uuid":  eventUUID,
		"action":      action,
		"user":        payload["user"],
		"created":     payload["created"],
		"received_at": time.Now().UTC(),
		"payload":     payload,
	}
	if _, err := collection.InsertOne(ctx, document); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		mongoDBConnection.Log.Error("Error inserting Authentik event", zap.Error(err))
		return false, fmt.Errorf("error inserting Authentik event: %w", err)
	}
	return true, nil
}
