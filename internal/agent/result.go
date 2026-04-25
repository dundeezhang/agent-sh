package agent

import (
	"strings"

	"github.com/dundeezhang/agent-sh/internal/provider"
	"github.com/dundeezhang/agent-sh/internal/render"
)

// ToolCallRecord captures a single tool invocation during an agent run.
type ToolCallRecord struct {
	Tool    string
	Input   string
	IsError bool
}

// InteractionResult is the compact output of a single agent Run().
type InteractionResult struct {
	Query     string
	ToolCalls []ToolCallRecord
	Summary   string
}

// extractText returns the concatenated text from response content blocks.
func extractText(blocks []provider.ContentBlock) string {
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}

// summarizeToolInput returns a short, descriptive string for a tool call,
// suitable for surfacing in memory or logs. Returns "" if the tool's
// display key is not registered or absent from the input.
func summarizeToolInput(name string, input map[string]interface{}) string {
	return render.Truncate(render.ToolDisplayValue(name, input), 100)
}

// buildResult constructs an InteractionResult from the collected data.
func buildResult(query string, toolCalls []ToolCallRecord, lastText string) *InteractionResult {
	return &InteractionResult{
		Query:     query,
		ToolCalls: toolCalls,
		Summary:   render.Truncate(lastText, 500),
	}
}
