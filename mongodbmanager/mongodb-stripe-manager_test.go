package mongodbmanager

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestGetUserReturnsEmptyUserForMissingMapping(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	uid := fmt.Sprintf("missing-customer-test-%d", time.Now().UnixNano())
	user, err := connection.GetUser(ctx, User{UID: uid})
	if err != nil {
		t.Fatalf("GetUser() error = %v, want nil", err)
	}
	if user != (User{}) {
		t.Fatalf("GetUser() = %+v, want empty user", user)
	}
}

func TestCreateUserRejectsDuplicateUID(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	if err := connection.EnsureIndexes(ctx); err != nil {
		t.Fatalf("ensure user unique index: %v", err)
	}
	uid := fmt.Sprintf("duplicate-customer-test-%d", time.Now().UnixNano())
	first := User{UID: uid, CustomerID: "cus_first"}
	if err := connection.CreateUser(ctx, first); err != nil {
		t.Fatalf("create first user mapping: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := connection.DeleteUser(cleanupContext, first.CustomerID); err != nil {
			t.Errorf("delete first user mapping: %v", err)
		}
	})

	if err := connection.CreateUser(ctx, User{UID: uid, CustomerID: "cus_second"}); !mongo.IsDuplicateKeyError(err) {
		t.Fatalf("duplicate CreateUser() error = %v, want duplicate-key error", err)
	}
	stored, err := connection.GetUser(ctx, User{UID: uid})
	if err != nil {
		t.Fatalf("read stored user mapping: %v", err)
	}
	if stored.CustomerID != first.CustomerID {
		t.Fatalf("customer mapping changed to %q, want original %q", stored.CustomerID, first.CustomerID)
	}
}

func TestCreateUserConcurrentDuplicateUIDHasOneWinner(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	if err := connection.EnsureIndexes(ctx); err != nil {
		t.Fatalf("ensure user unique index: %v", err)
	}
	uid := fmt.Sprintf("concurrent-customer-test-%d", time.Now().UnixNano())
	users := connection.Client.Database(databaseStripe).Collection(collectionUsers)
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := users.DeleteMany(cleanupContext, bson.M{"uid": uid}); err != nil {
			t.Errorf("delete concurrent user mappings: %v", err)
		}
	})
	const attempts = 12
	var waitGroup sync.WaitGroup
	results := make(chan error, attempts)
	for index := 0; index < attempts; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			results <- connection.CreateUser(ctx, User{
				UID:        uid,
				CustomerID: fmt.Sprintf("cus_concurrent_%d", index),
			})
		}(index)
	}
	waitGroup.Wait()
	close(results)

	created := 0
	duplicates := 0
	for err := range results {
		switch {
		case err == nil:
			created++
		case mongo.IsDuplicateKeyError(err):
			duplicates++
		default:
			t.Errorf("concurrent CreateUser() error = %v, want success or duplicate-key error", err)
		}
	}
	if created != 1 || duplicates != attempts-1 {
		t.Fatalf("created %d mappings and rejected %d duplicates; want 1 and %d", created, duplicates, attempts-1)
	}

	stored, err := connection.GetUser(ctx, User{UID: uid})
	if err != nil {
		t.Fatalf("read winning user mapping: %v", err)
	}
	if stored.CustomerID == "" {
		t.Fatal("winning mapping has an empty customer ID")
	}
}

func TestConcurrentUserProvisioningUsesOneMongoLease(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	uid := fmt.Sprintf("provisioning-lease-test-%d", time.Now().UnixNano())
	users := connection.Client.Database(databaseStripe).Collection(collectionUsers)
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := users.DeleteMany(cleanupContext, bson.M{"uid": uid}); err != nil {
			t.Errorf("delete provisioning test mapping: %v", err)
		}
	})

	const attempts = 12
	var waitGroup sync.WaitGroup
	results := make(chan string, attempts)
	errorsFound := make(chan error, attempts)
	for index := 0; index < attempts; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			if err := connection.EnsureUser(ctx, uid); err != nil {
				errorsFound <- err
				return
			}
			now := time.Now().UTC()
			token := fmt.Sprintf("provisioner-%d", index)
			claimed, err := connection.ClaimUserProvisioning(ctx, uid, token, now, now.Add(time.Minute))
			if err != nil {
				errorsFound <- err
				return
			}
			if claimed {
				results <- token
			}
		}(index)
	}
	waitGroup.Wait()
	close(results)
	close(errorsFound)
	for err := range errorsFound {
		t.Errorf("concurrent provisioning operation failed: %v", err)
	}

	winningToken := ""
	for token := range results {
		if winningToken != "" {
			t.Errorf("multiple callers acquired the provisioning lease: %q and %q", winningToken, token)
		}
		winningToken = token
	}
	if winningToken == "" {
		t.Fatal("no caller acquired the provisioning lease")
	}
	count, err := users.CountDocuments(ctx, bson.M{"uid": uid})
	if err != nil {
		t.Fatalf("count user mappings: %v", err)
	}
	if count != 1 {
		t.Fatalf("MongoDB contains %d mappings for one UID, want 1", count)
	}

	stored, err := connection.GetUser(ctx, User{UID: uid})
	if err != nil {
		t.Fatalf("read provisioning placeholder: %v", err)
	}
	if stored.CustomerID != "" || stored.ProvisioningToken != winningToken {
		t.Fatalf("unexpected placeholder state: customer=%q lease token=%q, want empty customer and winning token %q", stored.CustomerID, stored.ProvisioningToken, winningToken)
	}
	if stored.ProvisioningLeaseExpiresAt == nil {
		t.Fatal("provisioning lease expiration was not stored")
	}

	reclaimTime := stored.ProvisioningLeaseExpiresAt.Add(time.Second)
	const replacementToken = "replacement-provisioner"
	claimed, err := connection.ClaimUserProvisioning(ctx, uid, replacementToken, reclaimTime, reclaimTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("claim expired provisioning lease: %v", err)
	}
	if !claimed {
		t.Fatal("failed to claim expired provisioning lease")
	}
	if err := connection.SetUserCustomerID(ctx, uid, winningToken, "cus_stale"); err == nil {
		t.Fatal("stale lease owner was able to save the customer ID")
	}
	if err := connection.SetUserCustomerID(ctx, uid, replacementToken, "cus_winner"); err != nil {
		t.Fatalf("save customer ID under replacement lease: %v", err)
	}
	if claimed, err := connection.ClaimUserProvisioning(ctx, uid, "later-provisioner", reclaimTime.Add(time.Minute), reclaimTime.Add(2*time.Minute)); err != nil {
		t.Fatalf("check completed mapping: %v", err)
	} else if claimed {
		t.Fatal("provisioning lease was granted after customer ID was saved")
	}
}
