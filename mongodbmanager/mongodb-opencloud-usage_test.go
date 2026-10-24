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
