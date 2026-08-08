package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	aiv1 "github.com/fridencao/stardata/proto/gen/stardata/ai/v1"
	runtimev1 "github.com/fridencao/stardata/proto/gen/stardata/runtime/v1"
	"github.com/fridencao/stardata/runtime"
	"github.com/fridencao/stardata/runtime/ai/instructions"
	"github.com/fridencao/stardata/runtime/metricsview"
)

const AnalystAgentName = "analyst_agent"

type AnalystAgent struct {
	Runtime *runtime.Runtime
}

var _ Tool[*AnalystAgentArgs, *AnalystAgentResult] = (*AnalystAgent)(nil)

type AnalystAgentArgs struct {
	Prompt     string   `json:"prompt"`
	Explore    string   `json:"explore,omitempty" yaml:"explore" jsonschema:"Optional explore dashboard name. If provided, the exploration will be limited to this dashboard."`
	Dimensions []string `json:"dimensions,omitempty" yaml:"dimensions" jsonschema:"Optional list of dimensions for queries. If provided, the queries will be limited to these dimensions."`
	Measures   []string `json:"measures,omitempty" yaml:"measures" jsonschema:"Optional list of measures for queries. If provided, the queries will be limited to these measures."`

	Canvas              string                             `json:"canvas,omitempty" yaml:"canvas" jsonschema:"Optional canvas name. If provided, the exploration will be limited to this canvas."`
	CanvasComponent     string                             `json:"canvas_component,omitempty" yaml:"canvas_component" jsonschema:"Optional canvas component name. If provided, the exploration will be limited to this canvas component."`
	WherePerMetricsView map[string]*metricsview.Expression `json:"where_per_metrics_view,omitempty" yaml:"where_per_metrics_view" jsonschema:"Optional filter for queries per metrics view. If provided, this filter will be applied to queries for each metrics view."`

	Where               *metricsview.Expression `json:"where,omitempty" yaml:"where" jsonschema:"Optional filter for queries. If provided, this filter will be applied to all queries."`
	TimeStart           time.Time               `json:"time_start,omitempty" yaml:"time_start" jsonschema:"Optional start time for queries. time_end must be provided if time_start is provided."`
	TimeEnd             time.Time               `json:"time_end,omitempty" yaml:"time_end" jsonschema:"Optional end time for queries. time_start must be provided if time_end is provided."`
	ComparisonTimeStart time.Time               `json:"comparison_time_start" yaml:"comparison_time_start" jsonschema:"Optional comparison period start time."`
	ComparisonTimeEnd   time.Time               `json:"comparison_time_end" yaml:"comparison_time_end" jsonschema:"Optional comparison period end time."`
	DisableCharts       bool                    `json:"disable_charts" yaml:"disable_charts" jsonschema:"Flag indicating whether to disable chart creation in the analysis."`
	IsReport            bool                    `json:"is_report" yaml:"is_report" jsonschema:"Flag indicating this is an automated report."`
}

func (a *AnalystAgentArgs) ToLLM() *aiv1.ContentBlock {
	return &aiv1.ContentBlock{
		BlockType: &aiv1.ContentBlock_Text{
			Text: a.Prompt,
		},
	}
}

type AnalystAgentResult struct {
	Response string `json:"response"`
}

func (r *AnalystAgentResult) ToLLM() *aiv1.ContentBlock {
	return &aiv1.ContentBlock{
		BlockType: &aiv1.ContentBlock_Text{
			Text: r.Response,
		},
	}
}

// AnalystAnswer is the structured form of the analyst agent's final answer.
// It is serialized to JSON and stored in AnalystAgentResult.Response. The field
// stays a string (backward compatible with the router, chat server and SSE
// pipeline); only its content changes from free text to a JSON document.
type AnalystAnswer struct {
	Summary   string             `json:"summary,omitempty"`
	Body      string             `json:"body"`
	Insights  []string          `json:"insights,omitempty"`
	FollowUps []string          `json:"follow_ups,omitempty"`
	Charts    []AnalystChartRef `json:"charts,omitempty"`
}

// AnalystChartRef references a chart created during the analysis (via the
// create_chart tool). It is injected programmatically (not by the LLM) so the
// answer can be traced to the exact visualizations that were produced.
type AnalystChartRef struct {
	ChartType string         `json:"chart_type"`
	Title     string         `json:"title,omitempty"`
	Spec      map[string]any `json:"spec,omitempty"`
}

