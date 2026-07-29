package admin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fridencao/stardata/runtime/pkg/archive"
)

// archiveRepo represents a tarball archive file source.
// It is unsafe for concurrent reads and writes.
//
// In editable mode (dev deployments), files are extracted into a persistent draft directory:
// local edits survive restarts and are only replaced when a new archive version
// (i.e. a different archive ID) is published.
type archiveRepo struct {
	h *Handle
	// tmpDir is the parent directory for downloads and extracted files.
	// Despite the name, it is a persistent data directory in editable mode.
	tmpDir             string
	editable           bool
	archiveDownloadURL string
	archiveID          string
	archiveCreatedOn   time.Time

	filesDir          string
	syncedDownloadURL string
}

// syncedArchiveIDFile records the archive ID last extracted into an editable draft directory.
// It lives in tmpDir (outside filesDir), so it is never listed as a project file.
const syncedArchiveIDFile = "synced-archive-id"

func (r *archiveRepo) sync(ctx context.Context) error {
	if r.editable {
		return r.syncEditable(ctx)
	}

	if r.syncedDownloadURL == r.archiveDownloadURL {
		return nil
	}

	archivePath := filepath.Join(r.tmpDir, "archive.tar.gz")
	defer func() { _ = os.Remove(archivePath) }()

	filesDir, err := os.MkdirTemp(r.tmpDir, "files")
	if err != nil {
		return err
	}

	err = archive.Download(ctx, r.archiveDownloadURL, archivePath, filesDir, false, false)
	if err != nil {
		_ = os.RemoveAll(filesDir)
		return fmt.Errorf("archiveRepo: %w", err)
	}

	_ = os.RemoveAll(r.filesDir)
	r.filesDir = filesDir
	r.syncedDownloadURL = r.archiveDownloadURL
	return nil
}

// syncEditable extracts the archive into a persistent draft directory.
// If the currently extracted archive ID matches, it skips extraction to preserve local edits.
func (r *archiveRepo) syncEditable(ctx context.Context) error {
	filesDir := filepath.Join(r.tmpDir, "files")
	markerPath := filepath.Join(r.tmpDir, syncedArchiveIDFile)

	if b, err := os.ReadFile(markerPath); err == nil && strings.TrimSpace(string(b)) == r.archiveID {
		r.filesDir = filesDir
		r.syncedDownloadURL = r.archiveDownloadURL
		return nil
	}

	archivePath := filepath.Join(r.tmpDir, "archive.tar.gz")
	defer func() { _ = os.Remove(archivePath) }()

	// Remove the marker first so an interrupted extraction is retried on the next sync.
	_ = os.Remove(markerPath)

	// Clean extraction: a new archive version replaces the draft contents.
	err := archive.Download(ctx, r.archiveDownloadURL, archivePath, filesDir, true, false)
	if err != nil {
		return fmt.Errorf("archiveRepo: %w", err)
	}

	err = os.WriteFile(markerPath, []byte(r.archiveID), 0o644)
	if err != nil {
		return err
	}

	r.filesDir = filesDir
	r.syncedDownloadURL = r.archiveDownloadURL
	return nil
}

func (r *archiveRepo) root() string {
	return r.filesDir
}
