package stripemanager

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"html"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/scalecloud/scalecloud.de-api/authentikmanager"
	"github.com/scalecloud/scalecloud.de-api/authmanager"
	"github.com/scalecloud/scalecloud.de-api/emailmanager"
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

func TestCreateSeatInvitationReservesBeforeSendingAndBuildsEscapedLink(t *testing.T) {
	dependencies, store, mailer := newInvitationCreateTestDependencies(nil, nil)
	dependencies.stripe.(*fakeInvitationStripe).product.Name = "<Cloud & Storage>"

	reply, err := createSeatInvitation(t.Context(), dependencies, "sub_123", " Person@Example.com ", []mongomanager.Role{mongomanager.RoleUser}, 3)
	if err != nil {
		t.Fatalf("create seat invitation: %v", err)
	}
	if !reply.Success || reply.SubscriptionID != "sub_123" {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if store.createCalls != 1 || store.maxSeats != 3 || !store.createdBeforeEmail {
		t.Fatalf("seat was not reserved before email: createCalls=%d maxSeats=%d createdBeforeEmail=%t", store.createCalls, store.maxSeats, store.createdBeforeEmail)
	}
	if store.seat.Status != mongomanager.SeatStatusPending || store.seat.EMail != "person@example.com" || store.seat.InvitationTokenHash == "" || store.seat.InvitationExpiresAt == nil {
		t.Fatalf("unexpected pending seat: %+v", store.seat)
	}
	if remaining := time.Until(*store.seat.InvitationExpiresAt); remaining < seatInvitationLifetime-time.Minute || remaining > seatInvitationLifetime+time.Minute {
		t.Fatalf("invitation lifetime is %s, want approximately %s", remaining, seatInvitationLifetime)
	}
	if mailer.calls != 1 || mailer.message.To[0] != " Person@Example.com " {
		t.Fatalf("unexpected email delivery: calls=%d message=%+v", mailer.calls, mailer.message)
	}
	if !strings.Contains(mailer.message.Body, "&lt;Cloud &amp; Storage&gt;") || strings.Contains(mailer.message.Body, "<Cloud & Storage>") {
		t.Fatalf("product name was not HTML-escaped: %s", mailer.message.Body)
	}
	claimURL := invitationClaimURLFromMessage(t, mailer.message.Body)
	token := claimURL.Query().Get("token")
	if len(token) != 43 || hashSeatInvitationToken(token) != store.seat.InvitationTokenHash {
		t.Fatalf("claim token does not match stored token hash: token length %d", len(token))
	}
	if store.cleanupCalls != 0 {
		t.Fatalf("successful email unexpectedly cleaned up reservation %d times", store.cleanupCalls)
	}
}

func TestCreateSeatInvitationCleansPendingSeatWhenEmailFails(t *testing.T) {
	sendErr := errors.New("smtp unavailable")
	dependencies, store, mailer := newInvitationCreateTestDependencies(sendErr, nil)
	_, err := createSeatInvitation(t.Context(), dependencies, "sub_123", "person@example.com", []mongomanager.Role{mongomanager.RoleUser}, 3)
	if !errors.Is(err, sendErr) {
		t.Fatalf("create seat invitation error = %v, want %v", err, sendErr)
	}
	if store.createCalls != 1 || mailer.calls != 1 || store.cleanupCalls != 1 {
		t.Fatalf("expected reserve, send, and cleanup calls; got reserve=%d send=%d cleanup=%d", store.createCalls, mailer.calls, store.cleanupCalls)
	}
	claimURL := invitationClaimURLFromMessage(t, mailer.message.Body)
	if got, want := store.cleanupTokenHash, store.seat.InvitationTokenHash; got != want {
		t.Fatalf("cleanup token hash = %q, want reserved token hash %q", got, want)
	}
	if hashSeatInvitationToken(claimURL.Query().Get("token")) != store.cleanupTokenHash {
		t.Fatal("cleanup did not target the invitation token included in the email")
	}
}

func TestCreateSeatInvitationPreservesEmailErrorWhenCleanupFails(t *testing.T) {
	sendErr := errors.New("smtp unavailable")
	cleanupErr := errors.New("database cleanup unavailable")
	dependencies, store, _ := newInvitationCreateTestDependencies(sendErr, nil)
	store.cleanupErr = cleanupErr

	if _, err := createSeatInvitation(t.Context(), dependencies, "sub_123", "person@example.com", []mongomanager.Role{mongomanager.RoleUser}, 3); !errors.Is(err, sendErr) {
		t.Fatalf("create seat invitation error = %v, want original email error %v", err, sendErr)
	}
	if store.cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d, want 1", store.cleanupCalls)
	}
}