// parseAnalystAnswer robustly converts the LLM's raw output into a structured
// AnalystAnswer. It tolerates: pure JSON, ```json fenced JSON, JSON embedded
// in prose, and plain text (which falls back to a single body field).
func parseAnalystAnswer(raw string) *AnalystAnswer {
	raw = strings.TrimSpace(raw)

	candidates := []string{raw}

	// Strip a leading/trailing ```json (or ```) fence if present.
	if strings.HasPrefix(raw, "```") {
		rest := raw[3:]
		rest = strings.TrimPrefix(rest, "json")
		rest = strings.TrimSpace(rest)
		if end := strings.LastIndex(rest, "```"); end >= 0 {
			rest = rest[:end]
		}
		candidates = append(candidates, strings.TrimSpace(rest))
	}

	// Fall back to the substring between the first '{' and the last '}'.
	if i := strings.Index(raw, "{"); i >= 0 {
		if j := strings.LastIndex(raw, "}"); j > i {
			candidates = append(candidates, raw[i:j+1])
		}
	}

	// Last resort: repair unescaped inner double quotes (LLMs occasionally emit
	// e.g. "是绝对的"黄金客群"" inside a JSON string), which otherwise makes the
	// whole answer unparseable and leaks raw JSON to the user.
	for _, c := range candidates {
		if r := repairJSONStringQuotes(c); r != c {
			candidates = append(candidates, r)
		}
	}

	for _, c := range candidates {
		if c == "" {
			continue
		}
		var ans AnalystAnswer
		if err := json.Unmarshal([]byte(c), &ans); err == nil {
			if ans.Body != "" || ans.Summary != "" || len(ans.Insights) > 0 || len(ans.FollowUps) > 0 {
				return &ans
			}
		}
	}

	// Could not parse a structured answer; treat the raw text as the narrative body.
	return &AnalystAnswer{Body: raw}
}

// repairJSONStringQuotes escapes unescaped double quotes that appear *inside*
// JSON string values. It walks the document tracking string boundaries: a
// quote inside a string only terminates it if the next non-space character is
// structural (one of `,:}]`); otherwise it is an inner quote the LLM forgot to
// escape and gets rewritten as `\"`. Best-effort — the result is only used as
// an extra parse candidate.
func repairJSONStringQuotes(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inString := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inString && c == '\\' && i+1 < len(s) {
			// Preserve existing escape sequences verbatim.
			b.WriteByte(c)
			b.WriteByte(s[i+1])
			i++
			continue
		}
		if c != '"' {
			b.WriteByte(c)
			continue
		}
		if !inString {
			inString = true
			b.WriteByte(c)
			continue
		}
		// Quote while inside a string: closing quote or unescaped inner quote?
		j := i + 1
		for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
			j++
		}
		if j >= len(s) || s[j] == ',' || s[j] == ':' || s[j] == '}' || s[j] == ']' {
			inString = false
			b.WriteByte(c)
		} else {
			b.WriteString(`\"`)
		}
	}
	return b.String()
}

// collectCharts gathers the charts created during the analyst loop from the
// session messages (the create_chart tool results) and returns them as references.
func (t *AnalystAgent) collectCharts(ctx context.Context) []AnalystChartRef {
	s := GetSession(ctx)
	msgs := s.MessagesWithChildren(FilterByTool(CreateChartName), FilterByType(MessageTypeResult))
	refs := make([]AnalystChartRef, 0, len(msgs))
	for _, m := range msgs {
		res, err := s.UnmarshalMessageContent(m)
		if err != nil {
			continue
		}
		ccr, ok := res.(*CreateChartResult)
		if !ok || ccr == nil {
			continue
		}
		refs = append(refs, AnalystChartRef{
			ChartType: ccr.ChartType,
			Title:     chartTitleFromSpec(ccr.Spec),
			Spec:      ccr.Spec,
		})
	}
	return refs
}

// chartTitleFromSpec extracts a human-readable title from a chart spec, if present.
func chartTitleFromSpec(spec map[string]any) string {
	if spec == nil {
		return ""
	}
	if title, ok := spec["title"].(string); ok {
		return title
	}
	return ""
}

