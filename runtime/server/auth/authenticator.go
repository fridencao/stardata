package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"go.uber.org/zap"
)

// stardataIssuer is the fixed "iss" claim used for StarData's own (self-signed) JWTs.
// It does not need to be a reachable URL because HMAC validation only checks the string equality.
const stardataIssuer = "stardata"

// tokenTTL is the default lifetime of a StarData-issued access token.
const tokenTTL = 24 * time.Hour

// ErrInvalidCredentials is returned when local username/password authentication fails.
var ErrInvalidCredentials = errors.New("invalid username or password")

// Authenticator implements self-hosted authentication for the runtime.
// Depending on its Provider it either:
//   - local: validates a static user list (bcrypt) and issues JWTs,
//   - jwt:   only validates incoming JWTs signed with the shared secret (no login endpoint),
//   - oidc:  redirects to an external IdP and exchanges the authorization code for a StarData JWT.
type Authenticator struct {
	cfg        *AuthConfig
	provider   Provider
	externalURL string
	logger     *zap.Logger

	// issuer signs StarData's own JWTs (local / oidc flows). Nil for provider=jwt.
	issuer *HMACIssuer
	// audience validates StarData JWTs (all providers).
	audience TokenValidator

	// localUsers is the indexed account list (provider=local).
	localUsers map[string]LocalUser

	// oidcProvider / oauth2Config are set for provider=oidc.
	oidcProvider *oidc.Provider
	oauth2Config oauth2.Config
}

// NewAuthenticator builds an Authenticator from config.
// Returns (nil, nil) when cfg is nil (auth disabled).
func NewAuthenticator(ctx context.Context, logger *zap.Logger, cfg *AuthConfig, externalURL string) (*Authenticator, error) {
	if cfg == nil {
		return nil, nil
	}

	provider := cfg.Normalize()
	aud, err := NewHMACAudience(stardataIssuer, externalURL, cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	a := &Authenticator{
		cfg:        cfg,
		provider:   provider,
		externalURL: externalURL,
		logger:     logger,
		audience:   aud,
	}

	switch provider {
	case ProviderLocal:
		issuer, err := NewHMACIssuer(stardataIssuer, cfg.JWTSecret)
		if err != nil {
			return nil, fmt.Errorf("auth: %w", err)
		}
		a.issuer = issuer

		a.localUsers = make(map[string]LocalUser, len(cfg.LocalUsers))
		for _, u := range cfg.LocalUsers {
			if u.Username == "" {
				return nil, fmt.Errorf("auth: local_users entry missing username")
			}
			if !strings.HasPrefix(u.PasswordHash, "$2") {
				return nil, fmt.Errorf("auth: local_users %q password_hash is not a bcrypt hash", u.Username)
			}
			a.localUsers[strings.ToLower(u.Username)] = u
		}
		if len(a.localUsers) == 0 {
			return nil, fmt.Errorf("auth: provider=local requires at least one local_users entry")
		}

	case ProviderJWT:
		// Validate-only: no issuer, no login endpoint.

	case ProviderOIDC:
		if cfg.OIDC == nil {
			return nil, fmt.Errorf("auth: provider=oidc requires an oidc: block")
		}
		if cfg.OIDC.IssuerURL == "" || cfg.OIDC.ClientID == "" {
			return nil, fmt.Errorf("auth: oidc.issuer_url and oidc.client_id are required")
		}
		issuerURL := strings.TrimRight(cfg.OIDC.IssuerURL, "/") + "/"
		providerOIDC, err := oidc.NewProvider(ctx, issuerURL)
		if err != nil {
			return nil, fmt.Errorf("auth: failed to init oidc provider: %w", err)
		}
		a.oidcProvider = providerOIDC
		a.oauth2Config = oauth2.Config{
			ClientID:     cfg.OIDC.ClientID,
			ClientSecret: cfg.OIDC.ClientSecret,
			RedirectURL:  externalURL + "/auth/oidc/callback",
			Endpoint:     providerOIDC.Endpoint(),
			Scopes:       oidcScopes(cfg.OIDC.Scopes),
		}
		// OIDC users get a StarData JWT too, so we need an issuer.
		issuer, err := NewHMACIssuer(stardataIssuer, cfg.JWTSecret)
		if err != nil {
			return nil, fmt.Errorf("auth: %w", err)
		}
		a.issuer = issuer

	default:
		return nil, fmt.Errorf("auth: unknown provider %q", provider)
	}

	return a, nil
}

// Audience returns the TokenValidator used by the server interceptors.
func (a *Authenticator) Audience() TokenValidator { return a.audience }

// Enabled reports whether self-hosted auth is active.
func (a *Authenticator) Enabled() bool { return a != nil }

// Login validates local credentials and returns a signed JWT.
// Only valid for provider=local.
func (a *Authenticator) Login(ctx context.Context, username, password string) (string, error) {
	if a == nil || a.provider != ProviderLocal {
		return "", errors.New("login is only supported with provider=local")
	}
	u, ok := a.localUsers[strings.ToLower(strings.TrimSpace(username))]
	if !ok {
		return "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}
	return a.issueToken(u.Username, u.Role, u.Email, u.Name, u.SpacesOrDefault())
}

// issueToken mints a StarData JWT for the given subject/role.
// spaces is the list of page "spaces" the user may see (business | tech),
// carried in the token so the SPA can gate navigation by role.
func (a *Authenticator) issueToken(subject, role, email, name string, spaces []string) (string, error) {
	if a.issuer == nil {
		return "", errors.New("token issuer is not configured for this provider")
	}
	attrs := map[string]any{
		"id":     subject,
		"admin":  role == "admin",
		"spaces": spaces,
	}
	if email != "" {
		attrs["email"] = email
	}
	if name != "" {
		attrs["name"] = name
	}
	return a.issuer.NewToken(TokenOptions{
		AudienceURL:   a.externalURL,
		Subject:       subject,
		TTL:           tokenTTL,
		SystemPermissions: rolePermissions(role),
		Attributes:    attrs,
	})
}

// normalizeSpaces coerces the arbitrary value stored in a JWT's "spaces"
// claim (or a LocalUser.Spaces slice) into a clean []string. Tolerates
// nil, a comma-separated string, or a []interface{} produced by JSON
// decoding of the JWT payload.
func normalizeSpaces(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case []string:
		return t
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		parts := strings.Split(t, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// LoginHandler handles POST /auth/login (provider=local).
func (a *Authenticator) LoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		token, err := a.Login(r.Context(), req.Username, req.Password)
		if err != nil {
			http.Error(w, "invalid username or password", http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"token": token})
	}
}

// LogoutHandler handles POST /auth/logout.
// For stateless JWTs it clears the session cookie and returns the IdP logout URL
// (so the frontend can follow it) or a simple OK for local login.
func (a *Authenticator) LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Clear the token cookie if set.
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})
		switch {
		case a == nil, a.provider != ProviderOIDC:
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		default:
			// Redirect to IdP logout, then back to StarData login page.
			logoutURL := a.externalURL + "/login"
			if a.cfg.OIDC != nil && a.cfg.OIDC.IssuerURL != "" {
				logoutURL = strings.TrimRight(a.cfg.OIDC.IssuerURL, "/") +
					"/protocol/openid-connect/logout?post_logout_redirect_uri=" +
					url.QueryEscape(a.externalURL + "/login")
			}
			http.Redirect(w, r, logoutURL, http.StatusFound)
		}
	}
}

