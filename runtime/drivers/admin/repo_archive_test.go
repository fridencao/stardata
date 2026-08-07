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

	"github.com/fridencao/stardata/runtime/pkg/archive"
	"github.com/stretchr/testify/require"
)

// serveArchive starts a test server that serves a tar.gz built from the given files.
//
// It emits explicit directory entries (mode 0755) for every parent path so extraction
// works regardless of file order. This mirrors a well-formed project archive; see
// TestEditableDraftFromPublishArchiveIsWritable for the publish-path archive shape,
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

// TestEditableDraftFromPublishArchiveIsWritable reproduces the exact failure seen on a
// live stack (V-3 walkthrough):
//
//	archive sync failed: archiveRepo: open .../archive/files/dashboards/draft_explore.yaml:
//	permission denied
//
// Release archives are built by archive.CreateFromBlobs (the publish flow), which emits
// only file entries with mode 0644 and no directory entries. untar used to create each
// parent directory with that *file* mode, so nested directories lost their execute bit
// and writing into them failed. Pre-creating just the top-level draft directory was not
// enough — every nested directory hit the same problem.
func TestEditableDraftFromPublishArchiveIsWritable(t *testing.T) {
	dir := t.TempDir()

	// Build the archive exactly like packageDevDraft does.
	buf, err := archive.CreateFromBlobs(context.Background(), []archive.BlobEntry{
		{Path: "/rill.yaml", Data: []byte("compiler: rill-beta\n")},
		{Path: "/metrics/published_mv.yaml", Data: []byte("type: metrics_view\n")},
		{Path: "/dashboards/draft_explore.yaml", Data: []byte("type: explore\n")},
	})
	require.NoError(t, err)
	body := buf.Bytes()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	r := &archiveRepo{tmpDir: dir, editable: true, archiveDownloadURL: srv.URL, archiveID: "asset-v1"}
	require.NoError(t, r.sync(context.Background()))

	// Nested files must have extracted.
	for _, p := range []string{"rill.yaml", "metrics/published_mv.yaml", "dashboards/draft_explore.yaml"} {
		_, err := os.Stat(filepath.Join(r.root(), p))
		require.NoError(t, err, "expected %s to be extracted", p)
	}

	// And the nested directories must remain writable so Studio edits succeed.
	for _, d := range []string{"metrics", "dashboards"} {
		probe := filepath.Join(r.root(), d, "probe.yaml")
		require.NoError(t, os.WriteFile(probe, []byte("x: 1\n"), 0o644),
			"nested dir %q must stay writable for dev draft edits", d)
	}
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
