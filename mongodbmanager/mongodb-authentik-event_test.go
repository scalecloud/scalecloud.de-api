package mongodbmanager

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestStoreAuthentikEventIsIdempotent(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	eventUUID := "test-event-" + time.Now().UTC().Format("20060102150405.000000000")
	payload := bson.M{
		"event_uuid": eventUUID,
		"action":     "login",
		"user":       bson.M{"pk": 42, "username": "test-user"},
		"created":    time.Now().UTC(),
	}
	collection := connection.Client.Database(databaseAuthentik).Collection(collectionAuthentikEvents)
	t.Cleanup(func() {
		if _, err := collection.DeleteOne(ctx, bson.M{"_id": eventUUID}); err != nil {
			t.Errorf("delete test Authentik event: %v", err)
		}
	})

	inserted, err := connection.StoreAuthentikEvent(ctx, payload)
	if err != nil {
		t.Fatalf("store Authentik event: %v", err)
	}
	if !inserted {
		t.Fatal("first event delivery was not inserted")
	}

	inserted, err = connection.StoreAuthentikEvent(ctx, payload)
	if err != nil {
		t.Fatalf("store duplicate Authentik event: %v", err)
	}
	if inserted {
		t.Fatal("duplicate event delivery was inserted")
	}
	count, err := collection.CountDocuments(ctx, bson.M{"_id": eventUUID})
	if err != nil {
		t.Fatalf("count stored Authentik events: %v", err)
	}
	if count != 1 {
		t.Fatalf("stored %d documents for duplicate event deliveries, want 1", count)
	}
}
