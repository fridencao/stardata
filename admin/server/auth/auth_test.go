package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveIssuerURL(t *testing.T) {
	tests := []struct {
		name string
		opts *AuthenticatorOptions
		want string
	}{
		{
			name: "explicit http issuer is used verbatim (private Keycloak)",
			opts: &AuthenticatorOptions{AuthIssuerURL: "http://keycloak:8080/realms/stardata"},
			want: "http://keycloak:8080/realms/stardata",
		},
		{
			name: "https issuer with realm path is used verbatim",
			opts: &AuthenticatorOptions{AuthIssuerURL: "https://keycloak.host/realms/stardata"},
			want: "https://keycloak.host/realms/stardata",
		},
		{
			name: "empty issuer falls back to Auth0-compatible https domain",
			opts: &AuthenticatorOptions{AuthDomain: "example.auth0.com"},
			want: "https://example.auth0.com/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveIssuerURL(tt.opts); got != tt.want {
				t.Errorf("resolveIssuerURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiscoverEndSessionEndpoint(t *testing.T) {
	t.Run("parses end_session_endpoint from discovery doc", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/.well-known/openid-configuration" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"issuer":"http://keycloak:8080/realms/stardata","end_session_endpoint":"http://keycloak:8080/realms/stardata/protocol/openid-connect/logout"}`))
		}))
		defer srv.Close()

		got, err := discoverEndSessionEndpoint(srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "http://keycloak:8080/realms/stardata/protocol/openid-connect/logout"
		if got != want {
			t.Errorf("discoverEndSessionEndpoint() = %q, want %q", got, want)
		}
	})

	t.Run("trailing slash on issuer is normalized", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"end_session_endpoint":"http://keycloak:8080/realms/stardata/protocol/openid-connect/logout"}`))
		}))
		defer srv.Close()

		got, err := discoverEndSessionEndpoint(srv.URL + "/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := "http://keycloak:8080/realms/stardata/protocol/openid-connect/logout"
		if got != want {
			t.Errorf("discoverEndSessionEndpoint() = %q, want %q", got, want)
		}
	})

	t.Run("non-200 response returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		if _, err := discoverEndSessionEndpoint(srv.URL); err == nil {
			t.Error("expected error for non-200 discovery response, got nil")
		}
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not json`))
		}))
		defer srv.Close()

		if _, err := discoverEndSessionEndpoint(srv.URL); err == nil {
			t.Error("expected error for invalid json, got nil")
		}
	})
}
