package stripemanager

import (
	"errors"
	"math"
	"testing"

	"github.com/scalecloud/scalecloud.de-api/mongodbmanager"
	"github.com/scalecloud/scalecloud.de-api/opencloudmanager"
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
		{quantity: -1, unitBytes: 1},
		{quantity: 1, unitBytes: 0},
		{quantity: math.MaxInt64, unitBytes: 2},
	} {
		if _, err := openCloudQuantityBytes(test.quantity, test.unitBytes); err == nil {
			t.Errorf("openCloudQuantityBytes(%d, %d) should fail", test.quantity, test.unitBytes)
		}
	}
	if got, err := openCloudQuantityBytes(0, 1_000_000_000_000); err != nil || got != 0 {
		t.Fatalf("zero OpenCloud allocation = %d, %v", got, err)
	}
}

func TestValidateOpenCloudReplacement(t *testing.T) {
	if err := validateOpenCloudReplacement(100, 70, 30, 60); err != nil {
		t.Fatalf("valid replacement failed: %v", err)
	}
	if err := validateOpenCloudReplacement(100, 70, 30, 0); err != nil {
		t.Fatalf("zero allocation replacement failed: %v", err)
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
	if err := validateOpenCloudUserRoles([]mongodbmanager.OpenCloudRole{mongodbmanager.OpenCloudRoleAdministrator, mongodbmanager.OpenCloudRoleBilling}); err != nil {
		t.Fatalf("valid roles rejected: %v", err)
	}
	for _, roles := range [][]mongodbmanager.OpenCloudRole{
		{mongodbmanager.OpenCloudRoleOwner},
		{mongodbmanager.OpenCloudRole("User")},
		{mongodbmanager.OpenCloudRole("unknown")},
	} {
		if err := validateOpenCloudUserRoles(roles); err == nil {
			t.Errorf("validateOpenCloudUserRoles(%v) should fail", roles)
		}
	}
}

func TestValidateOpenCloudInvitation(t *testing.T) {
	tests := []struct {
		name     string
		quantity int64
		roles    []mongodbmanager.OpenCloudRole
		wantErr  bool
	}{
		{name: "storage-only user", quantity: 1},
		{name: "admin without storage", roles: []mongodbmanager.OpenCloudRole{mongodbmanager.OpenCloudRoleAdministrator}},
		{name: "billing without storage", roles: []mongodbmanager.OpenCloudRole{mongodbmanager.OpenCloudRoleBilling}},
		{name: "no role and no storage", wantErr: true},
		{name: "removed User role", quantity: 1, roles: []mongodbmanager.OpenCloudRole{mongodbmanager.OpenCloudRole("User")}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateOpenCloudInvitation(test.quantity, test.roles)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateOpenCloudInvitation() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestOpenCloudQuotaAfterRemovingAllocation(t *testing.T) {
	got, err := openCloudQuotaAfterRemovingAllocation(3, 2)
	if err != nil || got != 1 {
		t.Fatalf("quota after removal = %d, %v; want 1", got, err)
	}
	if _, err := openCloudQuotaAfterRemovingAllocation(1, 2); err == nil {
		t.Fatal("quota removal accepted an allocation larger than the aggregate")
	}
}

func TestFindOpenCloudUserByEmail(t *testing.T) {
	user, err := findOpenCloudUserByEmail([]opencloudmanager.User{
		{ID: "user-1", Mail: "person@example.com"},
	}, " PERSON@example.com ")
	if err != nil || user.ID != "user-1" {
		t.Fatalf("findOpenCloudUserByEmail() = %+v, %v", user, err)
	}
	if _, err := findOpenCloudUserByEmail(nil, "missing@example.com"); err == nil {
		t.Fatal("findOpenCloudUserByEmail() accepted an unknown email")
	}
}
