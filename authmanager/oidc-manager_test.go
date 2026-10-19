package authmanager

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVerifyTokenChecksSignatureAndAudience(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/application/o/test/.well-known/openid-configuration":
			writeJSONResponse(t, w, map[string]any{
				"issuer":                   "http://" + r.Host + "/application/o/test/",
				"jwks_uri":                 "http://" + r.Host + "/jwks",
				"authorization_endpoint":   "http://" + r.Host + "/authorize",
				"token_endpoint":           "http://" + r.Host + "/token",
				"response_types_supported": []string{"code"},
			})
		case "/jwks":
			writeJSONResponse(t, w, map[string]any{
				"keys": []map[string]string{{
					"kty": "RSA",
					"kid": "test-key",
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	issuer := server.URL + "/application/o/test/"
	t.Setenv("OIDC_ISSUER", issuer)
	t.Setenv("OIDC_CLIENT_ID", "test-client")

	connection, err := InitOIDCConnection(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.CheckHealth(context.Background()); err != nil {
		t.Fatalf("OIDC health check failed: %v", err)
	}

	validToken := signTestToken(t, privateKey, issuer, "test-client")
	details, err := connection.VerifyToken(context.Background(), validToken)
	if err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if details.UID != "user-123" || details.EMail != "user@example.com" || !details.EmailVerified {
		t.Fatalf("unexpected token details: %+v", details)
	}

	tokenParts := strings.Split(validToken, ".")
	if tokenParts[2][0] == 'A' {
		tokenParts[2] = "B" + tokenParts[2][1:]
	} else {
		tokenParts[2] = "A" + tokenParts[2][1:]
	}
	if _, err := connection.VerifyToken(context.Background(), strings.Join(tokenParts, ".")); err == nil {
		t.Fatal("token with an invalid signature was accepted")
	}

	wrongAudienceToken := signTestToken(t, privateKey, issuer, "another-client")
	if _, err := connection.VerifyToken(context.Background(), wrongAudienceToken); err == nil {
		t.Fatal("token with wrong audience was accepted")
	}
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode test response: %v", err)
	}
}

func signTestToken(t *testing.T, privateKey *rsa.PrivateKey, issuer, audience string) string {
	t.Helper()

	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": "test-key", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	claims, err := json.Marshal(map[string]any{
		"iss":            issuer,
		"aud":            audience,
		"sub":            "user-123",
		"email":          "user@example.com",
		"email_verified": true,
		"iat":            now.Unix(),
		"exp":            now.Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(encoded))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return encoded + "." + base64.RawURLEncoding.EncodeToString(signature)
}
