package stripemanager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v87"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func TestGetOrCreateStripeCustomerDoesNotProvisionWhenMongoLookupFails(t *testing.T) {
	lookupErr := errors.New("MongoDB unavailable")
	store := newFakeCustomerMappingStore(0)
	store.lookupErr = lookupErr
	stripeCreator := newFakeStripeCustomerCreator()

	_, err := getOrCreateStripeCustomer(t.Context(), store, stripeCreator, "person@example.com", "authentik-user-id")
	if !errors.Is(err, lookupErr) {
		t.Fatalf("getOrCreateStripeCustomer() error = %v, want %v", err, lookupErr)
	}
	if stripeCreator.calls != 0 || store.ensureCalls != 0 || store.mappingCount() != 0 {
		t.Fatalf("Mongo lookup failure triggered provisioning: Stripe calls=%d Mongo ensures=%d mappings=%d", stripeCreator.calls, store.ensureCalls, store.mappingCount())
	}
}

func TestGetOrCreateStripeCustomerDoesNotProvisionWhenPlaceholderCreationFails(t *testing.T) {
	store := newFakeCustomerMappingStore(0)
	store.ensureErr = errors.New("MongoDB insert failed")
	stripeCreator := newFakeStripeCustomerCreator()

	_, err := getOrCreateStripeCustomer(t.Context(), store, stripeCreator, "person@example.com", "authentik-user-id")
	if !errors.Is(err, store.ensureErr) {
		t.Fatalf("getOrCreateStripeCustomer() error = %v, want %v", err, store.ensureErr)
	}
	if stripeCreator.calls != 0 || store.mappingCount() != 0 {
		t.Fatalf("placeholder failure triggered provisioning: Stripe calls=%d mappings=%d", stripeCreator.calls, store.mappingCount())
	}
}

func TestGetOrCreateStripeCustomerDoesNotProvisionWhenLeaseClaimFails(t *testing.T) {
	store := newFakeCustomerMappingStore(0)
	store.claimErr = errors.New("MongoDB lease update failed")
	stripeCreator := newFakeStripeCustomerCreator()

	_, err := getOrCreateStripeCustomer(t.Context(), store, stripeCreator, "person@example.com", "authentik-user-id")
	if !errors.Is(err, store.claimErr) {
		t.Fatalf("getOrCreateStripeCustomer() error = %v, want %v", err, store.claimErr)
	}
	if stripeCreator.calls != 0 || store.mappingCount() != 1 {
		t.Fatalf("lease claim failure triggered Stripe: Stripe calls=%d mappings=%d", stripeCreator.calls, store.mappingCount())
	}
}

func TestGetOrCreateStripeCustomerReusesExistingMapping(t *testing.T) {
	store := newFakeCustomerMappingStore(0)
	store.users["authentik-user-id"] = mongomanager.User{UID: "authentik-user-id", CustomerID: "cus_existing"}
	stripeCreator := newFakeStripeCustomerCreator()

	customerID, err := getOrCreateStripeCustomer(t.Context(), store, stripeCreator, "person@example.com", "authentik-user-id")
	if err != nil {
		t.Fatalf("getOrCreateStripeCustomer() error = %v", err)
	}
	if customerID != "cus_existing" || stripeCreator.calls != 0 || store.ensureCalls != 0 {
		t.Fatalf("existing mapping was not reused: customer=%q Stripe calls=%d Mongo ensures=%d", customerID, stripeCreator.calls, store.ensureCalls)
	}
}

func TestGetOrCreateStripeCustomerDoesNotWriteWhenStripeFails(t *testing.T) {
	stripeErr := errors.New("Stripe unavailable")
	store := newFakeCustomerMappingStore(0)
	stripeCreator := newFakeStripeCustomerCreator()
	stripeCreator.err = stripeErr

	_, err := getOrCreateStripeCustomer(t.Context(), store, stripeCreator, "person@example.com", "authentik-user-id")
	if !errors.Is(err, stripeErr) {
		t.Fatalf("getOrCreateStripeCustomer() error = %v, want %v", err, stripeErr)
	}
	if store.mappingCount() != 1 || store.customerID("authentik-user-id") != "" || store.hasLease("authentik-user-id") {
		t.Fatalf("after Stripe failure, want one unlocked placeholder without customer ID; mappings=%d customer=%q lease=%t", store.mappingCount(), store.customerID("authentik-user-id"), store.hasLease("authentik-user-id"))
	}
}

