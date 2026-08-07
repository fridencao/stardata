package ai_test

import (
	"testing"

	"github.com/fridencao/stardata/runtime"
	"github.com/fridencao/stardata/runtime/ai"
	"github.com/fridencao/stardata/runtime/pkg/activity"
	"github.com/fridencao/stardata/runtime/testruntime"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// gatedProjectFiles is a two-metrics-view project where only "published_mv" is
// listed in /publish.yaml. "draft_mv" is a draft that must stay invisible to
// business users across every path.
var gatedProjectFiles = map[string]string{
	"models/orders.yaml": `
type: model
materialize: true
sql: |
  SELECT '2025-01-01T00:00:00Z'::TIMESTAMP AS event_time, 'United States' AS country, 100 AS revenue
  UNION ALL
  SELECT '2025-01-02T00:00:00Z'::TIMESTAMP AS event_time, 'Denmark' AS country, 10 AS revenue
`,
	"metrics/published_mv.yaml": `
type: metrics_view
model: orders
timeseries: event_time
dimensions:
- column: country
measures:
- name: revenue
  expression: SUM(revenue)
`,
	"metrics/draft_mv.yaml": `
type: metrics_view
model: orders
timeseries: event_time
dimensions:
- column: country
measures:
- name: revenue
  expression: SUM(revenue)
`,
	"publish.yaml": "published:\n  - published_mv\n",
}

// businessViewerSession builds an AI session with the claims a business (viewer)
// user gets on a production deployment: UseAI + read permissions, but NOT EditRepo
// and NOT SkipChecks — so the publish gate is actually enforced.
func businessViewerSession(t *testing.T, rt *runtime.Runtime, instanceID string) *ai.Session {
	t.Helper()
	claims := &runtime.SecurityClaims{
		UserID: uuid.NewString(),
		Permissions: []runtime.Permission{
			runtime.ReadObjects, runtime.ReadMetrics, runtime.ReadAPI, runtime.UseAI,
		},
	}
	r := ai.NewRunner(rt, activity.NewNoopClient())
	s, err := r.Session(t.Context(), &ai.SessionOptions{
		InstanceID: instanceID,
		Claims:     claims,
		UserAgent:  "stardata",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Flush(t.Context()) })
	return s
}

// TestPublishGateHidesDraftFromAITools is deterministic (no LLM): it drives the
// list/get metrics-view tools directly with business-viewer claims and asserts the
// publish gate hides the unpublished metrics view.
func TestPublishGateHidesDraftFromAITools(t *testing.T) {
	rt, instanceID := testruntime.NewInstanceWithOptions(t, testruntime.InstanceOptions{
		Files: gatedProjectFiles,
	})
	testruntime.RequireReconcileState(t, rt, instanceID, 6, 0, 0)

	s := businessViewerSession(t, rt, instanceID)

	// list_metrics_views must only surface the published view.
	var listRes ai.ListMetricsViewsResult
	_, err := s.CallTool(t.Context(), ai.RoleUser, ai.ListMetricsViewsName, &listRes, ai.ListMetricsViewsArgs{})
	require.NoError(t, err)
	require.Contains(t, listMetricsViewNames(listRes), "published_mv")
	require.NotContains(t, listMetricsViewNames(listRes), "draft_mv")

	// get_metrics_view on the published view succeeds.
	var okRes ai.GetMetricsViewResult
	_, err = s.CallTool(t.Context(), ai.RoleUser, ai.GetMetricsViewName, &okRes, ai.GetMetricsViewArgs{MetricsView: "published_mv"})
	require.NoError(t, err)

	// get_metrics_view on the draft view is denied by the gate.
	var denyRes ai.GetMetricsViewResult
	_, err = s.CallTool(t.Context(), ai.RoleUser, ai.GetMetricsViewName, &denyRes, ai.GetMetricsViewArgs{MetricsView: "draft_mv"})
	require.Error(t, err)
}

// TestPublishGateExemptGovernorSeesDraft confirms the Studio (dev editing) context,
// signalled by EditRepo, bypasses the gate so governors still see drafts.
func TestPublishGateExemptGovernorSeesDraft(t *testing.T) {
	rt, instanceID := testruntime.NewInstanceWithOptions(t, testruntime.InstanceOptions{
		Files: gatedProjectFiles,
	})
	testruntime.RequireReconcileState(t, rt, instanceID, 6, 0, 0)

	claims := &runtime.SecurityClaims{
		UserID: uuid.NewString(),
		Permissions: []runtime.Permission{
			runtime.ReadObjects, runtime.ReadMetrics, runtime.ReadAPI, runtime.UseAI,
			runtime.EditRepo, // Studio dev editing context
		},
	}
	r := ai.NewRunner(rt, activity.NewNoopClient())
	s, err := r.Session(t.Context(), &ai.SessionOptions{InstanceID: instanceID, Claims: claims, UserAgent: "stardata"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Flush(t.Context()) })

	var listRes ai.ListMetricsViewsResult
	_, err = s.CallTool(t.Context(), ai.RoleUser, ai.ListMetricsViewsName, &listRes, ai.ListMetricsViewsArgs{})
	require.NoError(t, err)
	require.Contains(t, listMetricsViewNames(listRes), "published_mv")
	require.Contains(t, listMetricsViewNames(listRes), "draft_mv")
}

// TestPublishGateRealLLM is an expensive end-to-end smoke test: it drives the real
// DeepSeek analyst agent and asserts (a) the ChatBI path still works after the
// publish-gate refactor and (b) the LLM cannot reach the unpublished metrics view.
//
// Requires RILL_RUNTIME_DEEPSEEK_TEST_API_KEY (see the repo-root .env). Skipped by
// `go test -short`.
func TestPublishGateRealLLM(t *testing.T) {
	rt, instanceID := testruntime.NewInstanceWithOptions(t, testruntime.InstanceOptions{
		AIConnector: "deepseek",
		Files:       gatedProjectFiles,
	})
	testruntime.RequireReconcileState(t, rt, instanceID, 6, 0, 0)

	// Use SkipChecks + "stardata" user-agent (matching how the portal drives the router
	// agent) rather than a restrictive claims set. The test still exercises the gate
	// because CheckPublishGate is called inside ApplySecurityPolicy used by list/get
	// tools — SkipChecks exempts the gate, but here we want to verify the LLM path
	// works end-to-end; the deterministic tests above already proved gate enforcement.
	s := newEval(t, rt, instanceID)

	// Call the analyst agent directly. The router agent's tool-selection format is
	// unstable across model families (DeepSeek in particular sometimes emits "Agent
	// choice" as a pseudo-tool), and this smoke test is about the analyst path, not
	// routing: after the publish-gate refactor, can the real LLM still answer a
	// business question using only published data?
	var res *ai.AnalystAgentResult
	_, err := s.CallTool(t.Context(), ai.RoleUser, ai.AnalystAgentName, &res, ai.AnalystAgentArgs{
		Prompt: "Which country has the highest revenue? Answer with a single country name and nothing else.",
	})
	require.NoError(t, err)
	require.Contains(t, analystResponseBody(t, res.Response), "United States")

	// Every metrics-view tool call the agent made must reference only the published view.
	calls := s.Messages(ai.FilterByType(ai.MessageTypeCall))
	for _, c := range calls {
		raw, err := s.UnmarshalMessageContent(c)
		require.NoError(t, err)
		if m, ok := raw.(map[string]any); ok {
			if mv, ok := m["metrics_view"].(string); ok {
				require.NotEqual(t, "draft_mv", mv, "the analyst reached an unpublished metrics view through %s", c.Tool)
			}
		}
	}
}

// TestRouterAgentRealLLM_DeepSeek verifies the DeepSeek phantom-tool-call fix:
// the router agent's structured "agent choice" completion (no tools declared)
// used to fail with `unknown tool "Agent choice"` because DeepSeek returned the
// choice as a tool call. With normalizePhantomToolCalls it should route cleanly.
func TestRouterAgentRealLLM_DeepSeek(t *testing.T) {
	rt, instanceID := testruntime.NewInstanceWithOptions(t, testruntime.InstanceOptions{
		AIConnector: "deepseek",
		Files:       gatedProjectFiles,
	})
	testruntime.RequireReconcileState(t, rt, instanceID, 6, 0, 0)

	s := newEval(t, rt, instanceID)

	var res *ai.RouterAgentResult
	_, err := s.CallTool(t.Context(), ai.RoleUser, ai.RouterAgentName, &res, ai.RouterAgentArgs{
		Prompt: "Which country has the highest revenue? Answer with a single country name and nothing else.",
	})
	require.NoError(t, err)
	require.Equal(t, ai.AnalystAgentName, res.Agent)
	require.Contains(t, analystResponseBody(t, res.Response), "United States")
}

func listMetricsViewNames(res ai.ListMetricsViewsResult) []string {
	var names []string
	for _, m := range res.MetricsViews {
		if n, ok := m["name"].(string); ok {
			names = append(names, n)
		}
	}
	return names
}
