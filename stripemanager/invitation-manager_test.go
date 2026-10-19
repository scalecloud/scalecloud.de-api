package stripemanager

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/scalecloud/scalecloud.de-api/authentikmanager"
	"github.com/scalecloud/scalecloud.de-api/authmanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
	"github.com/stripe/stripe-go/v87"
	"go.uber.org/zap"
)

func TestCreateSeatInvitationToken(t *testing.T) {
	token, tokenHash, err := createSeatInvitationToken()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 32 {
		t.Fatalf("expected 32 random bytes, got %d", len(decoded))
	}
	if len(tokenHash) != sha256.Size*2 {
		t.Fatalf("expected a SHA-256 hex digest, got length %d", len(tokenHash))
	}
	hash := sha256.Sum256([]byte(token))
	if tokenHash != hex.EncodeToString(hash[:]) {
		t.Fatal("stored token hash does not match the invitation token")
	}

	secondToken, _, err := createSeatInvitationToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == secondToken {
		t.Fatal("generated invitation tokens must be unique")
	}
}

func TestClaimSeatInvitationRejectsInvalidClaimsBeforeDatabaseAccess(t *testing.T) {
	validToken := strings.Repeat("a", 64)
	tests := []struct {
		name    string
		token   string
		details authmanager.TokenDetails
		wantErr error
	}{
		{
			name:    "unverified email",
			token:   validToken,
			details: authmanager.TokenDetails{UID: "user-1", EMail: "user@example.com"},
			wantErr: ErrInvitationEmailUnverified,
		},
		{
			name:    "missing user id",
			token:   validToken,
			details: authmanager.TokenDetails{EMail: "user@example.com", EmailVerified: true},
			wantErr: ErrInvitationUnavailable,
		},
		{
			name:    "missing email",
			token:   validToken,
			details: authmanager.TokenDetails{UID: "user-1", EmailVerified: true},
			wantErr: ErrInvitationUnavailable,
		},
		{
			name:    "malformed token",
			token:   "short",
			details: authmanager.TokenDetails{UID: "user-1", EMail: "user@example.com", EmailVerified: true},
			wantErr: ErrInvitationUnavailable,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			handler := &PaymentHandler{}
			if _, err := handler.ClaimSeatInvitation(t.Context(), testCase.token, testCase.details); !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ClaimSeatInvitation() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestPreviewSeatInvitationRejectsMalformedTokensBeforeDatabaseAccess(t *testing.T) {
	for _, token := range []string{"", strings.Repeat("a", 39), strings.Repeat("a", 129)} {
		t.Run("length_"+string(rune(len(token))), func(t *testing.T) {
			handler := &PaymentHandler{}
			if _, err := handler.PreviewSeatInvitation(t.Context(), token); !errors.Is(err, ErrInvitationUnavailable) {
				t.Fatalf("PreviewSeatInvitation() error = %v, want %v", err, ErrInvitationUnavailable)
			}
		})
	}
}

func TestRemovePendingSeatFromSubscriptionGroupDoesNothing(t *testing.T) {
	handler := &PaymentHandler{}
	seat := mongomanager.Seat{Status: mongomanager.SeatStatusPending}
	if err := handler.RemoveSeatFromSubscriptionGroup(t.Context(), seat); err != nil {
		t.Fatalf("RemoveSeatFromSubscriptionGroup() error = %v, want nil", err)
	}
}

func TestGetAuthentikUserPKUsesPersistedID(t *testing.T) {
	handler := &PaymentHandler{}
	seat := mongomanager.Seat{AuthentikUserPK: 123}

	userPK, err := handler.getAuthentikUserPK(t.Context(), seat)
	if err != nil {
		t.Fatalf("getAuthentikUserPK() error = %v, want nil", err)
	}
	if userPK != 123 {
		t.Fatalf("getAuthentikUserPK() = %d, want 123", userPK)
	}
}

func TestClaimSeatInvitationCompletesPendingSeat(t *testing.T) {
	dependencies, store, authentik, token, details := newClaimTestDependencies()

	reply, err := claimSeatInvitation(t.Context(), token, details, dependencies)
	if err != nil {
		t.Fatalf("claim invitation: %v", err)
	}
	if !reply.Success || reply.SubscriptionID != "sub_123" || reply.ProductName != "Cloud Storage" || reply.StorageAmount != 100 || reply.StorageUnit != "GB" {
		t.Fatalf("unexpected claim reply: %+v", reply)
	}
	if store.beginCalls != 1 || store.completeCalls != 1 {
		t.Fatalf("expected one begin and complete call, got begin=%d complete=%d", store.beginCalls, store.completeCalls)
	}
	if authentik.findCalls != 1 || authentik.addCalls != 1 || authentik.removeCalls != 0 {
		t.Fatalf("unexpected Authentik calls: find=%d add=%d remove=%d", authentik.findCalls, authentik.addCalls, authentik.removeCalls)
	}
}

func TestClaimSeatInvitationRollsBackAuthentikMembershipAfterDatabaseFailure(t *testing.T) {
	dependencies, store, authentik, token, details := newClaimTestDependencies()
	store.completeErr = errors.New("database unavailable")
	store.lookupSeat = mongomanager.Seat{Status: mongomanager.SeatStatusClaiming}

	if _, err := claimSeatInvitation(t.Context(), token, details, dependencies); !errors.Is(err, store.completeErr) {
		t.Fatalf("claim error = %v, want %v", err, store.completeErr)
	}
	if store.completeCalls != 1 || authentik.addCalls != 1 || authentik.removeCalls != 1 {
		t.Fatalf("expected activation failure to roll back membership, complete=%d add=%d remove=%d", store.completeCalls, authentik.addCalls, authentik.removeCalls)
	}
}

func TestClaimSeatInvitationRecoversWhenDatabaseActivationAlreadySucceeded(t *testing.T) {
	dependencies, store, authentik, token, details := newClaimTestDependencies()
	store.completeErr = errors.New("database response lost")
	store.lookupSeat = mongomanager.Seat{
		Status:          mongomanager.SeatStatusActive,
		AuthentikUserPK: 42,
	}

	reply, err := claimSeatInvitation(t.Context(), token, details, dependencies)
	if err != nil {
		t.Fatalf("claim should recover active seat: %v", err)
	}
	if !reply.Success || authentik.removeCalls != 0 {
		t.Fatalf("expected successful recovery without membership rollback, reply=%+v removeCalls=%d", reply, authentik.removeCalls)
	}
}

func TestClaimSeatInvitationRollsBackWhenAuthentikAddFails(t *testing.T) {
	dependencies, store, authentik, token, details := newClaimTestDependencies()
	authentik.addErr = errors.New("group API unavailable")

	if _, err := claimSeatInvitation(t.Context(), token, details, dependencies); !errors.Is(err, authentik.addErr) {
		t.Fatalf("claim error = %v, want %v", err, authentik.addErr)
	}
	if store.completeCalls != 0 || authentik.addCalls != 1 || authentik.removeCalls != 1 {
		t.Fatalf("unexpected failure handling: complete=%d add=%d remove=%d", store.completeCalls, authentik.addCalls, authentik.removeCalls)
	}
}

func newClaimTestDependencies() (invitationClaimDependencies, *fakeInvitationStore, *fakeInvitationAuthentik, string, authmanager.TokenDetails) {
	token := strings.Repeat("a", 64)
	expiresAt := time.Now().UTC().Add(time.Hour)
	store := &fakeInvitationStore{
		seat: mongomanager.Seat{
			SubscriptionID:      "sub_123",
			EMail:               "user@example.com",
			Roles:               []mongomanager.Role{mongomanager.RoleUser},
			Status:              mongomanager.SeatStatusPending,
			InvitationTokenHash: hashSeatInvitationToken(token),
			InvitationExpiresAt: &expiresAt,
		},
		claimingSeat: mongomanager.Seat{
			SubscriptionID:  "sub_123",
			EMail:           "user@example.com",
			UID:             "user-123",
			AuthentikUserPK: 42,
			Roles:           []mongomanager.Role{mongomanager.RoleUser},
			Status:          mongomanager.SeatStatusClaiming,
		},
	}
	authentik := &fakeInvitationAuthentik{user: authentikmanager.User{PK: 42, Email: "user@example.com", IsActive: true}}
	stripeClient := &fakeInvitationStripe{
		subscription: &stripe.Subscription{
			ID:     "sub_123",
			Status: stripe.SubscriptionStatusActive,
			Items: &stripe.SubscriptionItemList{Data: []*stripe.SubscriptionItem{{
				Price: &stripe.Price{Product: &stripe.Product{ID: "prod_123"}},
			}}},
		},
		product: &stripe.Product{
			ID:       "prod_123",
			Name:     "Cloud Storage",
			Metadata: map[string]string{"storageAmount": "100", "storageUnit": "GB"},
		},
	}
	return invitationClaimDependencies{
		store:     store,
		authentik: authentik,
		stripe:    stripeClient,
		logger:    zap.NewNop(),
	}, store, authentik, token, authmanager.TokenDetails{
		UID:           "user-123",
		EMail:         " USER@example.com ",
		EmailVerified: true,
	}
}

type fakeInvitationStore struct {
	seat            mongomanager.Seat
	claimingSeat    mongomanager.Seat
	lookupSeat      mongomanager.Seat
	getByTokenErr   error
	beginErr        error
	completeErr     error
	getByTokenCalls int
	beginCalls      int
	completeCalls   int
}

func (store *fakeInvitationStore) GetSeatByInvitationTokenHash(context.Context, string) (mongomanager.Seat, error) {
	store.getByTokenCalls++
	if store.getByTokenCalls == 1 {
		return store.seat, store.getByTokenErr
	}
	return store.claimingSeat, store.getByTokenErr
}

func (store *fakeInvitationStore) BeginSeatClaim(context.Context, string, string, string, int64, time.Time) (mongomanager.Seat, error) {
	store.beginCalls++
	return store.claimingSeat, store.beginErr
}

func (store *fakeInvitationStore) CompleteSeatClaim(context.Context, string, string) error {
	store.completeCalls++
	return store.completeErr
}

func (store *fakeInvitationStore) GetSeat(context.Context, string, string) (mongomanager.Seat, error) {
	return store.lookupSeat, nil
}

type fakeInvitationAuthentik struct {
	user        authentikmanager.User
	findErr     error
	addErr      error
	removeErr   error
	findCalls   int
	addCalls    int
	removeCalls int
}

func (authentik *fakeInvitationAuthentik) FindUserByEmail(context.Context, string) (authentikmanager.User, error) {
	authentik.findCalls++
	return authentik.user, authentik.findErr
}

func (authentik *fakeInvitationAuthentik) AddUserPKToSubscription(context.Context, string, int64) error {
	authentik.addCalls++
	return authentik.addErr
}

func (authentik *fakeInvitationAuthentik) RemoveUserFromSubscription(context.Context, string, int64) error {
	authentik.removeCalls++
	return authentik.removeErr
}

type fakeInvitationStripe struct {
	subscription    *stripe.Subscription
	product         *stripe.Product
	subscriptionErr error
	productErr      error
}

func (stripeClient *fakeInvitationStripe) GetSubscriptionByID(context.Context, string) (*stripe.Subscription, error) {
	return stripeClient.subscription, stripeClient.subscriptionErr
}

func (stripeClient *fakeInvitationStripe) GetProduct(context.Context, string) (*stripe.Product, error) {
	return stripeClient.product, stripeClient.productErr
}
