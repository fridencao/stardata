package runtime

import (
	runtimev1 "github.com/fridencao/stardata/proto/gen/stardata/runtime/v1"
)

// featureAccessPermissions maps a resource kind to the runtime permission that the
// StarData feature matrix requires in order to see it.
//
// Background: the feature matrix (design/feature-access-control.md) lets an admin turn
// features off per user/group. Those switches originally only hid UI tabs, so a user
// could still reach the data by calling the API directly or guessing a route
// (vertical privilege escalation). The admin service now grants a matching runtime
// permission per enabled feature, and this map is what enforces it server-side.
//
// Chat is not listed here: it is enforced through the existing UseAI permission, which
// every AI tool's CheckAccess already verifies.
var featureAccessPermissions = map[string]Permission{
	ResourceKindExplore: ReadDashboards,
	ResourceKindCanvas:  ReadDashboards,
	ResourceKindReport:  ReadReports,
	ResourceKindAlert:   ReadAlerts,
}

// CheckFeatureAccess reports whether the caller's feature-matrix permissions allow
// them to see the given resource.
//
// Metrics views are deliberately not gated here: they are the shared substrate for
// dashboards, reports, alerts and AI, and are already governed by ReadMetrics plus the
// publish gate. Gating them by a dashboard-only bit would break reports and chat.
func (r *Runtime) CheckFeatureAccess(claims *SecurityClaims, res *runtimev1.Resource) bool {
	if res == nil || claims == nil || claims.SkipChecks {
		return true
	}

	perm, ok := featureAccessPermissions[res.Meta.GetName().GetKind()]
	if !ok {
		return true // Not a feature-gated kind.
	}

	return claims.Can(perm)
}
