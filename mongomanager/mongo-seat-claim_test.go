package mongomanager

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestSeatClaimStateTransitions(t *testing.T) {
	connection, ctx := newTestMongoConnection(t)
	subscriptionID := fmt.Sprintf("claim-state-test-%d", time.Now().UnixNano())
	tokenHash := fmt.Sprintf("claim-token-%d", time.Now().UnixNano())
	expiresAt := time.Now().UTC().Add(time.Hour)
	seat := Seat{
		SubscriptionID:      subscriptionID,
		EMail:               "claimant@example.com",
		Roles:               []Role{RoleUser},
		InvitationTokenHash: tokenHash,
		InvitationExpiresAt: &expiresAt,
	}
	if err := connection.CreatePendingSeat(ctx, seat); err != nil {
		t.Fatalf("create pending seat: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := connection.DeletePendingSeatByInvitationTokenHash(cleanupContext, tokenHash); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			t.Errorf("delete remaining pending seat: %v", err)
		}
		if _, err := connection.Client.Database(databaseSubscription).Collection(collectionSeats).DeleteOne(cleanupContext, bson.M{"subscriptionID": subscriptionID}); err != nil {
			t.Errorf("delete test seat: %v", err)
		}
	})

	if _, err := connection.BeginSeatClaim(ctx, tokenHash, "different@example.com", "user-123", 42, time.Now().UTC()); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("BeginSeatClaim() with mismatched email error = %v, want ErrNoDocuments", err)
	}
	if _, err := connection.BeginSeatClaim(ctx, tokenHash, seat.EMail, "user-123", 42, time.Now().UTC()); err != nil {
		t.Fatalf("begin valid claim: %v", err)
	}
	claimingSeat, err := connection.GetSeatByInvitationTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("read claiming seat: %v", err)
	}
	if claimingSeat.Status != SeatStatusClaiming || claimingSeat.UID != "user-123" || claimingSeat.AuthentikUserPK != 42 || claimingSeat.EMailVerified == nil || !*claimingSeat.EMailVerified {
		t.Fatalf("unexpected claiming seat state: %+v", claimingSeat)
	}
	if _, err := connection.BeginSeatClaim(ctx, tokenHash, seat.EMail, "other-user", 43, time.Now().UTC()); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("replayed BeginSeatClaim() error = %v, want ErrNoDocuments", err)
	}
	if err := connection.CompleteSeatClaim(ctx, tokenHash, "other-user"); err == nil {
		t.Fatal("CompleteSeatClaim() accepted a different user")
	}
	if err := connection.CompleteSeatClaim(ctx, tokenHash, "user-123"); err != nil {
		t.Fatalf("complete valid claim: %v", err)
	}
	activeSeat, err := connection.GetSeat(ctx, subscriptionID, "user-123")
	if err != nil {
		t.Fatalf("read activated seat: %v", err)
	}
	if activeSeat.Status != SeatStatusActive || activeSeat.AuthentikUserPK != 42 {
		t.Fatalf("unexpected active seat state: %+v", activeSeat)
	}
	missingSeat, err := connection.GetSeatByInvitationTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("lookup completed invitation token: %v", err)
	}
	if missingSeat.SubscriptionID != "" {
		t.Fatalf("completed claim retained its token hash: seat = %+v", missingSeat)
	}
	if err := connection.CompleteSeatClaim(ctx, tokenHash, "user-123"); err == nil {
		t.Fatal("CompleteSeatClaim() allowed replay after activation")
	}
}

func TestBeginSeatClaimRejectsExpiredInvitation(t *testing.T) {
	connection, ctx := newTestMongoConnection(t)
	subscriptionID := fmt.Sprintf("expired-claim-test-%d", time.Now().UnixNano())
	tokenHash := fmt.Sprintf("expired-token-%d", time.Now().UnixNano())
	expiresAt := time.Now().UTC().Add(-time.Minute)
	seat := Seat{
		SubscriptionID:      subscriptionID,
		EMail:               "claimant@example.com",
		Roles:               []Role{RoleUser},
		InvitationTokenHash: tokenHash,
		InvitationExpiresAt: &expiresAt,
	}
	if err := connection.CreatePendingSeat(ctx, seat); err != nil {
		t.Fatalf("create expired pending seat: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := connection.Client.Database(databaseSubscription).Collection(collectionSeats).DeleteOne(cleanupContext, bson.M{"subscriptionID": subscriptionID}); err != nil {
			t.Errorf("delete expired test seat: %v", err)
		}
	})

	if _, err := connection.BeginSeatClaim(ctx, tokenHash, seat.EMail, "user-123", 42, time.Now().UTC()); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("BeginSeatClaim() for expired invitation error = %v, want ErrNoDocuments", err)
	}
	storedSeat, err := connection.GetSeatByInvitationTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("read expired seat: %v", err)
	}
	if storedSeat.Status != SeatStatusPending || storedSeat.UID != "" {
		t.Fatalf("expired seat was modified: %+v", storedSeat)
	}
}
