package cmdutil

import (
	"context"
	"fmt"
	"slices"
	"strings"

	adminv1 "github.com/fridencao/stardata/proto/gen/stardata/admin/v1"
	"github.com/fridencao/stardata/runtime/drivers"
	"github.com/fridencao/stardata/runtime/pkg/gitutil"
	"golang.org/x/sync/semaphore"
)

// GitHelper manages git operations for a project.
// It also caches the git credentials for the project.
// Do not use directly, use cmdutil.Helper to get an instance of GitHelper.
type GitHelper struct {
	h         *Helper
	org       string
	project   string
	localPath string

	// do not access gitConfig directly, use GitConfig
	gitConfig   *gitutil.Config
	gitConfigMu *semaphore.Weighted
}

func newGitHelper(h *Helper, org, project, localPath string) *GitHelper {
	return &GitHelper{
		h:           h,
		org:         org,
		project:     project,
		localPath:   localPath,
		gitConfigMu: semaphore.NewWeighted(1),
	}
}

func (g *GitHelper) GitConfig(ctx context.Context) (*gitutil.Config, error) {
	err := g.gitConfigMu.Acquire(ctx, 1)
	if err != nil {
		return nil, err
	}
	defer g.gitConfigMu.Release(1)
	if g.gitConfig != nil && !g.gitConfig.IsExpired() {
		return g.gitConfig, nil
	}

	c, err := g.h.Client()
	if err != nil {
		return nil, err
	}

	resp, err := c.GetCloneCredentials(ctx, &adminv1.GetCloneCredentialsRequest{
		Org:     g.org,
		Project: g.project,
	})
	if err != nil {
		return nil, err
	}
	if resp.GitRepoUrl == "" {
		return nil, fmt.Errorf("project %q is not connected to a git repository", g.project)
	}
	g.gitConfig = &gitutil.Config{
		Remote:            resp.GitRepoUrl,
		Username:          resp.GitUsername,
		Password:          resp.GitPassword,
		PasswordExpiresAt: resp.GitPasswordExpiresAt.AsTime(),
		DefaultBranch:     resp.GitPrimaryBranch,
		Subpath:           resp.GitSubpath,
		ManagedRepo:       resp.GitManagedRepo,
	}
	return g.gitConfig, nil
}

func SetupGitIgnore(ctx context.Context, repo drivers.RepoStore) error {
	// Ensure .gitignore exists and contains necessary entries
	contents, err := repo.Get(ctx, ".gitignore")
	if err != nil {
		if !strings.Contains(err.Error(), "no such file") {
			return err
		}
		// Create .gitignore if it does not exist
		err = repo.Put(ctx, ".gitignore", strings.NewReader(".DS_Store\n\n# StarData\n.env\ntmp\n"))
		if err != nil {
			return err
		}
		return nil
	}

	gitIgnoreContent := strings.ReplaceAll(contents, "\r\n", "\n")
	gitIgnoreEntries := strings.Split(gitIgnoreContent, "\n")
	var added bool
	for _, path := range []string{".DS_Store", ".env", "tmp"} {
		if slices.Contains(gitIgnoreEntries, path) {
			continue // already exists
		}
		added = true
		gitIgnoreContent += fmt.Sprintf("\n%s", path)
	}
	if !added {
		return nil // nothing to add
	}
	return repo.Put(ctx, ".gitignore", strings.NewReader(gitIgnoreContent))
}
