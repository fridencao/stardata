package admin

import (
	"testing"

	"github.com/fridencao/stardata/admin/database"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRenderSemanticResource(t *testing.T) {
	metricsYAML := "type: metrics_view\nmodel: orders\nmeasures:\n  - name: revenue\n    expression: sum(amount)\n"

	cases := []struct {
		name        string
		res         *database.SemanticResource
		wantPath    string
		wantContent string
		wantErr     string
	}{
		{
			name: "metrics_view renders to metrics/ as yaml, verbatim",
			res: &database.SemanticResource{
				ResourceKind: "metrics_view",
				ResourceName: "revenue_mv",
				Definition:   mustDefinition(t, map[string]any{"raw": metricsYAML}),
			},
			wantPath:    "metrics/revenue_mv.yaml",
			wantContent: metricsYAML,
		},
		{
			name: "model with sql format renders to models/ as .sql",
			res: &database.SemanticResource{
				ResourceKind: "model",
				ResourceName: "orders",
				Definition:   mustDefinition(t, map[string]any{"raw": "SELECT 1", "format": "sql"}),
			},
			wantPath:    "models/orders.sql",
			wantContent: "SELECT 1",
		},
		{
			name: "model without sql hint renders as yaml",
			res: &database.SemanticResource{
				ResourceKind: "model",
				ResourceName: "orders",
				Definition:   mustDefinition(t, map[string]any{"raw": "type: model\nsql: SELECT 1\n"}),
			},
			wantPath:    "models/orders.yaml",
			wantContent: "type: model\nsql: SELECT 1\n",
		},
		{
			name: "unknown kind still gets a stable home",
			res: &database.SemanticResource{
				ResourceKind: "config",
				ResourceName: "rill",
				Definition:   mustDefinition(t, map[string]any{"raw": "title: My Project\n"}),
			},
			wantPath:    "config/rill.yaml",
			wantContent: "title: My Project\n",
		},
		{
			name: "missing raw field is an error, not an empty file",
			res: &database.SemanticResource{
				ResourceKind: "metrics_view",
				ResourceName: "broken",
				Definition:   mustDefinition(t, map[string]any{"model": "orders"}),
			},
			wantErr: "no \"raw\" field",
		},
		{
			name: "empty raw is an error",
			res: &database.SemanticResource{
				ResourceKind: "metrics_view",
				ResourceName: "blank",
				Definition:   mustDefinition(t, map[string]any{"raw": "   \n"}),
			},
			wantErr: "empty",
		},
		{
			name: "non-object definition is an error",
			res: &database.SemanticResource{
				ResourceKind: "metrics_view",
				ResourceName: "bad",
				Definition:   []byte(`"just a string"`),
			},
			wantErr: "not a JSON object",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, content, err := RenderSemanticResource(tc.res)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantPath, p)
			require.Equal(t, tc.wantContent, string(content))
		})
	}
}

func TestRenderSemanticResource_NilAndEmpty(t *testing.T) {
	_, _, err := RenderSemanticResource(nil)
	require.Error(t, err)

	_, _, err = RenderSemanticResource(&database.SemanticResource{
		ResourceKind: "metrics_view",
		ResourceName: "x",
		Definition:   nil,
	})
	require.Error(t, err)
}

func mustDefinition(t *testing.T, m map[string]any) []byte {
	t.Helper()
	b, err := toJSON(m)
	require.NoError(t, err)
	return b
}

func TestRenderPublishGate(t *testing.T) {
	// The gate must be emitted even with nothing visible: an absent publish.yaml
	// means "no gating" to the runtime, which would expose everything. An empty
	// allowlist is what makes visibility fail-closed.
	empty := string(RenderPublishGate(nil))
	require.Contains(t, empty, "published:")
	require.NotContains(t, empty, "- \"")

	out := string(RenderPublishGate([]string{"revenue_mv", "customer_mv"}))
	require.Contains(t, out, `- "revenue_mv"`)
	require.Contains(t, out, `- "customer_mv"`)

	// Names are quoted so ones that look like YAML scalars survive the round trip.
	tricky := string(RenderPublishGate([]string{"123", "true", `we"ird`}))
	require.Contains(t, tricky, `- "123"`)
	require.Contains(t, tricky, `- "true"`)
	require.Contains(t, tricky, `- "we\"ird"`)
}

func TestRenderPublishGate_ParsesAsTheRuntimeExpects(t *testing.T) {
	// Round-trip through the same shape runtime/publishgate.go unmarshals into, so a
	// rendering change cannot silently break the gate.
	var doc struct {
		Published []string `yaml:"published"`
	}
	require.NoError(t, yaml.Unmarshal(RenderPublishGate([]string{"a", "b"}), &doc))
	require.Equal(t, []string{"a", "b"}, doc.Published)

	require.NoError(t, yaml.Unmarshal(RenderPublishGate(nil), &doc))
	require.Empty(t, doc.Published)
}
