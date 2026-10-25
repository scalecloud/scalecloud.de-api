package mongodbmanager

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestStoreAuthentikEventRejectsMissingRequiredFields(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)

	t.Run("missing event uuid", func(t *testing.T) {
		payload := bson.M{
			"user_pk":                42,
			"authorized_application": "OpenCloud",
			"created":                time.Now().UTC(),
		}
		inserted, err := connection.StoreAuthentikEvent(ctx, payload)
		if err == nil || !strings.Contains(err.Error(), "event UUID") {
			t.Fatalf("StoreAuthentikEvent() error = %v, want event UUID validation", err)
		}
		if inserted {
			t.Fatal("missing event UUID should not insert")
		}
	})

	t.Run("missing created timestamp", func(t *testing.T) {
		payload := bson.M{
			"event_uuid":             "event-without-created",
			"user_pk":                42,
			"authorized_application": "OpenCloud",
		}
		inserted, err := connection.StoreAuthentikEvent(ctx, payload)
		if err == nil || !strings.Contains(err.Error(), "created timestamp") {
			t.Fatalf("StoreAuthentikEvent() error = %v, want created timestamp validation", err)
		}
		if inserted {
			t.Fatal("missing created timestamp should not insert")
		}
	})
}

func TestStoreAuthentikEventIsIdempotent(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	eventUUID := "test-event-" + time.Now().UTC().Format("20060102150405.000000000")
	payload := bson.M{
		"event_uuid":             eventUUID,
		"user_pk":                42,
		"user_email":             "test-user@example.com",
		"authorized_application": "OpenCloud",
		"created":                time.Now().UTC(),
		"raw_payload":            bson.M{"secret": "must-not-be-stored"},
	}
	collection := connection.Client.Database(databaseAuthentik).Collection(collectionAuthentikEvents)
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := collection.DeleteOne(cleanupContext, bson.M{"event_uuid": eventUUID}); err != nil {
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
	count, err := collection.CountDocuments(ctx, bson.M{"event_uuid": eventUUID})
	if err != nil {
		t.Fatalf("count stored Authentik events: %v", err)
	}
	if count != 1 {
		t.Fatalf("stored %d documents for duplicate event deliveries, want 1", count)
	}
	var storedEvent bson.M
	if err := collection.FindOne(ctx, bson.M{"event_uuid": eventUUID}).Decode(&storedEvent); err != nil {
		t.Fatalf("read stored Authentik event: %v", err)
	}
	if _, exists := storedEvent["raw_payload"]; exists {
		t.Fatal("stored event contains the raw payload")
	}
	if _, exists := storedEvent["context"]; exists {
		t.Fatal("stored event contains the old nested context payload")
	}
}

func TestFindFirstOpenCloudAuthorizationByUser(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	collection := connection.Client.Database(databaseAuthentik).Collection(collectionAuthentikEvents)
	userPK := int64(999)
	firstEventUUID := "first-opencloud-auth-" + time.Now().UTC().Format("20060102150405.000000000")
	secondEventUUID := "second-opencloud-auth-" + time.Now().UTC().Format("20060102150405.000000001")

	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := collection.DeleteMany(cleanupContext, bson.M{"user_pk": userPK}); err != nil {
			t.Errorf("delete test Authentik events: %v", err)
		}
	})

	firstCreated := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	secondCreated := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)

	_, err := collection.InsertMany(ctx, []any{
		bson.M{
			"_id":                    secondEventUUID,
			"event_uuid":             secondEventUUID,
			"user_pk":                userPK,
			"user_email":             "later@example.com",
			"authorized_application": "OpenCloud",
			"created":                secondCreated,
			"received_at":            time.Now().UTC(),
		},
		bson.M{
			"_id":                    firstEventUUID,
			"event_uuid":             firstEventUUID,
			"user_pk":                userPK,
			"user_email":             "first@example.com",
			"authorized_application": "OpenCloud",
			"created":                firstCreated,
			"received_at":            time.Now().UTC(),
		},
	})
	if err != nil {
		t.Fatalf("insert test Authentik events: %v", err)
	}

	event, err := connection.FindFirstOpenCloudAuthorizationByUser(ctx, userPK)
	if err != nil {
		t.Fatalf("find first Authentik authorization event: %v", err)
	}
	if event == nil {
		t.Fatal("expected first Authentik authorization event for user")
	}
	if event["event_uuid"] != firstEventUUID {
		t.Fatalf("expected earliest event %q, got %v", firstEventUUID, event["event_uuid"])
	}
	if got := event["user_email"]; got != "first@example.com" {
		t.Fatalf("expected first event user email %q, got %#v", "first@example.com", got)
	}
}

func TestStoreAuthentikEventDeduplicatesByUser(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	collection := connection.Client.Database(databaseAuthentik).Collection(collectionAuthentikEvents)
	userPK := int64(77)
	firstUUID := "first-opencloud-duplicate-" + time.Now().UTC().Format("20060102150405.000000000")
	secondUUID := "second-opencloud-duplicate-" + time.Now().UTC().Format("20060102150405.000000001")

	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := collection.DeleteMany(cleanupContext, bson.M{"user_pk": userPK}); err != nil {
			t.Errorf("delete test Authentik events: %v", err)
		}
	})

	firstPayload := bson.M{
		"event_uuid":             firstUUID,
		"user_pk":                userPK,
		"user_email":             "dup@example.com",
		"authorized_application": "OpenCloud",
		"created":                time.Now().UTC().Add(-30 * time.Minute),
	}
	secondPayload := bson.M{
		"event_uuid":             secondUUID,
		"user_pk":                userPK,
		"user_email":             "dup@example.com",
		"authorized_application": "OpenCloud",
		"created":                time.Now().UTC(),
	}

	inserted, err := connection.StoreAuthentikEvent(ctx, firstPayload)
	if err != nil {
		t.Fatalf("store first Authentik event: %v", err)
	}
	if !inserted {
		t.Fatal("first duplicate-check event was not stored")
	}

	inserted, err = connection.StoreAuthentikEvent(ctx, secondPayload)
	if err != nil {
		t.Fatalf("store second Authentik event: %v", err)
	}
	if inserted {
		t.Fatal("second event for same user should be ignored")
	}

	count, err := collection.CountDocuments(ctx, bson.M{"user_pk": userPK})
	if err != nil {
		t.Fatalf("count Authentik events for user: %v", err)
	}
	if count != 1 {
		t.Fatalf("stored %d Authentik events for user with same OpenCloud authorization, want 1", count)
	}
}
