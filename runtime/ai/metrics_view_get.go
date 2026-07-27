package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	runtimev1 "github.com/fridencao/stardata/proto/gen/stardata/runtime/v1"
	"github.com/fridencao/stardata/runtime"
	parser "github.com/fridencao/stardata/runtime/parser"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
)

const GetMetricsViewName = "get_metrics_view"

type GetMetricsView struct {
	Runtime *runtime.Runtime
}

var _ Tool[*GetMetricsViewArgs, *GetMetricsViewResult] = (*GetMetricsView)(nil)

type GetMetricsViewArgs struct {
	MetricsView string `json:"metrics_view" jsonschema:"Name of the metrics view"`
}

type GetMetricsViewResult struct {
	Spec map[string]any `json:"spec"`
}

func (t *GetMetricsView) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        GetMetricsViewName,
		Title:       "Get Metrics View",
		Description: "Get the specification for a given metrics view, including available measures and dimensions",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: boolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   boolPtr(false),
			ReadOnlyHint:    true,
		},
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Getting metrics definition...",
			"openai/toolInvocation/invoked":  "Found metrics definition",
		},
	}
}

func (t *GetMetricsView) CheckAccess(ctx context.Context) (bool, error) {
	s := GetSession(ctx)
	return s.Claims().Can(runtime.ReadObjects), nil
}

func (t *GetMetricsView) Handler(ctx context.Context, args *GetMetricsViewArgs) (*GetMetricsViewResult, error) {
	session := GetSession(ctx)

	ctrl, err := t.Runtime.Controller(ctx, session.InstanceID())
	if err != nil {
		return nil, err
	}

	r, err := ctrl.Get(ctx, &runtimev1.ResourceName{Kind: runtime.ResourceKindMetricsView, Name: args.MetricsView}, false)
	if err != nil {
		return nil, err
	}

	r, access, err := t.Runtime.ApplySecurityPolicy(ctx, session.InstanceID(), session.Claims(), r)
	if err != nil {
		return nil, err
	}
	if !access {
		return nil, fmt.Errorf("resource not found")
	}

	if r.GetMetricsView().State.ValidSpec == nil {
		return nil, fmt.Errorf("metrics view %q is invalid", args.MetricsView)
	}

	// StarData publish gate: block direct access to unpublished metrics views.
	if pub, gated := publishedMetricsViews(ctx, t.Runtime, session.InstanceID()); gated && !pub[args.MetricsView] {
		return nil, fmt.Errorf("metrics view %q is not published", args.MetricsView)
	}

	specJSON, err := protojson.Marshal(r.GetMetricsView().State.ValidSpec)
	if err != nil {
		return nil, err
	}
	var specMap map[string]any
	err = json.Unmarshal(specJSON, &specMap)
	if err != nil {
		return nil, err
	}

	// Inject Chinese field aliases (label_cn) into the spec the LLM sees, without
	// modifying the proto. The proto MetricsViewSpec has no label_cn field, and the
	// parsed spec drops it, so we re-read the metrics view's raw YAML from the repo.
	if repo, rel, rerr := t.Runtime.Repo(ctx, session.InstanceID()); rerr == nil {
		defer rel()
		if yamlBytes, gerr := repo.Get(ctx, "/metrics_views/"+args.MetricsView+".yaml"); gerr == nil {
			var mv parser.MetricsViewYAML
			if yaml.Unmarshal([]byte(yamlBytes), &mv) == nil {
				labels := map[string]string{}
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
				if len(labels) > 0 {
					specMap["chinese_labels"] = labels
				}
			}
		}
	}

	return &GetMetricsViewResult{
		Spec: specMap,
	}, nil
}
