package admin

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
)

func TestNewGithubNotConfigured(t *testing.T) {
	gh, err := NewGithub(context.Background(), 0, "", "", zap.NewNop())
	if err != nil {
		t.Fatalf("NewGithub with empty config returned error: %v", err)
	}
	if _, ok := gh.(githubNoop); !ok {
		t.Fatalf("expected githubNoop client, got %T", gh)
	}
	if _, _, err := gh.InstallationToken(context.Background(), 1, 1); !errors.Is(err, ErrGithubNotConfigured) {
		t.Errorf("InstallationToken error = %v, want ErrGithubNotConfigured", err)
	}
	if _, err := gh.ManagedOrgInstallationID(); !errors.Is(err, ErrGithubNotConfigured) {
		t.Errorf("ManagedOrgInstallationID error = %v, want ErrGithubNotConfigured", err)
	}
	if _, err := gh.CreateManagedRepo(context.Background(), "x", false); !errors.Is(err, ErrGithubNotConfigured) {
		t.Errorf("CreateManagedRepo error = %v, want ErrGithubNotConfigured", err)
	}
}
