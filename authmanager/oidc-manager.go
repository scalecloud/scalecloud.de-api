package authmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
)

type TokenDetails struct {
	UID           string `json:"uid"`
	EMail         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

type OIDCConnection struct {
	issuer   string
	clientID string
	mu       sync.RWMutex
	verifier *oidc.IDTokenVerifier
}

func InitOIDCConnection(_ context.Context) (*OIDCConnection, error) {
	issuer := strings.TrimRight(os.Getenv("OIDC_ISSUER"), "/")
	clientID := os.Getenv("OIDC_CLIENT_ID")
	if issuer == "" {
		return nil, errors.New("OIDC_ISSUER is empty")
	}
	if clientID == "" {
		return nil, errors.New("OIDC_CLIENT_ID is empty")
	}

	return &OIDCConnection{
		issuer:   issuer,
		clientID: clientID,
	}, nil
}

func (connection *OIDCConnection) CheckHealth(ctx context.Context) error {
	var discovery struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	if err := getJSON(ctx, connection.issuer+"/.well-known/openid-configuration", &discovery); err != nil {
		return fmt.Errorf("OIDC discovery health check failed: %w", err)
	}
	if discovery.Issuer != connection.issuer+"/" {
		return errors.New("OIDC discovery returned an unexpected issuer")
	}
	jwksURL, err := url.Parse(discovery.JWKSURI)
	if err != nil || (jwksURL.Scheme != "https" && jwksURL.Scheme != "http") || jwksURL.Host == "" || jwksURL.User != nil {
		return errors.New("OIDC discovery returned an invalid JWKS URL")
	}
	var keySet struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := getJSON(ctx, discovery.JWKSURI, &keySet); err != nil {
		return fmt.Errorf("OIDC JWKS health check failed: %w", err)
	}
	if len(keySet.Keys) == 0 {
		return errors.New("OIDC JWKS contains no keys")
	}

	verifier := oidc.NewVerifier(discovery.Issuer, oidc.NewRemoteKeySet(context.Background(), discovery.JWKSURI), &oidc.Config{ClientID: connection.clientID})
	connection.mu.Lock()
	connection.verifier = verifier
	connection.mu.Unlock()
	return nil
}

func getJSON(ctx context.Context, endpoint string, target any) (resultErr error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		if err := response.Body.Close(); resultErr == nil && err != nil {
			resultErr = fmt.Errorf("close OIDC response body: %w", err)
		}
	}()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("endpoint returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target); err != nil {
		return err
	}
	return nil
}

func (connection *OIDCConnection) VerifyToken(ctx context.Context, rawToken string) (TokenDetails, error) {
	connection.mu.RLock()
	verifier := connection.verifier
	connection.mu.RUnlock()
	if verifier == nil {
		return TokenDetails{}, errors.New("OIDC provider is not healthy")
	}
	idToken, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return TokenDetails{}, err
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return TokenDetails{}, err
	}
	if idToken.Subject == "" {
		return TokenDetails{}, errors.New("token subject is empty")
	}

	return TokenDetails{
		UID:           idToken.Subject,
		EMail:         claims.Email,
		EmailVerified: claims.EmailVerified,
	}, nil
}