func (t *AnalystAgent) Spec() *mcp.Tool {
	// It can't automatically infer schemas that use the metricsview.Expression type, so we manually do that here.
	inputSchema, err := jsonschema.For[*AnalystAgentArgs](&jsonschema.ForOptions{
		TypeSchemas: metricsview.TypeSchemas(),
	})
	if err != nil {
		panic(fmt.Errorf("failed to infer input schema: %w", err))
	}

	return &mcp.Tool{
		Name:        AnalystAgentName,
		Title:       "Analyst Agent",
		Description: "Agent that assists with data analysis tasks.",
		InputSchema: inputSchema,
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Analyzing...",
			"openai/toolInvocation/invoked":  "Analysis completed",
		},
	}
}

func (t *AnalystAgent) CheckAccess(ctx context.Context) (bool, error) {
	// Must be allowed to use AI and query metrics
	s := GetSession(ctx)
	if !s.Claims().Can(runtime.UseAI) || !s.Claims().Can(runtime.ReadMetrics) {
		return false, nil
	}

	// Only allow for rill user agents since it's not useful in MCP contexts.
	if !strings.HasPrefix(s.CatalogSession().UserAgent, "stardata") {
		return false, nil
	}

	return true, nil
}

func (t *AnalystAgent) Handler(ctx context.Context, args *AnalystAgentArgs) (*AnalystAgentResult, error) {
	s := GetSession(ctx)

	// Determine if it's the first invocation of the agent in this session.
	first := len(s.Messages(FilterByType(MessageTypeCall), FilterByTool(AnalystAgentName))) == 1

	// If a specific dashboard is being explored, we pre-invoke some relevant tool calls for that dashboard.
	// TODO: This uses `first`, but that may not be safe if the user has navigated to another dashboard. We probably need some more sophisticated de-duplication here.
	var metricsViewNames []string
	if first {
		if args.Explore != "" {
			_, metricsView, err := t.getValidExploreAndMetricsView(ctx, args.Explore)
			if err != nil {
				return nil, err
			}
			metricsViewNames = append(metricsViewNames, metricsView.Meta.Name.Name)
		} else if args.Canvas != "" {
			// Pre-invoke the get_canvas tool to get the canvas definition.
			_, err := s.CallTool(ctx, RoleAssistant, GetCanvasName, nil, &GetCanvasArgs{
				Canvas: args.Canvas,
			})
			if err != nil && errors.Is(err, ctx.Err()) { // Don't exit on non-context errors
				return nil, err
			}

			_, metricsViews, err := t.getValidCanvasAndMetricsViews(ctx, args.Canvas)
			if err != nil {
				return nil, err
			}

			for _, res := range metricsViews {
				metricsViewNames = append(metricsViewNames, res.Meta.Name.Name)
			}
		}

		// Pre-invoke the query_metrics_view tool for each metrics view tied to the explore/canvas.
		for _, mvName := range metricsViewNames {
			_, err := s.CallTool(ctx, RoleAssistant, QueryMetricsViewSummaryName, nil, &QueryMetricsViewSummaryArgs{
				MetricsView: mvName,
			})
			if err != nil && errors.Is(err, ctx.Err()) { // Don't exit on non-context errors
				return nil, err
			}

			_, err = s.CallTool(ctx, RoleAssistant, GetMetricsViewName, nil, &GetMetricsViewArgs{
				MetricsView: mvName,
			})
			if err != nil && errors.Is(err, ctx.Err()) { // Don't exit on non-context errors
				return nil, err
			}
		}
	}

	// If no specific dashboard is being explored, we pre-invoke the list_metrics_views tool.
	if first && len(metricsViewNames) == 0 {
		_, err := s.CallTool(ctx, RoleAssistant, ListMetricsViewsName, nil, &ListMetricsViewsArgs{})
		if err != nil && errors.Is(err, ctx.Err()) { // Don't exit on non-context errors
			return nil, err
		}
	}

	// Determine tools that can be used
	tools := []string{}
	if args.Explore == "" {
		tools = append(tools, ListMetricsViewsName, GetMetricsViewName, GetCanvasName)
	}
	tools = append(tools, QueryMetricsViewSummaryName, QueryMetricsViewName)
	if !args.DisableCharts {
		tools = append(tools, CreateChartName)
	}

	// Build completion messages
	systemPrompt, err := t.systemPrompt(ctx)
	if err != nil {
		return nil, err
	}
	userPrompt, err := t.userPrompt(ctx, metricsViewNames, args)
	if err != nil {
		return nil, err
	}
	// 1. System prompt
	messages := []*aiv1.CompletionMessage{NewTextCompletionMessage(RoleSystem, systemPrompt)}
	// 2. Previous analyst calls with their tool calls
	notCurrentCall := func(m *Message) bool { return m.ID != s.ParentID }
	messages = append(messages, s.NewCompletionMessages(s.MessagesWithChildren(FilterByType(MessageTypeCall), FilterByTool(AnalystAgentName), notCurrentCall))...)
	// 3. User prompt
	messages = append(messages, NewTextCompletionMessage(RoleUser, userPrompt))
	// 4. Seeded tool calls for the current iteration
	messages = append(messages, s.NewCompletionMessages(s.MessagesWithResults(FilterByParent(s.ParentID)))...)

	// Run an LLM tool call loop
	var response string
	err = s.Complete(ctx, "Analyst loop", &response, &CompleteOptions{
		Messages:      messages,
		Tools:         tools,
		MaxIterations: 20,
		UnwrapCall:    true,
	})
	if err != nil {
		return nil, err
	}

	// Parse the free-form LLM answer into a structured form, then inject the
	// charts that were actually created during the loop (for traceability).
	answer := parseAnalystAnswer(response)
	if charts := t.collectCharts(ctx); len(charts) > 0 {
		answer.Charts = charts
	}
	answerJSON, err := json.Marshal(answer)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal structured analyst answer: %w", err)
	}

	return &AnalystAgentResult{Response: string(answerJSON)}, nil
}

