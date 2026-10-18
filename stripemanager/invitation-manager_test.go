package stripemanager

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
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
