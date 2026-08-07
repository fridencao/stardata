package deepseek

import (
	"testing"

	aiv1 "github.com/fridencao/stardata/proto/gen/stardata/ai/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func textBlock(s string) *aiv1.ContentBlock {
	return &aiv1.ContentBlock{BlockType: &aiv1.ContentBlock_Text{Text: s}}
}

func toolCallBlock(t *testing.T, name string, input map[string]any) *aiv1.ContentBlock {
	t.Helper()
	var in *structpb.Struct
	if input != nil {
		s, err := structpb.NewStruct(input)
		require.NoError(t, err)
		in = s
	}
	return &aiv1.ContentBlock{
		BlockType: &aiv1.ContentBlock_ToolCall{
			ToolCall: &aiv1.ToolCall{Id: "call_1", Name: name, Input: in},
		},
	}
}

func blockKinds(msg *aiv1.CompletionMessage) []string {
	var kinds []string
	for _, b := range msg.Content {
		switch b.BlockType.(type) {
		case *aiv1.ContentBlock_Text:
			kinds = append(kinds, "text")
		case *aiv1.ContentBlock_ToolCall:
			kinds = append(kinds, "tool_call")
		}
	}
	return kinds
}

// TestNormalizePhantomToolCallBecomesText covers the DeepSeek behaviour that broke the
// router agent: for a structured-output request with no tools declared, DeepSeek answers
// with a tool call named after the completion (e.g. "Agent choice") instead of JSON text.
// The framework would then try to execute that non-existent tool.
func TestNormalizePhantomToolCallBecomesText(t *testing.T) {
	msg := &aiv1.CompletionMessage{
		Role:    "assistant",
		Content: []*aiv1.ContentBlock{toolCallBlock(t, "Agent choice", map[string]any{"agent": "analyst_agent"})},
	}

	got := normalizePhantomToolCalls(msg)

	require.Equal(t, []string{"text"}, blockKinds(got))
	require.JSONEq(t, `{"agent":"analyst_agent"}`, got.Content[0].GetText())
}

// TestNormalizePhantomToolCallDroppedWhenTextPresent: if the model already produced the
// answer as text, the phantom call carries no extra information and is discarded.
func TestNormalizePhantomToolCallDroppedWhenTextPresent(t *testing.T) {
	msg := &aiv1.CompletionMessage{
		Role: "assistant",
		Content: []*aiv1.ContentBlock{
			textBlock(`{"agent":"analyst_agent"}`),
			toolCallBlock(t, "Agent choice", map[string]any{"agent": "developer_agent"}),
		},
	}

	got := normalizePhantomToolCalls(msg)

	require.Equal(t, []string{"text"}, blockKinds(got))
	require.JSONEq(t, `{"agent":"analyst_agent"}`, got.Content[0].GetText())
}

// TestNormalizePhantomToolCallWithoutInputIsDropped guards the nil-input case.
func TestNormalizePhantomToolCallWithoutInputIsDropped(t *testing.T) {
	msg := &aiv1.CompletionMessage{
		Role:    "assistant",
		Content: []*aiv1.ContentBlock{toolCallBlock(t, "Agent choice", nil)},
	}

	got := normalizePhantomToolCalls(msg)

	require.Empty(t, got.Content)
}

// TestNormalizeLeavesTextOnlyMessagesAlone is the common path: a plain text answer is untouched.
func TestNormalizeLeavesTextOnlyMessagesAlone(t *testing.T) {
	msg := &aiv1.CompletionMessage{
		Role:    "assistant",
		Content: []*aiv1.ContentBlock{textBlock("United States")},
	}

	got := normalizePhantomToolCalls(msg)

	require.Equal(t, []string{"text"}, blockKinds(got))
	require.Equal(t, "United States", got.Content[0].GetText())
}