func (t *AnalystAgent) systemPrompt(ctx context.Context) (string, error) {
	locale := "zh"
	if s := GetSession(ctx); s != nil {
		locale = s.Locale()
	}
	instr, err := instructions.LoadLocalized("analysis.md", locale, instructions.Options{})
	if err != nil {
		return "", fmt.Errorf("failed to load analyst agent system prompt: %w", err)
	}

	return instr.Body, nil
}

func (t *AnalystAgent) userPrompt(ctx context.Context, metricsViewNames []string, args *AnalystAgentArgs) (string, error) {
	// Prepare template data.
	// NOTE: All the template properties are optional and may be empty.
	session := GetSession(ctx)

	instance, err := t.Runtime.Instance(ctx, session.InstanceID())
	if err != nil {
		return "", fmt.Errorf("failed to get instance: %w", err)
	}
	instanceCfg, err := instance.Config()
	if err != nil {
		return "", fmt.Errorf("failed to get instance config: %w", err)
	}

	metricsViewsQuoted := make([]string, len(metricsViewNames))
	for i, mv := range metricsViewNames {
		metricsViewsQuoted[i] = fmt.Sprintf("`%s`", mv)
	}

	dimensionsQuoted := make([]string, len(args.Dimensions))
	for i, dim := range args.Dimensions {
		dimensionsQuoted[i] = fmt.Sprintf("`%s`", dim)
	}

	measuresQuoted := make([]string, len(args.Measures))
	for i, measure := range args.Measures {
		measuresQuoted[i] = fmt.Sprintf("`%s`", measure)
	}

	data := map[string]any{
		"prompt":           args.Prompt,
		"ai_instructions":  session.ProjectInstructions(),
		"is_prompt":        args.Prompt != "",
		"metrics_views":    strings.Join(metricsViewsQuoted, ", "),
		"explore":          args.Explore,
		"canvas":           args.Canvas,
		"canvas_component": args.CanvasComponent,
		"dimensions":       strings.Join(dimensionsQuoted, ", "),
		"measures":         strings.Join(measuresQuoted, ", "),
		"forked":           session.Forked(),
		"is_report":        args.IsReport,
		"now":              time.Now(),
		"max_query_limit":  instanceCfg.AIMaxQueryLimit,
	}

	if !args.TimeStart.IsZero() && !args.TimeEnd.IsZero() {
		data["time_start"] = args.TimeStart.Format(time.RFC3339)
		data["time_end"] = args.TimeEnd.Format(time.RFC3339)
	}

	if !args.ComparisonTimeStart.IsZero() && !args.ComparisonTimeEnd.IsZero() {
		data["comparison_start"] = args.ComparisonTimeStart.Format(time.RFC3339)
		data["comparison_end"] = args.ComparisonTimeEnd.Format(time.RFC3339)
	}

	if args.Where != nil {
		data["where"], err = metricsview.ExpressionToSQL(args.Where)
		if err != nil {
			return "", err
		}
	}

	if args.WherePerMetricsView != nil {
		wherePerMetricsView := map[string]string{}
		for metricsViewName, whereExpr := range args.WherePerMetricsView {
			wherePerMetricsView[metricsViewName], err = metricsview.ExpressionToSQL(whereExpr)
			if err != nil {
				return "", err
			}
		}
		data["where_per_metrics_view"] = wherePerMetricsView
	}

	// Generate the user prompt.
	// It carries all the per-invocation context: the current date, dashboard/report context, applied query settings, forked-session caveats, and finally the user's actual prompt.
	return executeTemplate(`Today's date is {{ .now.Format "Monday, January 2, 2006" }} ({{ .now.Format "2006-01-02" }}).

If the user's question is in Chinese, follow the "语言与本地化要求 (Language & Localization)" section in the system prompt: translate Chinese temporal expressions (e.g. "上个月", "同比", "环比", "近30天", "本季度") into precise ISO 8601 time ranges / comparison_time_ranges, and write all user-facing text (insights, chart titles, axis labels) in Chinese.

{{ if .is_report }}
You are operating in an automated scheduled insight report mode where you will come up with insights on your own without additional user input.
{{ if .is_prompt }}The user has provided a custom prompt for this scheduled insight report. Tailor your analysis to address this prompt specifically.{{ end }}
{{ end }}

{{ if .explore }}
Your goal is to analyze the contents of the dashboard "{{ .explore }}", which is powered by the metrics view(s) {{ .metrics_views }}.
{{ if not .is_report }}The user is actively viewing this dashboard, and it's what they refer to if they use expressions like "this dashboard", "the current view", etc. {{ end }}
The metrics view's definition and time range of available data has been provided in your tool calls.

Here is an overview of the settings applied to the dashboard:
{{ if (and .time_start .time_end) }}Use time range: start={{.time_start}}, end={{.time_end}}{{ end }}
{{ if (and .comparison_start .comparison_end) }}Use comparison time range: start={{.comparison_start}}, end={{.comparison_end}}{{ end }}
{{ if .where }}Use where filters: "{{ .where }}"{{ end }}
{{ if .measures }}Use measures: {{ .measures }}{{ end }}
{{ if .dimensions }}Use dimensions: {{ .dimensions }}{{ end }}

You should:
1. Carefully study the metrics view definition to understand the measures and dimensions available for analysis.
2. Remember the time range of available data and use it to inform and filter your queries.
{{ else if .canvas }}
Your goal is to analyze the contents of the canvas "{{ .canvas }}", which is powered by the metrics view(s) {{ .metrics_views }}.
The user is actively viewing this dashboard, and it's what they refer to if they use expressions like "this dashboard", "the current view", etc.
The metrics views and canvas definitions have been provided in your tool calls.

Here is an overview of the settings the user has currently applied to the dashboard (Merge component's dimension_filters with "and"):
{{ if (and .time_start .time_end) }}Use time range: start={{.time_start}}, end={{.time_end}}{{ end }}
{{ if .where_per_metrics_view }}{{range $mv, $filter := .where_per_metrics_view}}Use where filters for metrics view "{{ $mv }}": "{{ $filter }}"
{{end}}{{ end }}

You should:
1. Carefully study the canvas and metrics view definition to understand the measures and dimensions available for analysis.
2. Remember the time range of available data and use it to inform and filter your queries.
{{ if .canvas_component }}
The user is looking at "{{ .canvas_component }}". Pay special attention to its definition and filters and use it to inform your analysis.
{{ end }}
{{ end }}

{{ if .forked }}
Important instructions regarding access permissions:
This conversation has been transferred and the new owner may have different access permissions.
If you start seeing access errors like "action not allowed", "resource not found" (for resources earlier available) etc., consider repeating metadata listings and lookups.
If you run into such issues, explicitly mention to the user that this may be due to conversation forking and that they may not have access to the data that the previous user had.
{{ end }}

{{ if and .is_report (not .is_prompt) }}
{{ if (and .comparison_start .comparison_end) }}
<comparison_analysis>
You are doing comparative analysis between two time periods in scheduled insight report mode, your analysis should:
1. Compare current period to the comparison period for all key measures
2. Identify which measures changed significantly (>10%)
3. For each significant change, identify the dimensional drivers
4. Highlight any ranking changes in top dimensions
5. Generate 3-5 key insights with supporting charts

Focus areas:
- **Overall changes**: Which measures changed the most between periods?
- **Drivers of change**: Which dimensions contributed most to increases/decreases?
- **Ranking shifts**: Did any top dimensions change rank significantly?
- **Anomalies**: Any unusual patterns unique to one period?
</comparison_analysis>
{{ else }}
<single_period_analysis>
You are doing single period analysis in scheduled insight report mode, your analysis should:
1. Show totals for the most impactful measures in the period
2. Identify interesting trends within the time range (use time series)
3. Find anomalies - unusual spikes, drops, or outliers
4. Highlight top performers and notable dimension values
5. Generate 3-5 key insights with supporting charts

Focus areas:
- **Totals**: What are the key numbers for this period?
- **Trends**: How did metrics change over the period? Any acceleration/deceleration?
- **Anomalies**: Are there any unusual data points that stand out?
- **Distribution**: Which dimensions dominate? Any concentration issues?
</single_period_analysis>
{{ end }}
{{ end }}

{{ if .is_report }}
When formatting your final analysis, begin with a one-line summary wrapped in <summary></summary> tags. Do not include citation links in the summary.
{{ end }}

The system allows a max row limit of {{ .max_query_limit }} per query.

{{ if .ai_instructions }}
The administrator has provided the following project-wide instructions, which may or may not be relevant to this task:
{{ .ai_instructions }}
{{ end }}

{{ if .is_prompt }}
The user's request:
{{ .prompt }}
{{ end }}
`, data)
}