func TestCreateSeatInvitationContinuesWhenStripeLookupFails(t *testing.T) {
	dependencies, store, mailer := newInvitationCreateTestDependencies(nil, nil)
	dependencies.stripe.(*fakeInvitationStripe).subscriptionErr = errors.New("Stripe unavailable")

	reply, err := createSeatInvitation(t.Context(), dependencies, "sub_123", "person@example.com", []mongomanager.Role{mongomanager.RoleUser}, 3)
	if err != nil {
		t.Fatalf("create invitation should continue without Stripe product details: %v", err)
	}
	if !reply.Success || mailer.calls != 1 || store.createCalls != 1 {
		t.Fatalf("unexpected fallback behavior: reply=%+v reservations=%d emails=%d", reply, store.createCalls, mailer.calls)
	}
	if !strings.Contains(mailer.message.Body, "your subscription") {
		t.Fatalf("email did not use fallback product name: %s", mailer.message.Body)
	}
}

func TestCreateSeatInvitationStopsWhenCapacityReservationFails(t *testing.T) {
	reservationErr := mongomanager.ErrSeatCapacityReached
	dependencies, store, mailer := newInvitationCreateTestDependencies(nil, reservationErr)
	if _, err := createSeatInvitation(t.Context(), dependencies, "sub_123", "person@example.com", []mongomanager.Role{mongomanager.RoleUser}, 3); !errors.Is(err, reservationErr) {
		t.Fatalf("create seat invitation error = %v, want %v", err, reservationErr)
	}
	if store.createCalls != 1 || mailer.calls != 0 || store.cleanupCalls != 0 {
		t.Fatalf("unexpected calls after reservation failure: reserve=%d send=%d cleanup=%d", store.createCalls, mailer.calls, store.cleanupCalls)
	}
}

