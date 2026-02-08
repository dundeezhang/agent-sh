package agent

import (
	"strings"

	"github.com/dundeezhang/agent-sh/internal/provider"
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

// extractText pulls the concatenated text from response content blocks.
func extractText(blocks []provider.ContentBlock) string {
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}

// toolDisplayKey maps tool names to the key holding their most descriptive argument.
var toolDisplayKey = map[string]string{
	"bash":       "command",
	"read_file":  "path",
	"write_file": "path",
	"edit_file":  "path",
	"search":     "pattern",
	"glob":       "pattern",
	"web_search": "query",
	"web_fetch":  "url",
}

// extractToolInput returns a short string describing the primary input arg.
func extractToolInput(name string, input map[string]interface{}) string {
	key := toolDisplayKey[name]
	if key == "" {
		key = "command"
	}
	v, ok := input[key]
	if !ok {
		// Fallback: grab first string value.
		for _, val := range input {
			if s, ok := val.(string); ok {
				return truncate(s, 100)
			}
		}
		return ""
	}
	if s, ok := v.(string); ok {
		return truncate(s, 100)
	}
	return ""
}

// buildResult constructs an InteractionResult from the collected data.
func buildResult(query string, toolCalls []ToolCallRecord, lastText string) *InteractionResult {
	return &InteractionResult{
		Query:     query,
		ToolCalls: toolCalls,
		Summary:   truncate(lastText, 500),
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
