package authmanager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type TokenDetails struct {
	UID           string `json:"uid"`
	EMail         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

type OIDCConnection struct {
	verifier *oidc.IDTokenVerifier
}

func InitOIDCConnection(ctx context.Context) (*OIDCConnection, error) {
	issuer := strings.TrimRight(os.Getenv("OIDC_ISSUER"), "/")
	clientID := os.Getenv("OIDC_CLIENT_ID")
	if issuer == "" {
		return nil, errors.New("OIDC_ISSUER is empty")
	}
	if clientID == "" {
		return nil, errors.New("OIDC_CLIENT_ID is empty")
	}

	provider, err := oidc.NewProvider(ctx, issuer+"/")
	if err != nil {
		return nil, fmt.Errorf("initialize OIDC provider: %w", err)
	}

	return &OIDCConnection{
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

func (connection *OIDCConnection) VerifyToken(ctx context.Context, rawToken string) (TokenDetails, error) {
	idToken, err := connection.verifier.Verify(ctx, rawToken)
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
