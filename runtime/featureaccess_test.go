package runtime

import (
	"testing"

	runtimev1 "github.com/fridencao/stardata/proto/gen/stardata/runtime/v1"
	"github.com/stretchr/testify/require"
)

func TestCheckFeatureAccess(t *testing.T) {
	makeRes := func(kind, name string) *runtimev1.Resource {
		return &runtimev1.Resource{
			Meta: &runtimev1.ResourceMeta{
				Name: &runtimev1.ResourceName{Kind: kind, Name: name},
			},
		}
	}

	// SkipChecks always passes.
	t.Run("SkipChecks bypasses", func(t *testing.T) {
		rt := &Runtime{}
		claims := &SecurityClaims{SkipChecks: true}
		require.True(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindExplore, "x")))
		require.True(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindReport, "r")))
	})

	// Explore/Canvas require ReadDashboards.
	t.Run("Explore denied without ReadDashboards", func(t *testing.T) {
		rt := &Runtime{}
		claims := &SecurityClaims{Permissions: []Permission{ReadMetrics, ReadAPI, UseAI}}
		require.False(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindExplore, "x")))
		require.False(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindCanvas, "c")))
	})

	t.Run("Explore allowed with ReadDashboards", func(t *testing.T) {
		rt := &Runtime{}
		claims := &SecurityClaims{Permissions: []Permission{ReadDashboards}}
		require.True(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindExplore, "x")))
		require.True(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindCanvas, "c")))
	})

	// Reports require ReadReports.
	t.Run("Report denied without ReadReports", func(t *testing.T) {
		rt := &Runtime{}
		claims := &SecurityClaims{Permissions: []Permission{ReadDashboards, ReadAlerts}}
		require.False(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindReport, "r")))
	})

	t.Run("Report allowed with ReadReports", func(t *testing.T) {
		rt := &Runtime{}
		claims := &SecurityClaims{Permissions: []Permission{ReadReports}}
		require.True(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindReport, "r")))
	})

	// Alerts require ReadAlerts.
	t.Run("Alert denied without ReadAlerts", func(t *testing.T) {
		rt := &Runtime{}
		claims := &SecurityClaims{Permissions: []Permission{ReadDashboards, ReadReports}}
		require.False(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindAlert, "a")))
	})

	t.Run("Alert allowed with ReadAlerts", func(t *testing.T) {
		rt := &Runtime{}
		claims := &SecurityClaims{Permissions: []Permission{ReadAlerts}}
		require.True(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindAlert, "a")))
	})

	// MetricsView is NOT feature-gated (shared substrate).
	t.Run("MetricsView not gated", func(t *testing.T) {
		rt := &Runtime{}
		claims := &SecurityClaims{Permissions: []Permission{}} // no feature perms at all
		require.True(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindMetricsView, "mv")))
	})

	// Model/Source/API not gated.
	t.Run("Model not gated", func(t *testing.T) {
		rt := &Runtime{}
		claims := &SecurityClaims{Permissions: []Permission{}}
		require.True(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindModel, "m")))
		require.True(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindSource, "s")))
		require.True(t, rt.CheckFeatureAccess(claims, makeRes(ResourceKindAPI, "a")))
	})
}