func TestGetSeatProductDetails(t *testing.T) {
	baseSubscription := &stripe.Subscription{
		ID:     "sub_123",
		Status: stripe.SubscriptionStatusActive,
		Items: &stripe.SubscriptionItemList{Data: []*stripe.SubscriptionItem{{
			Price: &stripe.Price{Product: &stripe.Product{ID: "prod_123"}},
		}}},
	}
	baseProduct := &stripe.Product{
		ID:       "prod_123",
		Name:     "Cloud Storage",
		Metadata: map[string]string{"storageAmount": "100", "storageUnit": "GB"},
	}
	productErr := errors.New("Stripe product unavailable")
	tests := []struct {
		name            string
		subscription    *stripe.Subscription
		subscriptionErr error
		product         *stripe.Product
		productErr      error
		wantName        string
		wantAmount      int64
		wantUnit        string
		wantErr         error
	}{
		{
			name:         "active subscription",
			subscription: baseSubscription,
			product:      baseProduct,
			wantName:     "Cloud Storage",
			wantAmount:   100,
			wantUnit:     "GB",
		},
		{
			name: "trialing subscription",
			subscription: func() *stripe.Subscription {
				copy := *baseSubscription
				copy.Status = stripe.SubscriptionStatusTrialing
				return &copy
			}(),
			product:    baseProduct,
			wantName:   "Cloud Storage",
			wantAmount: 100,
			wantUnit:   "GB",
		},
		{
			name: "canceled subscription",
			subscription: func() *stripe.Subscription {
				copy := *baseSubscription
				copy.Status = stripe.SubscriptionStatusCanceled
				return &copy
			}(),
			product: baseProduct,
			wantErr: ErrInvitationSubscription,
		},
		{
			name: "subscription without product reference",
			subscription: &stripe.Subscription{
				Status: stripe.SubscriptionStatusActive,
				Items:  &stripe.SubscriptionItemList{Data: []*stripe.SubscriptionItem{{Price: &stripe.Price{}}}},
			},
			product: baseProduct,
			wantErr: errors.New("subscription product not found"),
		},
		{
			name:            "subscription lookup error",
			subscription:    baseSubscription,
			subscriptionErr: errors.New("Stripe subscription unavailable"),
			product:         baseProduct,
			wantErr:         errors.New("subscription product not found"),
		},
		{
			name:         "product lookup error is propagated",
			subscription: baseSubscription,
			productErr:   productErr,
			wantErr:      productErr,
		},
		{
			name:         "invalid storage amount",
			subscription: baseSubscription,
			product: &stripe.Product{
				Name:     "Cloud Storage",
				Metadata: map[string]string{"storageAmount": "many", "storageUnit": "GB"},
			},
			wantErr: errors.New("subscription storage amount not found"),
		},
		{
			name:         "incomplete product metadata",
			subscription: baseSubscription,
			product: &stripe.Product{
				Name:     "Cloud Storage",
				Metadata: map[string]string{"storageAmount": "100"},
			},
			wantErr: errors.New("subscription product details are incomplete"),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			stripeClient := &fakeInvitationStripe{
				subscription:    testCase.subscription,
				product:         testCase.product,
				subscriptionErr: testCase.subscriptionErr,
				productErr:      testCase.productErr,
			}
			name, amount, unit, err := getSeatProductDetails(t.Context(), stripeClient, "sub_123")
			if testCase.wantErr != nil {
				if err == nil || err.Error() != testCase.wantErr.Error() {
					t.Fatalf("getSeatProductDetails() error = %v, want %v", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("getSeatProductDetails() error = %v", err)
			}
			if name != testCase.wantName || amount != testCase.wantAmount || unit != testCase.wantUnit {
				t.Fatalf("getSeatProductDetails() = (%q, %d, %q), want (%q, %d, %q)", name, amount, unit, testCase.wantName, testCase.wantAmount, testCase.wantUnit)
			}
		})
	}
}

func newInvitationCreateTestDependencies(sendErr, reserveErr error) (invitationCreateDependencies, *fakeInvitationCreateStore, *fakeInvitationCreateMailer) {
	store := &fakeInvitationCreateStore{createErr: reserveErr}
	mailer := &fakeInvitationCreateMailer{store: store, sendErr: sendErr}
	stripeClient := &fakeInvitationStripe{
		subscription: &stripe.Subscription{
			ID: "sub_123",
			Items: &stripe.SubscriptionItemList{Data: []*stripe.SubscriptionItem{{
				Price: &stripe.Price{Product: &stripe.Product{ID: "prod_123"}},
			}}},
		},
		product: &stripe.Product{ID: "prod_123", Name: "Cloud Storage"},
	}
	return invitationCreateDependencies{
		store:    store,
		stripe:   stripeClient,
		mailer:   mailer,
		claimURL: "https://www.example.com/claim-invite",
		logger:   zap.NewNop(),
	}, store, mailer
}

func invitationClaimURLFromMessage(t *testing.T, body string) *url.URL {
	t.Helper()
	const hrefPrefix = `href="`
	start := strings.Index(body, hrefPrefix)
	if start < 0 {
		t.Fatalf("email body has no claim link: %s", body)
	}
	start += len(hrefPrefix)
	end := strings.IndexByte(body[start:], '"')
	if end < 0 {
		t.Fatalf("claim link is not terminated: %s", body)
	}
	claimURL, err := url.Parse(html.UnescapeString(body[start : start+end]))
	if err != nil {
		t.Fatalf("parse claim URL: %v", err)
	}
	return claimURL
}

type fakeInvitationCreateStore struct {
	seat               mongomanager.Seat
	createErr          error
	cleanupErr         error
	createCalls        int
	cleanupCalls       int
	cleanupTokenHash   string
	maxSeats           int64
	createdBeforeEmail bool
}

func (store *fakeInvitationCreateStore) CreatePendingSeatWithinCapacity(_ context.Context, seat mongomanager.Seat, maxSeats int64) error {
	store.createCalls++
	store.maxSeats = maxSeats
	if store.createErr == nil {
		store.seat = seat
	}
	return store.createErr
}

func (store *fakeInvitationCreateStore) DeletePendingSeatByInvitationTokenHash(_ context.Context, tokenHash string) error {
	store.cleanupCalls++
	store.cleanupTokenHash = tokenHash
	return store.cleanupErr
}

type fakeInvitationCreateMailer struct {
	store   *fakeInvitationCreateStore
	message emailmanager.EMail
	sendErr error
	calls   int
}

func (mailer *fakeInvitationCreateMailer) SendEMail(message emailmanager.EMail) error {
	mailer.calls++
	mailer.message = message
	mailer.store.createdBeforeEmail = mailer.store.createCalls == 1 && mailer.store.createErr == nil
	return mailer.sendErr
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
	productCalls    int
}

func (stripeClient *fakeInvitationStripe) GetSubscriptionByID(context.Context, string) (*stripe.Subscription, error) {
	return stripeClient.subscription, stripeClient.subscriptionErr
}

func (stripeClient *fakeInvitationStripe) GetProduct(context.Context, string) (*stripe.Product, error) {
	stripeClient.productCalls++
	return stripeClient.product, stripeClient.productErr
}