func (t *AnalystAgent) getValidExploreAndMetricsView(ctx context.Context, exploreName string) (*runtimev1.Resource, *runtimev1.Resource, error) {
	session := GetSession(ctx)

	ctrl, err := t.Runtime.Controller(ctx, session.InstanceID())
	if err != nil {
		return nil, nil, err
	}

	r, err := ctrl.Get(ctx, &runtimev1.ResourceName{Kind: runtime.ResourceKindExplore, Name: exploreName}, false)
	if err != nil {
		return nil, nil, err
	}

	explore, access, err := t.Runtime.ApplySecurityPolicy(ctx, session.InstanceID(), session.Claims(), r)
	if err != nil {
		return nil, nil, err
	}
	if !access {
		return nil, nil, fmt.Errorf("explore %q not found", exploreName)
	}

	exploreSpec := explore.GetExplore().State.ValidSpec
	if exploreSpec == nil {
		return nil, nil, fmt.Errorf("explore %q is not valid", exploreName)
	}

	metricsView, err := ctrl.Get(ctx, &runtimev1.ResourceName{Kind: runtime.ResourceKindMetricsView, Name: exploreSpec.MetricsView}, false)
	if err != nil {
		return nil, nil, err
	}

	metricsView, access, err = t.Runtime.ApplySecurityPolicy(ctx, session.InstanceID(), session.Claims(), metricsView)
	if err != nil {
		return nil, nil, err
	}
	if !access {
		return nil, nil, fmt.Errorf("metrics view %q not found", exploreSpec.MetricsView)
	}

	metricsViewSpec := metricsView.GetMetricsView().State.ValidSpec
	if metricsViewSpec == nil {
		return nil, nil, fmt.Errorf("metrics view %q is not valid", exploreSpec.MetricsView)
	}

	return explore, metricsView, nil
}

func (t *AnalystAgent) getValidCanvasAndMetricsViews(ctx context.Context, canvasName string) (*runtimev1.Resource, map[string]*runtimev1.Resource, error) {
	session := GetSession(ctx)

	resolvedCanvas, err := t.Runtime.ResolveCanvas(ctx, session.InstanceID(), canvasName, session.Claims(), false)
	if err != nil {
		return nil, nil, err
	}

	if resolvedCanvas == nil || resolvedCanvas.Canvas == nil {
		return nil, nil, fmt.Errorf("canvas %q not found", canvasName)
	}

	metricsViews := map[string]*runtimev1.Resource{}
	for mv, res := range resolvedCanvas.ReferencedMetricsViews {
		metricsView := res.GetMetricsView()
		if metricsView == nil || metricsView.State.ValidSpec == nil {
			continue
		}
		metricsViews[mv] = res
	}

	return resolvedCanvas.Canvas, metricsViews, nil
}
