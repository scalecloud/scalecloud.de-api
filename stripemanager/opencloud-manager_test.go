package stripemanager

import (
	"math"
	"testing"
)

func TestStorageQuotaBytes(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		unit   string
		want   int64
	}{
		{name: "decimal gigabytes", amount: 100, unit: "GB", want: 100_000_000_000},
		{name: "binary gibibytes", amount: 2, unit: "GiB", want: 2 << 30},
		{name: "bytes", amount: 10, unit: "B", want: 10},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := storageQuotaBytes(test.amount, test.unit)
			if err != nil || got != test.want {
				t.Fatalf("storageQuotaBytes() = %d, %v; want %d", got, err, test.want)
			}
		})
	}
	if _, err := storageQuotaBytes(math.MaxInt64, "GB"); err == nil {
		t.Fatal("expected overflow to be rejected")
	}
	if _, err := storageQuotaBytes(1, "unknown"); err == nil {
		t.Fatal("expected unknown unit to be rejected")
	}
}
