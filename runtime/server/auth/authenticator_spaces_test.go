package auth

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"go.uber.org/zap"
)

// TestLocalUserSpacesInToken verifies that a local user's `spaces`
// (page-visibility roles) are carried inside the issued JWT and can be
// read back — this is the data half of role-based page gating (scheme B).
func TestLocalUserSpacesInToken(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	hash, err := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	cfg := &AuthConfig{
		Provider:   "local",
		JWTSecret: "test-secret-at-least-32-bytes-long",
		LocalUsers: []LocalUser{
			{Username: "biz", PasswordHash: string(hash), Role: "viewer", Spaces: []string{"business"}},
			{Username: "all", PasswordHash: string(hash), Role: "admin", Spaces: []string{"business", "tech"}},
			{Username: "none", PasswordHash: string(hash), Role: "viewer"}, // empty -> default full
		},
	}
	a, err := NewAuthenticator(ctx, logger, cfg, "http://localhost:8080")
	if err != nil {
		t.Fatalf("NewAuthenticator: %v", err)
	}

	spacesOf := func(user string) []string {
		tok, err := a.Login(ctx, user, "pw")
		if err != nil {
			t.Fatalf("Login(%s): %v", user, err)
		}
		cp, err := a.Audience().ParseAndValidate(tok)
		if err != nil {
			t.Fatalf("ParseAndValidate(%s): %v", user, err)
		}
		return normalizeSpaces(cp.Claims("").UserAttributes["spaces"])
	}

	// business-only user -> token carries exactly ["business"]
	biz := spacesOf("biz")
	if len(biz) != 1 || biz[0] != "business" {
		t.Fatalf("biz spaces = %v, want [business]", biz)
	}

	// explicit both -> preserved
	all := spacesOf("all")
	if len(all) != 2 {
		t.Fatalf("all spaces = %v, want [business tech]", all)
	}

	// no spaces configured -> defaults to full visibility [business, tech]
	none := spacesOf("none")
	if len(none) != 2 {
		t.Fatalf("none default spaces = %v, want [business tech]", none)
	}
}
