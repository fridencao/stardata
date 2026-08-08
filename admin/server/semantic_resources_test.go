package server

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// collectSemanticRefs is what the draft layer's reference check is built on, so its
// coverage of each resource kind is what determines whether a governor gets told
// about a dangling reference before publishing or after.
func TestCollectSemanticRefs(t *testing.T) {
	cases := []struct {
		name string
		kind string
		body string
		want []string
	}{
		{
			name: "metrics_view references its model",
			kind: "metrics_view",
			body: "type: metrics_view\nmodel: orders\n",
			want: []string{"orders"},
		},
		{
			name: "metrics_view without a model has no refs",
			kind: "metrics_view",
			body: "type: metrics_view\ntable: raw_orders\n",
			want: nil,
		},
		{
			name: "explore references a single metrics view",
			kind: "explore",
			body: "type: explore\nmetrics_view: revenue_mv\n",
			want: []string{"revenue_mv"},
		},
		{
			name: "canvas references multiple metrics views",
			kind: "canvas",
			body: "type: canvas\nmetrics_views:\n  - revenue_mv\n  - customer_mv\n",
			want: []string{"revenue_mv", "customer_mv"},
		},
		{
			name: "alert references a metrics view",
			kind: "alert",
			body: "type: alert\nmetrics_view: revenue_mv\n",
			want: []string{"revenue_mv"},
		},
		{
			name: "model has no tracked refs at this layer",
			kind: "model",
			body: "type: model\nsql: SELECT 1\n",
			want: nil,
		},
		{
			name: "blank ref values are ignored rather than reported as missing",
			kind: "metrics_view",
			body: "type: metrics_view\nmodel: \"  \"\n",
			want: nil,
		},
		{
			name: "unknown kind yields nothing",
			kind: "theme",
			body: "type: theme\ncolors:\n  primary: blue\n",
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(tc.body), &body))
			require.Equal(t, tc.want, collectSemanticRefs(tc.kind, body))
		})
	}
}

func TestCollectSemanticRefs_NilBody(t *testing.T) {
	// A bare-SQL model is never parsed into a body, so nil must be safe.
	require.Nil(t, collectSemanticRefs("model", nil))
	require.Nil(t, collectSemanticRefs("metrics_view", nil))
}

func TestValidSemanticResourceKinds(t *testing.T) {
	// The map must stay in step with the semantic_resource_kind DB enum, otherwise a
	// save would pass this check and then fail on a raw constraint violation.
	for _, k := range []string{
		"source", "model", "metrics_view", "explore", "canvas",
		"report", "alert", "theme", "api", "config",
	} {
		require.True(t, validSemanticResourceKinds[k], "kind %q should be valid", k)
	}
	require.False(t, validSemanticResourceKinds["dashboard"])
	require.False(t, validSemanticResourceKinds[""])
}
