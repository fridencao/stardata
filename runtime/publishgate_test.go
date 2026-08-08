package runtime

import (
	"testing"

	runtimev1 "github.com/fridencao/stardata/proto/gen/stardata/runtime/v1"
	"github.com/stretchr/testify/require"
)

func metricsViewResource(name string) *runtimev1.Resource {
	return &runtimev1.Resource{
		Meta: &runtimev1.ResourceMeta{
			Name: &runtimev1.ResourceName{Kind: ResourceKindMetricsView, Name: name},
		},
	}
}

func dashboardResource(kind, name string, metricsViews ...string) *runtimev1.Resource {
	refs := make([]*runtimev1.ResourceName, 0, len(metricsViews))
	for _, mv := range metricsViews {
		refs = append(refs, &runtimev1.ResourceName{Kind: ResourceKindMetricsView, Name: mv})
	}
	return &runtimev1.Resource{
		Meta: &runtimev1.ResourceMeta{
			Name: &runtimev1.ResourceName{Kind: kind, Name: name},
			Refs: refs,
		},
	}
}

func TestPublishGateAllows(t *testing.T) {
	// A project that has not opted into gating exposes everything.
	require.True(t, (&PublishGate{}).Allows("anything"))
	require.True(t, (*PublishGate)(nil).Allows("anything"))

	gate := &PublishGate{Gated: true, Published: map[string]bool{"sales": true}}
	require.True(t, gate.Allows("sales"))
	require.False(t, gate.Allows("draft_sales"))

	// Opting in with an empty list hides everything.
	require.False(t, (&PublishGate{Gated: true, Published: map[string]bool{}}).Allows("sales"))
}

func TestPublishGateSubjects(t *testing.T) {
	// A metrics view gates on itself.
	require.Equal(t, []string{"sales"}, publishGateSubjects(metricsViewResource("sales")))

	// A dashboard gates on every metrics view it references.
	require.Equal(t, []string{"sales"},
		publishGateSubjects(dashboardResource(ResourceKindExplore, "sales_explore", "sales")))
	require.Equal(t, []string{"sales", "orders"},
		publishGateSubjects(dashboardResource(ResourceKindCanvas, "overview", "sales", "orders")))

	// Non-metrics-view refs are ignored.
	res := dashboardResource(ResourceKindExplore, "e", "sales")
	res.Meta.Refs = append(res.Meta.Refs, &runtimev1.ResourceName{Kind: ResourceKindModel, Name: "raw"})
	require.Equal(t, []string{"sales"}, publishGateSubjects(res))

	// A dashboard with no metrics view refs has nothing to gate on.
	require.Empty(t, publishGateSubjects(dashboardResource(ResourceKindCanvas, "empty")))
}

// TestPublishGateExemptions covers the callers that must never be gated: internal
// machinery (SkipChecks) and the Studio editing context (EditRepo).
func TestPublishGateExemptions(t *testing.T) {
	// Only kinds reachable from the business portal are gated.
	require.True(t, publishGateKinds[ResourceKindMetricsView])
	require.True(t, publishGateKinds[ResourceKindExplore])
	require.True(t, publishGateKinds[ResourceKindCanvas])
	require.False(t, publishGateKinds[ResourceKindModel])
	require.False(t, publishGateKinds[ResourceKindSource])
	require.False(t, publishGateKinds[ResourceKindAPI])

	// EditRepo is the Studio (dev) signal. Production deployments do not grant it,
	// so production serves published content to governors and business users alike.
	editor := &SecurityClaims{Permissions: []Permission{EditRepo}}
	require.True(t, editor.Can(EditRepo))

	viewer := &SecurityClaims{Permissions: []Permission{ReadObjects, ReadMetrics, ReadAPI, UseAI}}
	require.False(t, viewer.Can(EditRepo))
}