func TestGetOrCreateStripeCustomerDoesNotReturnCustomerWhenMongoSaveFails(t *testing.T) {
	store := newFakeCustomerMappingStore(0)
	store.setErr = errors.New("MongoDB update failed")
	stripeCreator := newFakeStripeCustomerCreator()

	customerID, err := getOrCreateStripeCustomer(t.Context(), store, stripeCreator, "person@example.com", "authentik-user-id")
	if !errors.Is(err, store.setErr) {
		t.Fatalf("getOrCreateStripeCustomer() error = %v, want %v", err, store.setErr)
	}
	if customerID != "" || stripeCreator.created != 1 || store.customerID("authentik-user-id") != "" || store.hasLease("authentik-user-id") {
		t.Fatalf("Mongo save failure returned or stored an unmapped customer: returned=%q Stripe created=%d stored=%q lease=%t", customerID, stripeCreator.created, store.customerID("authentik-user-id"), store.hasLease("authentik-user-id"))
	}
}

func TestGetOrCreateStripeCustomerConcurrentRequestsConvergeOnOneMapping(t *testing.T) {
	const attempts = 12
	store := newFakeCustomerMappingStore(attempts)
	stripeCreator := newFakeStripeCustomerCreator()
	results := make(chan string, attempts)
	errorsFound := make(chan error, attempts)
	var waitGroup sync.WaitGroup
	for range attempts {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			customerID, err := getOrCreateStripeCustomer(t.Context(), store, stripeCreator, "person@example.com", "authentik-user-id")
			if err != nil {
				errorsFound <- err
				return
			}
			results <- customerID
		}()
	}
	waitGroup.Wait()
	close(results)
	close(errorsFound)
	for err := range errorsFound {
		t.Errorf("concurrent customer provisioning failed: %v", err)
	}

	winnerID := ""
	for customerID := range results {
		if winnerID == "" {
			winnerID = customerID
		} else if customerID != winnerID {
			t.Errorf("concurrent callers received different customer IDs: %q and %q", winnerID, customerID)
		}
	}
	if winnerID == "" {
		t.Fatal("no concurrent caller received a customer ID")
	}
	if stripeCreator.created != 1 {
		t.Errorf("Stripe created %d customers for one Authentik subject, want 1", stripeCreator.created)
	}
	if store.ensureCalls == 0 || store.ensureCalls > attempts || store.mappingCount() != 1 {
		t.Errorf("Mongo ensures=%d mappings=%d; want 1-%d ensures converging to one mapping", store.ensureCalls, store.mappingCount(), attempts)
	}
}

func TestGetOrCreateStripeCustomerConcurrentRequestsWithMongo(t *testing.T) {
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("set MONGODB_TEST_URI to run the MongoDB integration test")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect to test MongoDB: %v", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("ping test MongoDB: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if err := client.Disconnect(cleanupContext); err != nil {
			t.Errorf("disconnect test MongoDB client: %v", err)
		}
	})
	connection := &mongomanager.MongoConnection{Client: client, Log: zap.NewNop()}
	if err := connection.EnsureIndexes(ctx); err != nil {
		t.Fatalf("ensure MongoDB indexes: %v", err)
	}

	uid := fmt.Sprintf("concurrent-customer-flow-test-%d", time.Now().UnixNano())
	users := client.Database("stripe").Collection("users")
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := users.DeleteMany(cleanupContext, map[string]string{"uid": uid}); err != nil {
			t.Errorf("delete test customer mapping: %v", err)
		}
	})

	const attempts = 12
	type result struct {
		customerID string
		err        error
	}
	results := make(chan result, attempts)
	stripeCreator := newFakeStripeCustomerCreator()
	var waitGroup sync.WaitGroup
	for index := 0; index < attempts; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			customerID, err := getOrCreateStripeCustomer(ctx, connection, stripeCreator, "person@example.com", uid)
			results <- result{customerID: customerID, err: err}
		}()
	}
	waitGroup.Wait()
	close(results)

	winnerID := ""
	for result := range results {
		if result.err != nil {
			t.Errorf("concurrent customer provisioning failed: %v", result.err)
			continue
		}
		if winnerID == "" {
			winnerID = result.customerID
		} else if winnerID != result.customerID {
			t.Errorf("concurrent callers received different customer IDs: %q and %q", winnerID, result.customerID)
		}
	}
	if winnerID == "" {
		t.Fatal("no concurrent caller received a customer ID")
	}
	if stripeCreator.created != 1 {
		t.Errorf("Stripe created %d customers for one Authentik subject, want 1", stripeCreator.created)
	}
	count, err := users.CountDocuments(ctx, map[string]string{"uid": uid})
	if err != nil {
		t.Fatalf("count MongoDB customer mappings: %v", err)
	}
	if count != 1 {
		t.Fatalf("MongoDB contains %d mappings for one UID, want 1", count)
	}
	stored, err := connection.GetUser(ctx, mongomanager.User{UID: uid})
	if err != nil {
		t.Fatalf("read stored customer mapping: %v", err)
	}
	if stored.CustomerID != winnerID || stored.ProvisioningToken != "" {
		t.Fatalf("unexpected final mapping state: customer=%q lease token=%q", stored.CustomerID, stored.ProvisioningToken)
	}
}

