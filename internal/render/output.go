package render

import (
	"fmt"
	"os"
	"strings"

	"github.com/dundeezhang/agent-sh/internal/provider"
)

// Renderer handles styled terminal output for the agent.
type Renderer struct {
	spinner *Spinner
}

// NewRenderer creates a new Renderer.
func NewRenderer() *Renderer {
	return &Renderer{
		spinner: NewSpinner(),
	}
}

// ToolCall displays a tool invocation with its arguments.
func (r *Renderer) ToolCall(name string, input map[string]interface{}) {
	fmt.Fprintf(os.Stdout, "\n\033[1;33m⚡ %s\033[0m", name)

	// Show a concise summary of the input.
	// Map tool name to the key holding the most descriptive argument.
	displayKeys := map[string]string{
		"bash":       "command",
		"read_file":  "path",
		"write_file": "path",
		"edit_file":  "path",
		"search":     "pattern",
		"glob":       "pattern",
		"web_search": "query",
		"web_fetch":  "url",
	}
	if key, ok := displayKeys[name]; ok {
		if val, ok := input[key].(string); ok {
			fmt.Fprintf(os.Stdout, " \033[2m%s\033[0m", truncate(val, 120))
		}
	}
	_, _ = fmt.Fprintln(os.Stdout)
}

// ToolResult displays the result of a tool execution.
func (r *Renderer) ToolResult(name string, content string, isError bool) {
	if isError {
		fmt.Fprintf(os.Stdout, "\033[1;31m✗ %s error:\033[0m %s\n", name, truncate(content, 200))
		return
	}

	// Show a truncated preview of the result
	lines := strings.Split(content, "\n")
	maxLines := 10
	if len(lines) > maxLines {
		preview := strings.Join(lines[:maxLines], "\n")
		fmt.Fprintf(os.Stdout, "\033[2m%s\n... (%d more lines)\033[0m\n", preview, len(lines)-maxLines)
	} else if content != "" && content != "(no output)" {
		fmt.Fprintf(os.Stdout, "\033[2m%s\033[0m\n", content)
	}
}

// Error displays an error message.
func (r *Renderer) Error(msg string) {
	fmt.Fprintf(os.Stdout, "\033[1;31m%s\033[0m\n", msg)
}

// Usage displays token usage information.
func (r *Renderer) Usage(usage provider.Usage) {
	if usage.InputTokens > 0 || usage.OutputTokens > 0 {
		fmt.Fprintf(os.Stdout, "\033[2m[tokens: %d in, %d out]\033[0m\n", usage.InputTokens, usage.OutputTokens)
	}
}

// StartSpinner starts the loading spinner with a message.
func (r *Renderer) StartSpinner(msg string) {
	r.spinner.Start(msg)
}

// StopSpinner stops the loading spinner.
func (r *Renderer) StopSpinner() {
	r.spinner.Stop()
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max]) + "..."
	}
	return s
}
