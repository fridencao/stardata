package auth

import (
	"context"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"go.uber.org/zap"
)

func testAuthConfig(t *testing.T, role, password string) *AuthConfig {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt hash: %v", err)
	}
	return &AuthConfig{
		Provider:   "local",
		JWTSecret: "test-secret-at-least-32-bytes-long",
		LocalUsers: []LocalUser{
			{Username: "admin", PasswordHash: string(hash), Role: role},
		},
	}
}

func TestLocalLoginRoundTrip(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	cfg := testAuthConfig(t, "admin", "s3cr3t")

	a, err := NewAuthenticator(ctx, logger, cfg, "http://localhost:8080")
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}
	if !a.Enabled() {
		t.Fatal("expected authenticator to be enabled")
	}

	// Valid credentials -> token that validates as admin (full access).
	tok, err := a.Login(ctx, "admin", "s3cr3t")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	cp, err := a.Audience().ParseAndValidate(tok)
	if err != nil {
		t.Fatalf("ParseAndValidate: %v", err)
	}
	claims := cp.Claims("")
	if claims.UserID != "admin" {
		t.Fatalf("UserID = %q, want admin", claims.UserID)
	}
	if !claims.Admin() {
		t.Fatal("expected admin claim")
	}
	if !claims.Can(0x11) { // ReadInstance
		t.Fatal("admin should have ReadInstance")
	}

	// Wrong password -> ErrInvalidCredentials.
	if _, err := a.Login(ctx, "admin", "wrong"); err == nil || !strings.Contains(err.Error(), "invalid username or password") {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	// Unknown user -> ErrInvalidCredentials.
	if _, err := a.Login(ctx, "ghost", "x"); err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestViewerRolePermissions(t *testing.T) {
	perms := rolePermissions("viewer")
	for _, p := range perms {
		if p == 0x11 { // ReadInstance
			return
		}
	}
	t.Fatal("viewer should have ReadInstance")
	if len(rolePermissions("admin")) != 1 {
		t.Fatal("admin should be a single ManageInstances permission")
	}
}

func TestJWTProviderValidateOnly(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	secret := "test-secret-at-least-32-bytes-long"

	// Issuer side (external system mints a token).
	iss, err := NewHMACIssuer(stardataIssuer, secret)
	if err != nil {
		t.Fatalf("NewHMACIssuer: %v", err)
	}
	issued, err := iss.NewToken(TokenOptions{
		AudienceURL:   "http://localhost:8080",
		Subject:       "svc",
		TTL:           tokenTTL,
		SystemPermissions: rolePermissions("viewer"),
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	// Validate-only provider (provider=jwt) must accept it.
	cfg := &AuthConfig{Provider: "jwt", JWTSecret: secret}
	a, err := NewAuthenticator(ctx, logger, cfg, "http://localhost:8080")
	if err != nil {
		t.Fatalf("NewAuthenticator(jwt): %v", err)
	}
	cp, err := a.Audience().ParseAndValidate(issued)
	if err != nil {
		t.Fatalf("validate external token: %v", err)
	}
	if cp.Claims("").UserID != "svc" {
		t.Fatal("expected subject svc")
	}

	// Wrong secret must reject.
	badCfg := &AuthConfig{Provider: "jwt", JWTSecret: "different-secret-also-at-least-32-byt"}
	bad, _ := NewAuthenticator(ctx, logger, badCfg, "http://localhost:8080")
	if _, err := bad.Audience().ParseAndValidate(issued); err == nil {
		t.Fatal("expected rejection with wrong secret")
	}
}
