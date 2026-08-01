package ai

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestPromptToTitle(t *testing.T) {
	cases := []struct {
		name   string
		prompt string
		want   string
	}{
		{
			name:   "plain prompt",
			prompt: "What country has the highest revenue?",
			want:   "What country has the highest revenue?",
		},
		{
			name:   "metrics view reference only",
			prompt: `<chat-reference>type="metricsView" metricsView="sales"</chat-reference>`,
			want:   "@sales",
		},
		{
			name:   "metrics view reference with question",
			prompt: `<chat-reference>type="metricsView" metricsView="sales"</chat-reference> 分析销售额分布`,
			want:   "@sales 分析销售额分布",
		},
		{
			name:   "measure reference",
			prompt: `Explain <chat-reference>type="measure" metricsView="adbids" measure="impressions"</chat-reference>`,
			want:   "Explain @impressions",
		},
		{
			name:   "dimension reference",
			prompt: `<chat-reference>type="dimension" metricsView="adbids" dimension="publisher"</chat-reference> breakdown`,
			want:   "@publisher breakdown",
		},
		{
			name:   "empty prompt falls back",
			prompt: "",
			want:   "New Conversation",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, promptToTitle(c.prompt))
		})
	}
}

func TestPromptToTitleTruncatesRuneSafe(t *testing.T) {
	// 60 Chinese characters (each 3 bytes in UTF-8): must truncate on a rune
	// boundary so the result stays valid UTF-8.
	prompt := strings.Repeat("销", 60)

	title := promptToTitle(prompt)

	require.True(t, utf8.ValidString(title), "title must be valid UTF-8")
	require.Equal(t, 47, utf8.RuneCountInString(strings.TrimSuffix(title, "...")))
	require.True(t, strings.HasSuffix(title, "..."))
}
