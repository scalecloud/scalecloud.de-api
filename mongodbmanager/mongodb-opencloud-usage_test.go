package mongodbmanager

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestOpenCloudDailyUsageUpsertsOneRecordPerDriveAndDay(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	if err := connection.ensureOpenCloudUsageCollections(ctx); err != nil {
		t.Fatalf("ensure usage collection: %v", err)
	}
	driveID := "test-" + bson.NewObjectID().Hex()
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = connection.Client.Database(databaseOpenCloud).Collection(collectionOpenCloudHistory).DeleteMany(cleanupContext, bson.M{"driveId": driveID})
	})

	now := time.Now().UTC().Truncate(24 * time.Hour)
	used, quota := int64(7), int64(100)
	first := OpenCloudUsageRecord{DriveID: driveID, SubscriptionID: "sub-1", DriveType: "project", SampledAt: now, UsedBytes: &used, QuotaBytes: &quota}
	if err := connection.StoreOpenCloudDailyUsage(ctx, first); err != nil {
		t.Fatalf("store first daily sample: %v", err)
	}

	used = 12
	second := first
	second.SampledAt = now.Add(12 * time.Hour)
	second.UsedBytes = &used
	if err := connection.StoreOpenCloudDailyUsage(ctx, second); err != nil {
		t.Fatalf("update daily sample: %v", err)
	}
	history, err := connection.GetOpenCloudUsageHistory(ctx, driveID, now, now.Add(24*time.Hour))
	if err != nil || len(history) != 1 || history[0].UsedBytes == nil || *history[0].UsedBytes != 12 {
		t.Fatalf("daily history = (%+v, %v), want one updated sample", history, err)
	}
}

func TestOpenCloudStorageUsedBytesIncludesSeatsAndManagedSpaces(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	if err := connection.ensureOpenCloudUsageCollections(ctx); err != nil {
		t.Fatalf("ensure OpenCloud registry collection: %v", err)
	}
	subscriptionID := "allocation-test-" + bson.NewObjectID().Hex()
	seatCollection := connection.Client.Database(databaseSubscription).Collection(collectionSeats)
	registryCollection := connection.Client.Database(databaseOpenCloud).Collection(collectionOpenCloudRegistry)
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = seatCollection.DeleteMany(cleanupContext, bson.M{"subscriptionID": subscriptionID})
		_, _ = registryCollection.DeleteMany(cleanupContext, bson.M{"subscriptionID": subscriptionID})
	})

	seats := []any{
		Seat{SubscriptionID: subscriptionID, EMail: "owner@example.com", Roles: []Role{RoleOwner}, OpenCloudStorageBytes: 2_000_000_000_000},
		Seat{SubscriptionID: subscriptionID, EMail: "person@example.com", Roles: []Role{RoleUser}, Status: SeatStatusPending, OpenCloudStorageBytes: 1_000_000_000_000},
	}
	for _, seat := range seats {
		if _, err := seatCollection.InsertOne(ctx, seat); err != nil {
			t.Fatalf("insert test seat: %v", err)
		}
	}
	if err := connection.UpsertOpenCloudResource(ctx, OpenCloudResourceRegistry{
		SubscriptionID:    subscriptionID,
		ResourceType:      "space",
		ResourceID:        "drive-1",
		Scope:             "allocation",
		StorageQuotaBytes: 3_000_000_000_000,
	}); err != nil {
		t.Fatalf("insert test space allocation: %v", err)
	}
	if err := connection.UpsertOpenCloudResource(ctx, OpenCloudResourceRegistry{
		SubscriptionID:    subscriptionID,
		ResourceType:      "space",
		ResourceID:        "legacy-drive",
		Scope:             "project",
		StorageQuotaBytes: 9_000_000_000_000,
	}); err != nil {
		t.Fatalf("insert legacy space record: %v", err)
	}

	usedBytes, err := connection.GetOpenCloudStorageUsedBytes(ctx, subscriptionID)
	if err != nil || usedBytes != 6_000_000_000_000 {
		t.Fatalf("GetOpenCloudStorageUsedBytes() = %d, %v; want 6000000000000", usedBytes, err)
	}
}

func TestGetOpenCloudProjectSpacesIncludesManagedSpaces(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	if err := connection.ensureOpenCloudUsageCollections(ctx); err != nil {
		t.Fatalf("ensure OpenCloud registry collection: %v", err)
	}
	subscriptionID := "space-discovery-test-" + bson.NewObjectID().Hex()
	seatCollection := connection.Client.Database(databaseSubscription).Collection(collectionSeats)
	registryCollection := connection.Client.Database(databaseOpenCloud).Collection(collectionOpenCloudRegistry)
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = seatCollection.DeleteMany(cleanupContext, bson.M{"subscriptionID": subscriptionID})
		_, _ = registryCollection.DeleteMany(cleanupContext, bson.M{"subscriptionID": subscriptionID})
	})

	if _, err := seatCollection.InsertOne(ctx, Seat{
		SubscriptionID:   subscriptionID,
		EMail:            "owner@example.com",
		Roles:            []Role{RoleOwner},
		OpenCloudDriveID: "legacy-drive",
	}); err != nil {
		t.Fatalf("insert owner seat: %v", err)
	}
	if err := connection.UpsertOpenCloudResource(ctx, OpenCloudResourceRegistry{
		SubscriptionID: subscriptionID,
		ResourceType:   "space",
		ResourceID:     "managed-drive",
		Scope:          "allocation",
		DriveType:      "project",
	}); err != nil {
		t.Fatalf("insert allocated space: %v", err)
	}

	spaces, err := connection.GetOpenCloudProjectSpaces(ctx)
	if err != nil {
		t.Fatalf("GetOpenCloudProjectSpaces(): %v", err)
	}
	found := map[string]bool{}
	for _, space := range spaces {
		if space.SubscriptionID == subscriptionID {
			found[space.DriveID] = true
		}
	}
	if !found["legacy-drive"] || !found["managed-drive"] {
		t.Fatalf("project spaces = %+v, want both owner and managed drives", spaces)
	}
}
