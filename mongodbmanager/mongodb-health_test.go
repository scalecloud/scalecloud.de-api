package mongodbmanager

import (
	"context"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestCheckHealthValidatesCollectionsAndEnsuresIndexes(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	for databaseName, collectionNames := range databases {
		database := connection.Client.Database(databaseName)
		for _, collectionName := range collectionNames {
			names, err := database.ListCollectionNames(ctx, bson.M{"name": collectionName})
			if err != nil {
				t.Fatalf("check collection %s.%s: %v", databaseName, collectionName, err)
			}
			if len(names) == 0 {
				if err := database.CreateCollection(ctx, collectionName); err != nil {
					t.Fatalf("create collection %s.%s: %v", databaseName, collectionName, err)
				}
			}
		}
	}

	if err := connection.CheckHealth(ctx); err != nil {
		t.Fatalf("CheckHealth() with required collections: %v", err)
	}
	assertMongoIndexExists(t, ctx, connection.Client.Database(databaseSubscription).Collection(collectionSeats), "UniqueSubscriptionEmail")
	assertMongoIndexExists(t, ctx, connection.Client.Database(databaseSubscription).Collection(collectionSeats), "UniqueSeatInvitationTokenHash")
	assertMongoIndexExists(t, ctx, connection.Client.Database(databaseNewsletters).Collection(collectionSubscribers), "UniqueNewsletterEmail")
	assertMongoIndexExists(t, ctx, connection.Client.Database(databaseStripe).Collection(collectionUsers), "UniqueStripeUserUID")

	productDatabase := connection.Client.Database(databaseProduct)
	if err := productDatabase.Collection(collectionTrial).Drop(ctx); err != nil {
		t.Fatalf("drop required collection for negative health check: %v", err)
	}
	if err := connection.CheckHealth(ctx); err == nil || !strings.Contains(err.Error(), collectionTrial) {
		t.Fatalf("CheckHealth() error = %v, want missing collection %q", err, collectionTrial)
	}
	if err := productDatabase.CreateCollection(ctx, collectionTrial); err != nil {
		t.Fatalf("restore required collection: %v", err)
	}
	if err := connection.CheckHealth(ctx); err != nil {
		t.Fatalf("CheckHealth() after restoring required collection: %v", err)
	}
}

func assertMongoIndexExists(t *testing.T, ctx context.Context, collection *mongo.Collection, expectedName string) {
	t.Helper()
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		t.Fatalf("list indexes for %s: %v", collection.Name(), err)
	}
	defer func() { _ = cursor.Close(ctx) }()
	var indexes []bson.M
	if err := cursor.All(ctx, &indexes); err != nil {
		t.Fatalf("read indexes for %s: %v", collection.Name(), err)
	}
	for _, index := range indexes {
		if index["name"] == expectedName {
			return
		}
	}
	t.Fatalf("index %q not found on collection %s", expectedName, collection.Name())
}
