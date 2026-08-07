package admin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// serveArchive starts a test server that serves a tar.gz built from the given files.
//
// It emits explicit directory entries (mode 0755) for every parent path so extraction
// works regardless of file order. This mirrors a well-formed project archive; see
// TestCreateFromBlobsOmitsDirEntries for the separate finding about the publish path,
// whose archives (built via archive.CreateFromBlobs) omit directory entries.
func serveArchive(t *testing.T, files map[string]string) string {
	t.Helper()

	body := buildTarGz(t, files)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	dirs := map[string]bool{}
	for p := range files {
		d := path.Dir(strings.TrimPrefix(p, "/"))
		for d != "." && d != "/" && d != "" {
			dirs[d] = true
			d = path.Dir(d)
		}
	}
	dirList := make([]string, 0, len(dirs))
	for d := range dirs {
		dirList = append(dirList, d)
	}
	sort.Strings(dirList)

	var b strings.Builder
	gw := gzip.NewWriter(&stringWriter{&b})
	tw := tar.NewWriter(gw)
	for _, d := range dirList {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: d + "/", Typeflag: tar.TypeDir, Mode: 0o755}))
	}
	for p, data := range files {
		name := strings.TrimPrefix(p, "/")
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(data))}))
		_, err := tw.Write([]byte(data))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return []byte(b.String())
}

type stringWriter struct{ b *strings.Builder }

func (w *stringWriter) Write(p []byte) (int, error) { return w.b.Write(p) }

