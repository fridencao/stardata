package admin

import (
	"context"
	"testing"

	"github.com/fridencao/stardata/admin/database"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The dry-run gate is what stands between a governor's mistake and a published
// version that would break the runtime. It has to catch the mistakes reliably —
// this test is the only place that pins that behavior end-to-end without a
// running Postgres.

type fakeDB struct {
	database.DB
	resources []*database.SemanticResource
}

func (f *fakeDB) FindProjectVersionResources(ctx context.Context, id string) ([]*database.SemanticResource, error) {
	return f.resources, nil
}

func TestDryRunPublishVersion(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name       string
		resources  []*database.SemanticResource
		expectOK   bool
		errSubstr  string
	}{
		{
			name:      "clean metrics view + model passes",
			resources: []*database.SemanticResource{
				semanticRow("model", "orders", `type: model
sql: SELECT 1 AS id, 100 AS amount, DATE '2024-01-01' AS event_time
`),
				semanticRow("metrics_view", "revenue_mv", `type: metrics_view
model: orders
timeseries: event_time
dimensions:
  - name: id
    column: id
measures:
  - name: total
    expression: SUM(amount)
`),
			},
			expectOK: true,
		},
		{
			name:      "malformed yaml is rejected",
			resources: []*database.SemanticResource{
				semanticRow("metrics_view", "bad", `type: metrics_view
model: orders
measures: [ this: is: invalid
`),
			},
			expectOK:  false,
			errSubstr: "",
		},
		{
			name:      "empty snapshot is rejected",
			resources: []*database.SemanticResource{},
			expectOK:  false,
			errSubstr: "没有资源",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &Service{
				DB:     &fakeDB{resources: tc.resources},
				Logger: zap.NewNop(),
			}
			res, err := svc.DryRunPublishVersion(ctx, "ver-id")
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, tc.expectOK, res.OK, "errors: %v", res.Errors)
			if tc.errSubstr != "" {
				joined := ""
				for _, e := range res.Errors {
					joined += e + "|"
				}
				require.Contains(t, joined, tc.errSubstr)
			}
		})
	}
}

func semanticRow(kind, name, raw string) *database.SemanticResource {
	def, _ := toJSON(map[string]any{"raw": raw})
	return &database.SemanticResource{
		ResourceKind: kind,
		ResourceName: name,
		Definition:   def,
		Status:       database.SemanticResourceStatusDraft,
	}
}
