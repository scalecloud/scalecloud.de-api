package mongodbmanager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func (mongoDBConnection *MongoDBConnection) StoreAuthentikEvent(ctx context.Context, payload bson.M) (bool, error) {
	eventUUID, ok := payload["event_uuid"].(string)
	if !ok || eventUUID == "" {
		return false, errors.New("Authentik event UUID is required")
	}
	applicationName, _ := payload["authorized_application"].(string)
	if !strings.EqualFold(applicationName, "OpenCloud") {
		return false, nil
	}
	userPK, err := getInt64FieldFromPayload(payload, "user_pk")
	if err != nil {
		return false, errors.New("Authentik user ID is required")
	}
	if _, ok := payload["created"]; !ok {
		return false, errors.New("Authentik event created timestamp is required")
	}

	collection, err := mongoDBConnection.getCollection(ctx, databaseAuthentik, collectionAuthentikEvents)
	if err != nil {
		return false, err
	}
	documentID := fmt.Sprintf("authentik-opencloud-user-%d", userPK)
	document := bson.M{
		"_id":                    documentID,
		"event_uuid":             eventUUID,
		"user_pk":                userPK,
		"user_email":             payload["user_email"],
		"authorized_application": applicationName,
		"created":                payload["created"],
		"received_at":            time.Now().UTC(),
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

func getInt64FieldFromPayload(doc bson.M, key string) (int64, error) {
	value := doc[key]
	switch v := value.(type) {
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("missing or invalid %s", key)
	}
}

func (mongoDBConnection *MongoDBConnection) FindFirstOpenCloudAuthorizationByUser(ctx context.Context, userPK int64) (bson.M, error) {
	collection, err := mongoDBConnection.getCollection(ctx, databaseAuthentik, collectionAuthentikEvents)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"authorized_application": "OpenCloud",
		"user_pk":                userPK,
	}
	var event bson.M
	err = collection.FindOne(ctx, filter, options.FindOne().SetSort(bson.D{{Key: "created", Value: 1}})).Decode(&event)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("error querying first Authentik authorization event: %w", err)
	}
	return event, nil
}
