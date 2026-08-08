package admin

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/fridencao/stardata/admin/database"
)

// StarData Phase 5: semantic resources live as rows, not files. But the runtime's
// parser and reconciler still consume files, and rewriting that whole pipeline is
// out of scope for the foundation. So the DB→file boundary is drawn here, on the
// admin side (the "route 乙" decision): a resource row is rendered back into the
// exact file the parser expects, and the runtime materializes it unchanged.
//
// The definition JSONB carries a "raw" key holding the original editor text. We
// render that verbatim rather than re-serializing structured fields, so parser
// fidelity is perfect — no YAML round-trip drift. Structured fields in the JSONB
// (for dependency queries) are additive and never authoritative for rendering.

// resourceDir maps a resource kind to the directory the parser conventionally
// expects it under. The directory is not semantically required by the parser
// (resource kind comes from the file body's `type:`), but keeping the layout
// conventional makes the materialized tree readable and matches existing projects.
var resourceDir = map[string]string{
	"source":       "sources",
	"model":        "models",
	"metrics_view": "metrics",
	"explore":      "explores",
	"canvas":       "canvas",
	"report":       "reports",
	"alert":        "alerts",
	"theme":        "themes",
	"api":          "apis",
}

// RenderSemanticResource converts a semantic resource row into the (path, content)
// pair the runtime should see. The path is repo-root-relative and starts without a
// leading slash, matching what the virtual-file transport expects.
func RenderSemanticResource(r *database.SemanticResource) (string, []byte, error) {
	if r == nil {
		return "", nil, fmt.Errorf("nil semantic resource")
	}

	raw, err := extractRawDefinition(r.Definition)
	if err != nil {
		return "", nil, fmt.Errorf("render %s/%s: %w", r.ResourceKind, r.ResourceName, err)
	}

	dir, ok := resourceDir[r.ResourceKind]
	if !ok {
		// Unknown kinds still get a stable home rather than being dropped silently.
		dir = r.ResourceKind
	}

	// Model bodies are SQL when the definition says so; everything else is YAML.
	ext := ".yaml"
	if r.ResourceKind == "model" && isSQLDefinition(r.Definition) {
		ext = ".sql"
	}

	p := path.Join(dir, r.ResourceName+ext)
	return p, []byte(raw), nil
}

// extractRawDefinition pulls the verbatim file body out of the JSONB definition.
// It requires a non-empty "raw" string: a resource with no raw body cannot be
// rendered into a file the parser can read, and silently emitting an empty file
// would produce confusing downstream parse errors.
func extractRawDefinition(definition []byte) (string, error) {
	if len(definition) == 0 {
		return "", fmt.Errorf("empty definition")
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(definition, &m); err != nil {
		return "", fmt.Errorf("definition is not a JSON object: %w", err)
	}
	rawMsg, ok := m["raw"]
	if !ok {
		return "", fmt.Errorf("definition has no \"raw\" field")
	}
	var raw string
	if err := json.Unmarshal(rawMsg, &raw); err != nil {
		return "", fmt.Errorf("\"raw\" field is not a string: %w", err)
	}
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("\"raw\" field is empty")
	}
	return raw, nil
}

// isSQLDefinition reports whether a model definition should render as a .sql file.
// A model may be expressed either as YAML (with a sql: field) or as a bare .sql
// file; the definition records which via an optional "format" hint.
func isSQLDefinition(definition []byte) bool {
	var m struct {
		Format string `json:"format"`
	}
	if err := json.Unmarshal(definition, &m); err != nil {
		return false
	}
	return strings.EqualFold(m.Format, "sql")
}

// toJSON is a small helper for tests to build JSONB fixtures.
func toJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}
