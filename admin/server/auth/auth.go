package auth

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/fridencao/stardata/admin"
	"github.com/fridencao/stardata/admin/server/cookies"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

const (
	authorizationCodeGrantType = "authorization_code"
	refreshTokenGrantType      = "refresh_token"
	deviceCodeGrantType        = "urn:ietf:params:oauth:grant-type:device_code"
	longLivedAccessTokenScope  = "long_lived_access_token" // nolint:gosec // custom scope to indicate long-lived access token
)

// AuthenticatorOptions provides options for Authenticator
type AuthenticatorOptions struct {
	// AuthIssuerURL is the full OIDC issuer URL (e.g. http://keycloak:8080/realms/stardata).
	// When set, it is used verbatim (supports http for private/internal deployments and the
	// /realms/<realm> path). When empty, AuthDomain is used with the Auth0-compatible
	// "https://<AuthDomain>/" fallback.
	AuthIssuerURL    string
	AuthDomain       string
	AuthClientID     string
	AuthClientSecret string
}

// Authenticator wraps functionality for admin server auth.
// It provides endpoints for login/logout, creates users, issues cookie-based auth tokens, and provides middleware for authenticating requests.
// The implementation was derived from: https://auth0.com/docs/quickstart/webapp/golang/01-login.
type Authenticator struct {
	logger  *zap.Logger
	admin   *admin.Service
	cookies *cookies.Store
	opts    *AuthenticatorOptions
	oidc    *oidc.Provider
	oauth2  oauth2.Config
}

// NewAuthenticator creates an Authenticator.
func NewAuthenticator(logger *zap.Logger, adm *admin.Service, cookieStore *cookies.Store, opts *AuthenticatorOptions) (*Authenticator, error) {
	// Resolve the OIDC issuer URL. Prefer the explicit AuthIssuerURL (supports http for private
	// Keycloak deployments with a /realms/<realm> path); fall back to the Auth0-compatible
	// "https://<AuthDomain>/" construction when AuthIssuerURL is empty.
	issuerURL := resolveIssuerURL(opts)
	oidcProvider, err := oidc.NewProvider(context.Background(), issuerURL)
	if err != nil {
		return nil, err
	}

	oauth2Config := oauth2.Config{
		ClientID:     opts.AuthClientID,
		ClientSecret: opts.AuthClientSecret,
		RedirectURL:  adm.URLs.AuthLoginCallback(),
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}

	a := &Authenticator{
		logger:  logger,
		admin:   adm,
		cookies: cookieStore,
		opts:    opts,
		oidc:    oidcProvider,
		oauth2:  oauth2Config,
	}

	return a, nil
}

// resolveIssuerURL returns the OIDC issuer URL to use for discovery.
// It prefers the explicit AuthIssuerURL (which supports http for private Keycloak deployments and
// the /realms/<realm> path), and falls back to the Auth0-compatible "https://<AuthDomain>/" form.
func resolveIssuerURL(opts *AuthenticatorOptions) string {
	if opts.AuthIssuerURL != "" {
		return opts.AuthIssuerURL
	}
	return "https://" + opts.AuthDomain + "/"
}
