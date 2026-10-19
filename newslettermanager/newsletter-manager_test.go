package newslettermanager

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "valid address", value: "person@example.com", valid: true},
		{name: "invalid address", value: "not-an-email"},
		{name: "empty address", value: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(test *testing.T) {
			if got := IsValidEmail(testCase.value); got != testCase.valid {
				t.Fatalf("IsValidEmail(%q) = %t, want %t", testCase.value, got, testCase.valid)
			}
		})
	}
}

func TestCanSendVerificationEmail(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		sentAt  *time.Time
		wantErr bool
	}{
		{name: "never sent", sentAt: nil},
		{name: "cooldown active", sentAt: timePointer(now.Add(-cooldownDuration / 2)), wantErr: true},
		{name: "cooldown expired", sentAt: timePointer(now.Add(-cooldownDuration - time.Second))},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(test *testing.T) {
			err := CanSendVerificationEmail(testCase.sentAt)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("CanSendVerificationEmail() error = %v, wantErr %t", err, testCase.wantErr)
			}
		})
	}
}

func TestGenerateVerificationToken(t *testing.T) {
	token, err := generateVerificationToken()
	if err != nil {
		t.Fatalf("generate verification token: %v", err)
	}
	if strings.ContainsAny(token, "+/") {
		t.Fatalf("token is not URL-safe: %q", token)
	}
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("decode verification token: %v", err)
	}
	if len(decoded) != 64 {
		t.Fatalf("decoded token has %d bytes, want 64", len(decoded))
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}
