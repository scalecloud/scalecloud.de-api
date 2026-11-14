package stripemanager

import (
	"context"
	"testing"

	"github.com/scalecloud/scalecloud.de-api/authmanager"
)

func TestCreateCheckoutSubscriptionRequiresQuantityAtLeastOne(t *testing.T) {
	paymentHandler := &PaymentHandler{}
	_, err := paymentHandler.CreateCheckoutSubscription(
		context.Background(),
		authmanager.TokenDetails{},
		CheckoutCreateSubscriptionRequest{Quantity: 0},
	)
	if err == nil {
		t.Fatal("CreateCheckoutSubscription() accepted quantity 0")
	}
}

func TestOpenCloudInitialOwnerStorageBytesAllocatesAllPurchasedQuantity(t *testing.T) {
	got, err := openCloudInitialOwnerStorageBytes(2, map[string]string{
		"storageAmount": "1",
		"storageUnit":   "TB",
	})
	if err != nil {
		t.Fatalf("openCloudInitialOwnerStorageBytes() error = %v", err)
	}
	if want := int64(2_000_000_000_000); got != want {
		t.Fatalf("initial owner allocation = %d bytes, want %d", got, want)
	}

	if _, err := openCloudInitialOwnerStorageBytes(0, map[string]string{
		"storageAmount": "1",
		"storageUnit":   "TB",
	}); err == nil {
		t.Fatal("openCloudInitialOwnerStorageBytes() accepted zero quantity")
	}
}
