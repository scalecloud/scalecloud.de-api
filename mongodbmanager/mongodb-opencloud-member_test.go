package mongodbmanager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestOpenCloudMemberStoresAllocationAndIdentifiersSeparatelyFromSeats(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	if err := connection.ensureOpenCloudUsageCollections(ctx); err != nil {
		t.Fatalf("ensure OpenCloud collections: %v", err)
	}
	subscriptionID := fmt.Sprintf("opencloud-member-%d", time.Now().UnixNano())
	collection := connection.Client.Database(databaseOpenCloud).Collection(collectionOpenCloudMembers)
	member := OpenCloudMember{
		SubscriptionID:  subscriptionID,
		UID:             "owner-uid",
		EMail:           "Owner@Example.com",
		Roles:           []OpenCloudRole{OpenCloudRoleOwner, OpenCloudRoleAdministrator},
		Status:          OpenCloudMemberActive,
		AuthentikUserPK: 99,
	}
	if err := connection.CreateOpenCloudMember(ctx, member); err != nil {
		t.Fatalf("create OpenCloud owner member: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = collection.DeleteMany(cleanupContext, bson.M{"subscriptionID": subscriptionID})
	})

	if err := connection.SetOpenCloudOwnerDriveID(ctx, subscriptionID, "drive-123"); err != nil {
		t.Fatalf("SetOpenCloudOwnerDriveID() error = %v", err)
	}
	if err := connection.SetOpenCloudMemberUserID(ctx, subscriptionID, 99, "user-456"); err != nil {
		t.Fatalf("SetOpenCloudMemberUserID() error = %v", err)
	}
	if err := connection.SetOpenCloudOwnerStorageBytes(ctx, subscriptionID, 1_000); err != nil {
		t.Fatalf("SetOpenCloudOwnerStorageBytes() error = %v", err)
	}

	updated, err := connection.GetOpenCloudOwner(ctx, subscriptionID)
	if err != nil || updated.OpenCloudDriveID != "drive-123" || updated.OpenCloudUserID != "user-456" || updated.OpenCloudStorageBytes != 1_000 {
		t.Fatalf("OpenCloud owner member = %+v, %v", updated, err)
	}
	if updated.EMail != "owner@example.com" {
		t.Fatalf("OpenCloud owner email = %q, want normalized email", updated.EMail)
	}
	seat, err := connection.GetSeat(ctx, subscriptionID, "owner-uid")
	if err != nil {
		t.Fatalf("check generic seat collection: %v", err)
	}
	if seat.SubscriptionID != "" {
		t.Fatalf("OpenCloud member leaked into generic seat collection: %+v", seat)
	}
}

func TestOpenCloudInvitationClaimLifecycleUsesMemberCollection(t *testing.T) {
	connection, ctx := newTestMongoDBConnection(t)
	if err := connection.ensureOpenCloudUsageCollections(ctx); err != nil {
		t.Fatalf("ensure OpenCloud collections: %v", err)
	}
	subscriptionID := fmt.Sprintf("opencloud-invite-%d", time.Now().UnixNano())
	tokenHash := fmt.Sprintf("token-%d", time.Now().UnixNano())
	collection := connection.Client.Database(databaseOpenCloud).Collection(collectionOpenCloudMembers)
	expiresAt := time.Now().UTC().Add(time.Hour)
	member := OpenCloudMember{
		SubscriptionID:      subscriptionID,
		EMail:               "Person@Example.com",
		Roles:               []OpenCloudRole{OpenCloudRoleBilling},
		InvitationTokenHash: tokenHash,
		InvitationExpiresAt: &expiresAt,
	}
	if err := connection.CreatePendingOpenCloudMember(ctx, member); err != nil {
		t.Fatalf("create pending member: %v", err)
	}
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = collection.DeleteMany(cleanupContext, bson.M{"subscriptionID": subscriptionID})
	})

	claiming, err := connection.BeginOpenCloudMemberClaim(ctx, tokenHash, "person@example.com", "person-uid", 42, time.Now().UTC())
	if err != nil || claiming.Status != OpenCloudMemberClaiming || claiming.AuthentikUserPK != 42 {
		t.Fatalf("BeginOpenCloudMemberClaim() = %+v, %v", claiming, err)
	}
	if err := connection.CompleteOpenCloudMemberClaim(ctx, tokenHash, claiming.UID); err != nil {
		t.Fatalf("CompleteOpenCloudMemberClaim() error = %v", err)
	}
	active, err := connection.GetOpenCloudMember(ctx, subscriptionID, claiming.UID)
	if err != nil || active.Status != OpenCloudMemberActive || active.InvitationTokenHash != "" {
		t.Fatalf("claimed member = %+v, %v", active, err)
	}
}
