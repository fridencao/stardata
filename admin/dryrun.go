package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/runtime/drivers"
	_ "github.com/fridencao/stardata/runtime/drivers/file" // registers the "file" repo driver used for dry-run parsing
	"github.com/fridencao/stardata/runtime/parser"
	"github.com/fridencao/stardata/runtime/pkg/activity"
	rillstorage "github.com/fridencao/stardata/runtime/storage"
)

// DryRunResult carries the outcome of a publish dry-run: whether it passed, plus
// all parse errors found. The same shape feeds both the `validation_report` JSONB
// stored on the project_versions row and the live response to the governor's UI.
type DryRunResult struct {
	OK     bool     `json:"ok"`
	Errors []string `json:"errors,omitempty"`
}

// DryRunPublishVersion validates a project version by rendering its resource
// snapshot into a temp directory, opening a `file` repo driver on it, and running
// the parser. If any parse error is found the version is considered invalid.
//
// This is the "临时 dev instance dry-run" from the Phase 5 design. We opted for
// the lightest possible implementation:
//   - No actual runtime instance is created (no provisioner, no OLAP, no reconcile
//     of data connectors). The parser alone catches YAML errors, missing references,
//     and malformed SQL — which are the mistakes a governor realistically makes.
//   - Full reconcile (model materialization, source connectivity) would require
//     spinning up a real instance with an OLAP store, which is Phase 5.3 scope.
//
// The trade-off is explicit: model SQL correctness (e.g. a bad column name) will
// slip through the dry-run and only fail at reconcile time. The Q14 decision
// accepted this for the MVP, and the UI already warns "真实数据需要在运行时环境中查看".
func (s *Service) DryRunPublishVersion(ctx context.Context, projectVersionID string) (*DryRunResult, error) {
	// 1. Load the snapshot.
	resources, err := s.DB.FindProjectVersionResources(ctx, projectVersionID)
	if err != nil {
		return nil, fmt.Errorf("dry-run: load snapshot: %w", err)
	}
	if len(resources) == 0 {
		return &DryRunResult{OK: false, Errors: []string{"版本快照中没有资源"}}, nil
	}

	// 2. Render resources into a temp directory.
	tmpDir, err := os.MkdirTemp("", "stardata-dryrun-*")
	if err != nil {
		return nil, fmt.Errorf("dry-run: create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, r := range resources {
		p, data, err := RenderSemanticResource(r)
		if err != nil {
			return &DryRunResult{OK: false, Errors: []string{fmt.Sprintf("无法渲染资源 %s/%s: %v", r.ResourceKind, r.ResourceName, err)}}, nil
		}
		full := filepath.Join(tmpDir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, fmt.Errorf("dry-run: mkdir %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, data, 0o644); err != nil {
			return nil, fmt.Errorf("dry-run: write %s: %w", p, err)
		}
	}

	// Also render the rill.yaml config if one exists in the snapshot (resource_kind='config', name='rill').
	renderRillYAML(resources, tmpDir)

	// 3. Open a file repo driver on the temp directory.
	st := rillstorage.MustNew(os.TempDir(), nil)
	handle, err := drivers.Open("file", "", "dryrun", map[string]any{"dsn": tmpDir}, st, activity.NewNoopClient(), s.Logger)
	if err != nil {
		return nil, fmt.Errorf("dry-run: open file driver: %w", err)
	}
	defer handle.Close()

	repo, ok := handle.AsRepoStore("dryrun")
	if !ok {
		return nil, fmt.Errorf("dry-run: file driver does not implement RepoStore")
	}

	// 4. Run the parser.
	parseCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	p, err := parser.Parse(parseCtx, repo, "dryrun", "prod", "duckdb", false)
	if err != nil {
		return &DryRunResult{OK: false, Errors: []string{fmt.Sprintf("解析失败: %v", err)}}, nil
	}

	// 5. Collect errors.
	if p.HasParseErrors() {
		msgs := make([]string, 0, len(p.Errors))
		for _, pe := range p.Errors {
			msgs = append(msgs, fmt.Sprintf("%s: %s", pe.FilePath, pe.Message))
		}
		return &DryRunResult{OK: false, Errors: msgs}, nil
	}

	return &DryRunResult{OK: true}, nil
}

// renderRillYAML writes a minimal rill.yaml to the temp dir if one of the snapshot's
// resources is kind=config name=rill. Without it the parser warns about missing rill.yaml.
func renderRillYAML(resources []*database.SemanticResource, dir string) {
	for _, r := range resources {
		if r.ResourceKind == "config" && r.ResourceName == "rill" {
			_, data, err := RenderSemanticResource(r)
			if err == nil {
				_ = os.WriteFile(filepath.Join(dir, "rill.yaml"), data, 0o644)
			}
			return
		}
	}
	// No config/rill resource; write a minimal placeholder so the parser doesn't error.
	_ = os.WriteFile(filepath.Join(dir, "rill.yaml"), []byte("title: dry-run\n"), 0o644)
}

// dryRunReport marshals a DryRunResult into the JSONB stored on a version row.
func dryRunReport(r *DryRunResult) []byte {
	b, _ := json.Marshal(r)
	return b
}
