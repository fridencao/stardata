package admin

import (
	"testing"

	"github.com/fridencao/stardata/admin/database"
	"github.com/stretchr/testify/require"
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
