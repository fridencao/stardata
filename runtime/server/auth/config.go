package auth

import (
	"strings"

	runtime "github.com/fridencao/stardata/runtime"
)

// Provider identifies the authentication strategy used by the self-hosted authenticator.
type Provider string

const (
	// ProviderLocal authenticates users against a static list of local accounts (username + bcrypt password hash).
	ProviderLocal Provider = "local"
	// ProviderOIDC authenticates users against an external OIDC identity provider (Keycloak / Authing / self-hosted).
	ProviderOIDC Provider = "oidc"
	// ProviderJWT only validates incoming JWTs signed with the shared secret (use when an external system issues tokens).
	ProviderJWT Provider = "jwt"
)

// AuthConfig is the self-hosted authentication configuration.
// It is populated from environment variables (RILL_RUNTIME_AUTH_*) and/or the `auth:` block
// of the YAML config file mounted via STARDATA_CONFIG.
type AuthConfig struct {
	// Provider selects the authentication strategy (local | oidc | jwt). Defaults to "local".
	Provider string `yaml:"provider" split_words:"true"`

	// JWTSecret is the HMAC-SHA256 signing secret used to issue and validate StarData's own JWTs.
	// Required for provider=local and provider=jwt. Use a random string of at least 32 bytes.
	JWTSecret string `yaml:"jwt_secret" split_words:"true"`

	// OIDC holds the OIDC provider settings (only used when Provider=oidc).
	OIDC *OIDCConfig `yaml:"oidc"`

	// LocalUsers is the static account list (only used when Provider=local).
	LocalUsers []LocalUser `yaml:"local_users"`
}

// OIDCConfig holds OIDC connection settings.
type OIDCConfig struct {
	// IssuerURL is the OIDC issuer (e.g. https://auth.company.com/realms/stardata).
	IssuerURL string `yaml:"issuer_url" split_words:"true"`
	// ClientID is the OIDC client / application ID registered with the IdP.
	ClientID string `yaml:"client_id" split_words:"true"`
	// ClientSecret is the OIDC client secret.
	ClientSecret string `yaml:"client_secret" split_words:"true"`
	// Scopes is an optional space-separated list of OIDC scopes (default "openid profile email").
	Scopes string `yaml:"scopes" split_words:"true"`
}

// LocalUser is a static local account.
type LocalUser struct {
	Username     string `yaml:"username"`
	PasswordHash string `yaml:"password_hash"` // bcrypt hash
	Role         string `yaml:"role"`           // admin | editor | viewer
	Email        string `yaml:"email"`
	Name         string `yaml:"name"`
}

// Normalize returns the resolved Provider, defaulting to local.
func (c *AuthConfig) Normalize() Provider {
	p := Provider(strings.ToLower(strings.TrimSpace(c.Provider)))
	if p == "" {
		return ProviderLocal
	}
	return p
}

// IsConfigured reports whether any auth setting was actually provided.
// envconfig unconditionally allocates nil struct pointers while processing,
// so an all-zero AuthConfig means auth was never configured and must be
// treated as disabled (otherwise `stardata start` without any auth config
// would fail with "jwt secret must not be empty").
func (c *AuthConfig) IsConfigured() bool {
	if c == nil {
		return false
	}
	return c.Provider != "" || c.JWTSecret != "" || c.OIDC.isConfigured() || len(c.LocalUsers) > 0
}

// isConfigured reports whether any OIDC field was actually provided.
func (c *OIDCConfig) isConfigured() bool {
	if c == nil {
		return false
	}
	return c.IssuerURL != "" || c.ClientID != "" || c.ClientSecret != "" || c.Scopes != ""
}

// rolePermissions maps a role to the runtime permissions it grants.
// admin gets ManageInstances, which (per jwtClaims.Claims) flips SkipChecks=true → full access.
func rolePermissions(role string) []runtime.Permission {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin":
		return []runtime.Permission{runtime.ManageInstances}
	case "editor":
		return []runtime.Permission{
			runtime.ReadInstance,
			runtime.ReadRepo,
			runtime.EditRepo,
			runtime.ReadObjects,
			runtime.ReadOLAP,
			runtime.ReadMetrics,
			runtime.ReadProfiling,
			runtime.ReadAPI,
			runtime.ReadResolvers,
			runtime.UseAI,
			runtime.EditTrigger,
			runtime.ManageInstance,
		}
	case "viewer", "":
		fallthrough
	default:
		return []runtime.Permission{
			runtime.ReadInstance,
			runtime.ReadRepo,
			runtime.ReadObjects,
			runtime.ReadOLAP,
			runtime.ReadMetrics,
			runtime.ReadProfiling,
			runtime.ReadAPI,
			runtime.ReadResolvers,
			runtime.UseAI,
		}
	}
}
