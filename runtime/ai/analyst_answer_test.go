package ai

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseAnalystAnswer covers the robust parser used to turn the LLM's
// free-form final output into a structured AnalystAnswer. It runs without an
// LLM, exercising pure-JSON, fenced-JSON, JSON-embedded-in-prose, and
// plain-text fallback cases.
func TestParseAnalystAnswer(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		wantBody string
		wantSum  bool
	}{
		{
			name:     "pure json",
			raw:      `{"summary":"s","body":"hello **world**","insights":["a"],"follow_ups":["q?"]}`,
			wantBody: "hello **world**",
			wantSum:  true,
		},
		{
			name:     "fenced json",
			raw:      "```json\n{\"summary\":\"s\",\"body\":\"hi\"}\n```",
			wantBody: "hi",
			wantSum:  true,
		},
		{
			name:     "json embedded in prose",
			raw:      "Sure! Here is the result:\n{\"summary\":\"s\",\"body\":\"the body text\"}\nHope that helps.",
			wantBody: "the body text",
			wantSum:  true,
		},
		{
			name:     "plain text fallback",
			raw:      "The revenue increased by 25% last quarter.",
			wantBody: "The revenue increased by 25% last quarter.",
			wantSum:  false,
		},
		{
			// LLM forgot to escape inner double quotes inside a string value
			// (real-world case: 是绝对的"黄金客群"). The repair candidate must
			// recover the structured answer instead of leaking raw JSON.
			name:     "unescaped inner quotes repaired",
			raw:      `{"summary":"s","body":"b","insights":["是绝对的"黄金客群"、价值极高"],"follow_ups":["q?"]}`,
			wantBody: "b",
			wantSum:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ans := parseAnalystAnswer(tc.raw)
			require.Equal(t, tc.wantBody, ans.Body)
			if tc.wantSum {
				require.NotEmpty(t, ans.Summary)
			}
		})
	}
}

// TestAnalystAnswerMarshalRoundTrip verifies that a structured answer (including
// an injected chart reference) serializes back into a JSON string that the
// frontend can decode and surface via the `body` field.
func TestAnalystAnswerMarshalRoundTrip(t *testing.T) {
	ans := &AnalystAnswer{
		Summary:  "Revenue grew 25% QoQ.",
		Body:     "Based on the data analysis, here are the key insights:\n\n1. ## Revenue up 25%\n   Driven by the APAC region.",
		Insights: []string{"APAC led growth", "Churn flat"},
		FollowUps: []string{"Which APAC country grew most?"},
		Charts: []AnalystChartRef{
			{ChartType: "line_chart", Title: "Revenue trend", Spec: map[string]any{"metrics_view": "orders"}},
		},
	}
	b, err := json.Marshal(ans)
	require.NoError(t, err)

	// Frontend-style decode: RouterAgentResult.response holds this JSON string,
	// and the UI extracts `body`.
	var decoded struct {
		Body string `json:"body"`
	}
	require.NoError(t, json.Unmarshal(b, &decoded))
	require.Contains(t, decoded.Body, "Revenue up 25%")

	// Charts must survive the round trip for traceability.
	var full AnalystAnswer
	require.NoError(t, json.Unmarshal(b, &full))
	require.Len(t, full.Charts, 1)
	require.Equal(t, "line_chart", full.Charts[0].ChartType)
}