// RefreshHandler handles POST /auth/refresh.
// For stateless JWTs we simply re-issue if the caller is already authenticated.
// When no valid bearer is presented, it returns 401 so the client can re-login.
func (a *Authenticator) RefreshHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		cp, err := a.audience.ParseAndValidate(strings.TrimSpace(authHeader[7:]))
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}
		claims := cp.Claims("")
		role := "viewer"
		if claims.Admin() {
			role = "admin"
		}
		spaces := normalizeSpaces(claims.UserAttributes["spaces"])
		token, err := a.issueToken(claims.UserID, role, "", "", spaces)
		if err != nil {
			http.Error(w, "failed to refresh token", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"token": token})
	}
}

// OIDCLoginHandler handles GET /auth/oidc/login (provider=oidc).
func (a *Authenticator) OIDCLoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a == nil || a.provider != ProviderOIDC {
			http.Error(w, "oidc is not enabled", http.StatusNotFound)
			return
		}
		state, err := randomString(32)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "oidc_state",
			Value:    state,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int((10 * time.Minute).Seconds()),
		})
		redirect := a.oauth2Config.AuthCodeURL(state)
		http.Redirect(w, r, redirect, http.StatusFound)
	}
}

// OIDCCallbackHandler handles GET /auth/oidc/callback (provider=oidc).
func (a *Authenticator) OIDCCallbackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a == nil || a.provider != ProviderOIDC {
			http.Error(w, "oidc is not enabled", http.StatusNotFound)
			return
		}
		stateCookie, err := r.Cookie("oidc_state")
		if err != nil || stateCookie.Value == "" || r.URL.Query().Get("state") != stateCookie.Value {
			http.Error(w, "invalid oidc state", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing oidc code", http.StatusBadRequest)
			return
		}
		oauth2Token, err := a.oauth2Config.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, "oidc token exchange failed", http.StatusBadGateway)
			return
		}
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "oidc response missing id_token", http.StatusBadGateway)
			return
		}
		verifier := a.oidcProvider.Verifier(&oidc.Config{ClientID: a.oauth2Config.ClientID})
		idToken, err := verifier.Verify(r.Context(), rawIDToken)
		if err != nil {
			http.Error(w, "oidc id_token verification failed", http.StatusUnauthorized)
			return
		}
		var claims struct {
			Sub   string `json:"sub"`
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := idToken.Claims(&claims); err != nil {
			http.Error(w, "oidc claims extraction failed", http.StatusUnauthorized)
			return
		}
		subject := claims.Sub
		if subject == "" {
			subject = claims.Email
		}
		// OIDC users are granted the viewer role by default. Role mapping from IdP groups
		// can be added later (see plan 1.4 / Phase 3 RBAC).
		token, err := a.issueToken(subject, "viewer", claims.Email, claims.Name, []string{"business", "tech"})
		if err != nil {
			http.Error(w, "failed to issue token", http.StatusInternalServerError)
			return
		}
		// Hand the token back to the SPA via redirect to login page (cleans up URL).
		baseURL, _ := url.Parse(a.externalURL)
		loginURL := baseURL.ResolveReference(&url.URL{Path: "/login"})
		q := loginURL.Query()
		q.Set("token", token)
		loginURL.RawQuery = q.Encode()
		http.Redirect(w, r, loginURL.String(), http.StatusFound)
	}
}

// --- helpers ---

func oidcScopes(cfg string) []string {
	if strings.TrimSpace(cfg) == "" {
		return []string{oidc.ScopeOpenID, "email", "profile"}
	}
	return strings.Fields(cfg)
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
