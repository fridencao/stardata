package ai

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	runtimev1 "github.com/fridencao/stardata/proto/gen/stardata/runtime/v1"
	"github.com/fridencao/stardata/runtime"
	"github.com/fridencao/stardata/runtime/parser"
	"gopkg.in/yaml.v3"
)

const ListMetricsViewsName = "list_metrics_views"

type ListMetricsViews struct {
	Runtime *runtime.Runtime
}

var _ Tool[*ListMetricsViewsArgs, *ListMetricsViewsResult] = (*ListMetricsViews)(nil)

type ListMetricsViewsArgs struct{}

type ListMetricsViewsResult struct {
	MetricsViews []map[string]any `json:"metrics_views"`
}

func (t *ListMetricsViews) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        ListMetricsViewsName,
		Title:       "List Metrics Views",
		Description: "List all metrics views in the current project",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: boolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
			ReadOnlyHint:    true,
		},
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Listing metrics...",
			"openai/toolInvocation/invoked":  "Listed metrics",
		},
	}
}

func (t *ListMetricsViews) CheckAccess(ctx context.Context) (bool, error) {
	s := GetSession(ctx)
	return s.Claims().Can(runtime.ReadObjects), nil
}

func (t *ListMetricsViews) Handler(ctx context.Context, args *ListMetricsViewsArgs) (*ListMetricsViewsResult, error) {
	session := GetSession(ctx)

	ctrl, err := t.Runtime.Controller(ctx, session.InstanceID())
	if err != nil {
		return nil, err
	}

	rs, err := ctrl.List(ctx, runtime.ResourceKindMetricsView, "", false)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(rs, func(a, b *runtimev1.Resource) int {
		an := a.Meta.Name
		bn := b.Meta.Name
		if an.Kind < bn.Kind {
			return -1
		}
		if an.Kind > bn.Kind {
			return 1
		}
		return strings.Compare(an.Name, bn.Name)
	})

	i := 0
	for i < len(rs) {
		r := rs[i]
		r, access, err := t.Runtime.ApplySecurityPolicy(ctx, session.InstanceID(), session.Claims(), r)
		if err != nil {
			return nil, err
		}
		if !access {
			// Remove from the slice
			rs[i] = rs[len(rs)-1]
			rs[len(rs)-1] = nil
			rs = rs[:len(rs)-1]
			continue
		}
		rs[i] = r
		i++
	}

	res := make(map[string]any)

	// Find instance-wide AI context and add it to the response.
	// NOTE: These arguably belong in the top-level instructions or other metadata, but that doesn't currently support dynamic values.
	instance, err := t.Runtime.Instance(ctx, session.InstanceID())
	if err != nil {
		return nil, fmt.Errorf("failed to get instance %q: %w", session.InstanceID(), err)
	}
	if instance.AIInstructions != "" {
		res["ai_instructions"] = instance.AIInstructions
	}

	var metricsViews []map[string]any
	for _, r := range rs {
		mv := r.GetMetricsView()
		if mv == nil || mv.State.ValidSpec == nil {
			continue
		}

		mvName := r.Meta.Name.Name
		entry := map[string]any{
			"name":         mvName,
			"display_name": mv.State.ValidSpec.DisplayName,
			"description":  mv.State.ValidSpec.Description,
		}
		labelMap := chineseLabelsForMV(ctx, t.Runtime, session.InstanceID(), r.Meta.Name.Name, mv.State.ValidSpec)
		if len(labelMap) > 0 {
			entry["chinese_labels"] = labelMap
		}
		metricsViews = append(metricsViews, entry)
	}
	res["metrics_views"] = metricsViews

	return &ListMetricsViewsResult{
		MetricsViews: metricsViews,
	}, nil
}

// chineseLabelsForMV reads the raw YAML of a metrics view from the repo and
// extracts {field_name -> Chinese_label} maps for dimensions and measures.
// Mirrors the same pattern used in metrics_view_get.go to keep the AI context bilingual.
func chineseLabelsForMV(ctx context.Context, rt *runtime.Runtime, instanceID, mvName string, spec *runtimev1.MetricsViewSpec) map[string]string {
	labels := make(map[string]string)
	if spec == nil || mvName == "" {
		return labels
	}
	repo, release, err := rt.Repo(ctx, instanceID)
	if err != nil || repo == nil {
		return labels
	}
	defer release()
	yamlBytes, err := repo.Get(ctx, "/metrics_views/"+mvName+".yaml")
	if err != nil {
		return labels
	}
	var mv parser.MetricsViewYAML
	if yaml.Unmarshal([]byte(yamlBytes), &mv) == nil {
		for _, d := range mv.Dimensions {
			if d.LabelCn != "" {
				labels[d.Name] = d.LabelCn
			}
		}
		for _, m := range mv.Measures {
			if m.LabelCn != "" {
				labels[m.Name] = m.LabelCn
			}
		}
	}
	return labels
}