func TestStripeCustomerIdempotencyKeyIsStablePerSubject(t *testing.T) {
	first := stripeCustomerIdempotencyKey("authentik-user-id")
	second := stripeCustomerIdempotencyKey("authentik-user-id")
	other := stripeCustomerIdempotencyKey("another-user-id")
	if first == "" || first != second || first == other {
		t.Fatalf("unexpected idempotency keys: first=%q second=%q other=%q", first, second, other)
	}
}

type fakeCustomerMappingStore struct {
	mu               sync.Mutex
	users            map[string]mongomanager.User
	lookupErr        error
	ensureErr        error
	claimErr         error
	setErr           error
	initialLookups   int
	lookupCalls      int
	ensureCalls      int
	initialLookupsOK chan struct{}
}

func newFakeCustomerMappingStore(initialLookups int) *fakeCustomerMappingStore {
	return &fakeCustomerMappingStore{
		users:            make(map[string]mongomanager.User),
		initialLookups:   initialLookups,
		initialLookupsOK: make(chan struct{}),
	}
}

func (store *fakeCustomerMappingStore) GetUser(_ context.Context, filter mongomanager.User) (mongomanager.User, error) {
	store.mu.Lock()
	store.lookupCalls++
	lookupNumber := store.lookupCalls
	if store.initialLookups > 0 && lookupNumber <= store.initialLookups {
		if lookupNumber == store.initialLookups {
			close(store.initialLookupsOK)
		}
		barrier := store.initialLookupsOK
		store.mu.Unlock()
		<-barrier
		store.mu.Lock()
	}
	defer store.mu.Unlock()
	if store.lookupErr != nil {
		return mongomanager.User{}, store.lookupErr
	}
	user, exists := store.users[filter.UID]
	if !exists {
		return mongomanager.User{}, mongo.ErrNoDocuments
	}
	return user, nil
}

func (store *fakeCustomerMappingStore) EnsureUser(_ context.Context, uid string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureCalls++
	if store.ensureErr != nil {
		return store.ensureErr
	}
	if _, exists := store.users[uid]; !exists {
		store.users[uid] = mongomanager.User{UID: uid}
	}
	return nil
}

func (store *fakeCustomerMappingStore) ClaimUserProvisioning(_ context.Context, uid, token string, now, expiresAt time.Time) (bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.claimErr != nil {
		return false, store.claimErr
	}
	user, exists := store.users[uid]
	if !exists || user.CustomerID != "" || user.ProvisioningToken != "" && user.ProvisioningLeaseExpiresAt != nil && user.ProvisioningLeaseExpiresAt.After(now) {
		return false, nil
	}
	user.ProvisioningToken = token
	user.ProvisioningLeaseExpiresAt = &expiresAt
	store.users[uid] = user
	return true, nil
}

func (store *fakeCustomerMappingStore) SetUserCustomerID(_ context.Context, uid, token, customerID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.setErr != nil {
		return store.setErr
	}
	user, exists := store.users[uid]
	if !exists || user.ProvisioningToken != token || user.CustomerID != "" {
		return errors.New("customer provisioning lease is no longer owned")
	}
	user.CustomerID = customerID
	user.ProvisioningToken = ""
	user.ProvisioningLeaseExpiresAt = nil
	store.users[uid] = user
	return nil
}

func (store *fakeCustomerMappingStore) ReleaseUserProvisioning(_ context.Context, uid, token string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	user, exists := store.users[uid]
	if exists && user.ProvisioningToken == token {
		user.ProvisioningToken = ""
		user.ProvisioningLeaseExpiresAt = nil
		store.users[uid] = user
	}
	return nil
}

func (store *fakeCustomerMappingStore) customerID(uid string) string {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.users[uid].CustomerID
}

func (store *fakeCustomerMappingStore) hasLease(uid string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.users[uid].ProvisioningToken != ""
}

func (store *fakeCustomerMappingStore) mappingCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.users)
}

type fakeStripeCustomerCreator struct {
	mu       sync.Mutex
	idsByUID map[string]string
	err      error
	calls    int
	created  int
}

func newFakeStripeCustomerCreator() *fakeStripeCustomerCreator {
	return &fakeStripeCustomerCreator{idsByUID: make(map[string]string)}
}

func (creator *fakeStripeCustomerCreator) CreateCustomer(_ context.Context, _ string, uid string) (*stripe.Customer, error) {
	creator.mu.Lock()
	defer creator.mu.Unlock()
	creator.calls++
	if creator.err != nil {
		return nil, creator.err
	}
	if customerID, exists := creator.idsByUID[uid]; exists {
		return &stripe.Customer{ID: customerID}, nil
	}
	creator.created++
	customerID := fmt.Sprintf("cus_test_%d", creator.created)
	creator.idsByUID[uid] = customerID
	return &stripe.Customer{ID: customerID}, nil
}
