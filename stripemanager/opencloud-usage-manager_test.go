package stripemanager

import (
	"context"
	"testing"
	"time"

	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
)

func TestOpenCloudUsageRecordMapsToDailyPoint(t *testing.T) {
	used, quota, remaining := int64(15), int64(100), int64(85)
	record := mongodbmanager.OpenCloudUsageRecord{
		Day:            time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		SampledAt:      time.Date(2026, 10, 1, 21, 0, 0, 0, time.UTC),
		UsedBytes:      &used,
		QuotaBytes:     &quota,
		RemainingBytes: &remaining,
		State:          "normal",
	}
	point := openCloudUsagePoint(record)
	if point.Date != "2026-10-01" || point.UsedBytes == nil || *point.UsedBytes != used || point.QuotaBytes == nil || *point.QuotaBytes != quota {
		t.Fatalf("point = %+v, want daily OpenCloud usage mapping", point)
	}
}

func TestOpenCloudUsageCacheHitsUntilExpired(t *testing.T) {
	handler := &PaymentHandler{}
	loads := 0
	load := func(ctx context.Context, subscriptionID string) (OpenCloudUsageReply, error) {
		loads++
		return OpenCloudUsageReply{SubscriptionID: subscriptionID, SampledAt: time.Now().UTC()}, nil
	}

	first, err := handler.getOrLoadOpenCloudUsage(context.Background(), "sub-1", load)
	if err != nil || first.SubscriptionID != "sub-1" {
		t.Fatalf("first cache load = (%+v, %v)", first, err)
	}
	if _, err := handler.getOrLoadOpenCloudUsage(context.Background(), "sub-1", load); err != nil {
		t.Fatal(err)
	}
	if loads != 1 {
		t.Fatalf("load called %d times before expiry, want 1", loads)
	}

	handler.openCloudUsageMu.Lock()
	entry := handler.openCloudUsageCache["sub-1"]
	entry.expiresAt = time.Now().Add(-time.Second)
	handler.openCloudUsageCache["sub-1"] = entry
	handler.openCloudUsageMu.Unlock()

	if _, err := handler.getOrLoadOpenCloudUsage(context.Background(), "sub-1", load); err != nil {
		t.Fatal(err)
	}
	if loads != 2 {
		t.Fatalf("load called %d times after expiry, want 2", loads)
	}
}
