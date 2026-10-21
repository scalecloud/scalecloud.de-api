package mongodbmanager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func TestCreatePendingSeatWithinCapacityRejectsNonPositiveCapacity(t *testing.T) {
	connection := &MongoDBConnection{}
	for _, maxSeats := range []int64{0, -1} {
		if err := connection.CreatePendingSeatWithinCapacity(context.Background(), Seat{}, maxSeats); !errors.Is(err, ErrSeatCapacityReached) {
			t.Fatalf("capacity %d: error = %v, want %v", maxSeats, err, ErrSeatCapacityReached)
		}
	}
}

func TestCreatePendingSeatWithinCapacitySerializesConcurrentReservations(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	client := connection.Client
	subscriptionID := fmt.Sprintf("capacity-test-%d", time.Now().UnixNano())
	seats := client.Database(databaseSubscription).Collection(collectionSeats)
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := seats.DeleteMany(cleanupContext, bson.M{"subscriptionID": subscriptionID}); err != nil {
			t.Errorf("remove test seats: %v", err)
		}
		if _, err := client.Database(databaseSubscription).Collection(collectionSeatLocks).DeleteOne(cleanupContext, bson.M{"_id": subscriptionID}); err != nil {
			t.Errorf("remove test reservation lock: %v", err)
		}
	})

	const maxSeats = int64(3)
	const attempts = 9
	var waitGroup sync.WaitGroup
	results := make(chan error, attempts)
	for index := 0; index < attempts; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			expiresAt := time.Now().UTC().Add(time.Hour)
			seat := Seat{
				SubscriptionID:      subscriptionID,
				EMail:               fmt.Sprintf("user-%d@example.com", index),
				Roles:               []Role{RoleUser},
				InvitationTokenHash: fmt.Sprintf("test-token-%d", index),
				InvitationExpiresAt: &expiresAt,
			}
			reservationContext, reservationCancel := context.WithTimeout(ctx, 10*time.Second)
			defer reservationCancel()
			results <- connection.CreatePendingSeatWithinCapacity(reservationContext, seat, maxSeats)
		}(index)
	}
	waitGroup.Wait()
	close(results)

	created := int64(0)
	capacityRejected := 0
	for err := range results {
		switch {
		case err == nil:
			created++
		case errors.Is(err, ErrSeatCapacityReached):
			capacityRejected++
		default:
			t.Errorf("unexpected reservation error: %v", err)
		}
	}
	if created != maxSeats || capacityRejected != attempts-int(maxSeats) {
		t.Fatalf("created %d seats and rejected %d reservations; want %d and %d", created, capacityRejected, maxSeats, attempts-int(maxSeats))
	}
	count, err := connection.CountSeats(ctx, subscriptionID)
	if err != nil {
		t.Fatalf("count reserved seats: %v", err)
	}
	if count != maxSeats {
		t.Fatalf("stored %d seats, want capacity limit %d", count, maxSeats)
	}
}

func newTestMongoDBConnection(t *testing.T) (*MongoDBConnection, context.Context) {
	t.Helper()
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("set MONGODB_TEST_URI to run the MongoDB integration test")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect to test MongoDB: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := client.Disconnect(cleanupContext); err != nil {
			t.Errorf("disconnect test MongoDB client: %v", err)
		}
	})
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("ping test MongoDB: %v", err)
	}
	return &MongoDBConnection{Client: client, Log: zap.NewNop()}, ctx
}
