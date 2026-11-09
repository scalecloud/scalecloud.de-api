package stripemanager

import (
	"errors"
	"math"
	"testing"

	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
)

func TestOpenCloudQuantityBytes(t *testing.T) {
	got, err := openCloudQuantityBytes(3, 1_000_000_000_000)
	if err != nil || got != 3_000_000_000_000 {
		t.Fatalf("openCloudQuantityBytes() = %d, %v", got, err)
	}
	for _, test := range []struct {
		quantity  int64
		unitBytes int64
	}{
		{quantity: 0, unitBytes: 1},
		{quantity: 1, unitBytes: 0},
		{quantity: math.MaxInt64, unitBytes: 2},
	} {
		if _, err := openCloudQuantityBytes(test.quantity, test.unitBytes); err == nil {
			t.Errorf("openCloudQuantityBytes(%d, %d) should fail", test.quantity, test.unitBytes)
		}
	}
}

func TestValidateOpenCloudReplacement(t *testing.T) {
	if err := validateOpenCloudReplacement(100, 70, 30, 60); err != nil {
		t.Fatalf("valid replacement failed: %v", err)
	}
	for _, test := range []struct {
		capacity int64
		used     int64
		old      int64
		updated  int64
	}{
		{capacity: 100, used: 70, old: 30, updated: 61},
		{capacity: 100, used: 20, old: 30, updated: 10},
		{capacity: 100, used: 101, old: 1, updated: 1},
		{capacity: 0, used: 0, old: 0, updated: 1},
	} {
		if err := validateOpenCloudReplacement(test.capacity, test.used, test.old, test.updated); !errors.Is(err, mongodbmanager.ErrOpenCloudStorageCapacityReached) {
			t.Errorf("validateOpenCloudReplacement(%+v) = %v, want capacity error", test, err)
		}
	}
}

func TestValidateOpenCloudUserRoles(t *testing.T) {
	if err := validateOpenCloudUserRoles([]mongodbmanager.Role{mongodbmanager.RoleUser, mongodbmanager.RoleBilling}); err != nil {
		t.Fatalf("valid roles rejected: %v", err)
	}
	for _, roles := range [][]mongodbmanager.Role{
		{},
		{mongodbmanager.RoleOwner},
		{mongodbmanager.Role("unknown")},
	} {
		if err := validateOpenCloudUserRoles(roles); err == nil {
			t.Errorf("validateOpenCloudUserRoles(%v) should fail", roles)
		}
	}
}
