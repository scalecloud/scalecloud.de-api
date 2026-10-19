package stripemanager

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/scalecloud/scalecloud.de-api/authmanager"
	"github.com/scalecloud/scalecloud.de-api/mongomanager"
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