// TestEditableDraftSurvivesResyncOfSameArchive documents the intended behaviour:
// re-syncing while the archive ID is unchanged (e.g. a runtime restart or a routine
// handshake refresh) must preserve unpublished draft edits.
func TestEditableDraftSurvivesResyncOfSameArchive(t *testing.T) {
	dir := t.TempDir()
	url := serveArchive(t, map[string]string{"rill.yaml": "compiler: rill-beta\n"})

	r := &archiveRepo{tmpDir: dir, editable: true, archiveDownloadURL: url, archiveID: "asset-v1"}
	require.NoError(t, r.sync(context.Background()))

	draft := filepath.Join(r.root(), "metrics", "sales.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(draft), os.ModePerm))
	require.NoError(t, os.WriteFile(draft, []byte("type: metrics_view\n"), 0o644))

	// Same archive ID: the marker matches, so extraction is skipped.
	require.NoError(t, r.sync(context.Background()))

	b, err := os.ReadFile(draft)
	require.NoError(t, err, "unpublished draft edit must survive a re-sync of the same archive")
	require.Equal(t, "type: metrics_view\n", string(b))
}

// TestEditableDraftClobberedByNewArchiveID pins down R1 risk #2.
//
// Publishing changes the project's ArchiveAssetID, so the next repo handshake hands the
// dev deployment a new archive ID. syncEditable then does a clean extraction, which wipes
// the draft directory. Any edit made after the publish snapshot is silently lost — there
// is no conflict detection and no warning.
func TestEditableDraftClobberedByNewArchiveID(t *testing.T) {
	dir := t.TempDir()
	v1 := serveArchive(t, map[string]string{"rill.yaml": "compiler: rill-beta\n"})

	r := &archiveRepo{tmpDir: dir, editable: true, archiveDownloadURL: v1, archiveID: "asset-v1"}
	require.NoError(t, r.sync(context.Background()))

	draft := filepath.Join(r.root(), "metrics", "sales.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(draft), os.ModePerm))
	require.NoError(t, os.WriteFile(draft, []byte("type: metrics_view\n"), 0o644))

	// A publish elsewhere produced a new asset; the handshake updates the archive ID.
	r.archiveDownloadURL = serveArchive(t, map[string]string{"rill.yaml": "compiler: rill-beta\n"})
	r.archiveID = "asset-v2"
	require.NoError(t, r.sync(context.Background()))

	_, err := os.Stat(draft)
	require.True(t, os.IsNotExist(err),
		"KNOWN GAP (R1 risk #2): a new archive ID clean-extracts over the draft area, destroying unpublished edits")
}

// TestEditableDraftClobberedByRollback pins down R1 risk #3.
//
// Rollback points the project at an older asset. The dev deployment tracks the project's
// asset, so the draft area is reset to the rolled-back content: the governor's in-progress
// work disappears even though rollback is presented as a production-only action.
func TestEditableDraftClobberedByRollback(t *testing.T) {
	dir := t.TempDir()
	v2 := serveArchive(t, map[string]string{"rill.yaml": "compiler: rill-beta\n", "metrics/orders.yaml": "version: 2\n"})

	r := &archiveRepo{tmpDir: dir, editable: true, archiveDownloadURL: v2, archiveID: "asset-v2"}
	require.NoError(t, r.sync(context.Background()))

	draft := filepath.Join(r.root(), "metrics", "wip.yaml")
	require.NoError(t, os.WriteFile(draft, []byte("type: metrics_view\n"), 0o644))

	// Rollback to v1: the project asset goes backwards, and so does the draft area.
	r.archiveDownloadURL = serveArchive(t, map[string]string{"rill.yaml": "compiler: rill-beta\n", "metrics/orders.yaml": "version: 1\n"})
	r.archiveID = "asset-v1"
	require.NoError(t, r.sync(context.Background()))

	_, err := os.Stat(draft)
	require.True(t, os.IsNotExist(err),
		"KNOWN GAP (R1 risk #3): rollback resets the dev draft area, discarding unpublished work without warning")

	rolled, err := os.ReadFile(filepath.Join(r.root(), "metrics", "orders.yaml"))
	require.NoError(t, err)
	require.Equal(t, "version: 1\n", string(rolled), "draft area follows the rolled-back asset content")
}

// TestNonEditableArchiveAlwaysReExtracts covers the production path for contrast:
// prod deployments are not editable, so every new archive replaces the files wholesale.
func TestNonEditableArchiveAlwaysReExtracts(t *testing.T) {
	dir := t.TempDir()
	url := serveArchive(t, map[string]string{"rill.yaml": "compiler: rill-beta\n"})

	r := &archiveRepo{tmpDir: dir, editable: false, archiveDownloadURL: url, archiveID: "asset-v1"}
	require.NoError(t, r.sync(context.Background()))

	b, err := os.ReadFile(filepath.Join(r.root(), "rill.yaml"))
	require.NoError(t, err)
	require.Equal(t, "compiler: rill-beta\n", string(b))

	// A non-editable repo has no marker file, so it must not leak one into the files dir.
	_, err = os.Stat(filepath.Join(r.root(), syncedArchiveIDFile))
	require.True(t, os.IsNotExist(err))
}

// TestCreateFromBlobsOmitsDirEntries pins down an additional R1 finding.
//
// packageDevDraft (used by every publish + rollback) builds the release archive via
// archive.CreateFromBlobs, which writes only file entries. When the receiving side
// (dev or prod) runs an "untar into an empty directory", untar creates the parent
// directory with the file mode of the first entry (0644, no execute bit), so every
// subsequent write into that directory fails with EACCES.
//
// syncEditable used to trigger this: it called archive.Download(..., clean=true, ...),
// which lets untar create the top-level draft directory. We now pre-create the draft
// directory with 0755 and pass clean=false so untar only writes files into it, which
// sidesteps the issue for the editable path.
//
// The underlying archive/untar contract still allows an ill-formed archive to produce
// unwritable directories — a caller that decompresses into a fresh path without a
// pre-existing writable root will hit the same problem. Consider whether prod
// deployment redeploy is affected (their sync builds files under a MkdirTemp'd
// directory, which is writable, so the extraction succeeds; but any nested-only
// archive relies on that pre-existing writable root).
func TestCreateFromBlobsOmitsDirEntries(t *testing.T) {
	t.Skip("addressed for the editable dev path by pre-creating the draft directory in syncEditable; " +
		"prod path relies on the pre-existing MkdirTemp'd directory. See " +
		"design/phase4-review-and-hardening-r1-findings.md for the full analysis.")
}
