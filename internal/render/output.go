package render

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dundeezhang/agent-sh/internal/provider"
)

// toolDisplayKey maps tool names to the input key holding their most
// descriptive argument. Single source of truth for the agent and renderer.
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

// ToolDisplayValue returns the descriptive argument value for a tool call,
// or "" if the tool has no display key registered or the value is missing.
func ToolDisplayValue(name string, input map[string]interface{}) string {
	key, ok := toolDisplayKey[name]
	if !ok {
		return ""
	}
	s, _ := input[key].(string)
	return s
}

// Truncate clips s to at most max runes, flattening newlines to spaces and
// appending "..." when the input was truncated.
func Truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max]) + "..."
	}
	return s
}

// Renderer handles styled terminal output for the agent. All output is
// directed at the writer it was constructed with — reused by the spinner and
// streaming markdown writer so a single Renderer fully owns its sink.
type Renderer struct {
	out     io.Writer
	spinner *Spinner
}

// NewRenderer creates a Renderer that writes to stdout.
func NewRenderer() *Renderer {
	return newRendererWith(os.Stdout)
}

func newRendererWith(out io.Writer) *Renderer {
	return &Renderer{
		out:     out,
		spinner: NewSpinner(out),
	}
}

// ToolCall displays a tool invocation with its primary argument.
func (r *Renderer) ToolCall(name string, input map[string]interface{}) {
	fmt.Fprintf(r.out, "\n\033[1;33m⚡ %s\033[0m", name)
	if val := ToolDisplayValue(name, input); val != "" {
		fmt.Fprintf(r.out, " \033[2m%s\033[0m", Truncate(val, 120))
	}
	fmt.Fprintln(r.out)
}

// ToolResult displays the result of a tool execution.
func (r *Renderer) ToolResult(name string, content string, isError bool) {
	if isError {
		fmt.Fprintf(r.out, "\033[1;31m✗ %s error:\033[0m %s\n", name, Truncate(content, 200))
		return
	}

	const maxLines = 10
	lines := strings.Split(content, "\n")
	if len(lines) > maxLines {
		preview := strings.Join(lines[:maxLines], "\n")
		fmt.Fprintf(r.out, "\033[2m%s\n... (%d more lines)\033[0m\n", preview, len(lines)-maxLines)
		return
	}
	if content != "" && content != "(no output)" {
		fmt.Fprintf(r.out, "\033[2m%s\033[0m\n", content)
	}
}

// Error displays an error message.
func (r *Renderer) Error(msg string) {
	fmt.Fprintf(r.out, "\033[1;31m%s\033[0m\n", msg)
}

// Usage displays token usage information.
func (r *Renderer) Usage(usage provider.Usage) {
	if usage.InputTokens > 0 || usage.OutputTokens > 0 {
		fmt.Fprintf(r.out, "\033[2m[tokens: %d in, %d out]\033[0m\n", usage.InputTokens, usage.OutputTokens)
	}
}

// Newline writes a single line break — used to terminate the streamed
// assistant message before printing the usage line.
func (r *Renderer) Newline() {
	fmt.Fprintln(r.out)
}

// StartSpinner starts the loading spinner with a message.
func (r *Renderer) StartSpinner(msg string) { r.spinner.Start(msg) }

// StopSpinner stops the loading spinner.
func (r *Renderer) StopSpinner() { r.spinner.Stop() }

// Stream returns a fresh writer for one LLM streaming turn. The returned
// writer auto-stops the spinner on first text or tool event, renders
// markdown formatting, and surfaces I/O errors via Flush.
//
// A new StreamWriter is required per turn: the underlying MarkdownWriter
// tracks fenced-code-block state, so reusing one across turns would let
// stale state corrupt rendering of the next response.
func (r *Renderer) Stream() *StreamWriter {
	return &StreamWriter{
		spinner: r.spinner,
		mdw:     NewMarkdownWriter(r.out),
	}
}

// StreamWriter ties together spinner control and markdown rendering for a
// single LLM streaming turn.
type StreamWriter struct {
	spinner *Spinner
	mdw     *MarkdownWriter
	err     error
}

// Text writes a streamed text chunk, stopping the spinner first.
func (s *StreamWriter) Text(text string) {
	s.spinner.Stop()
	if s.err != nil {
		return
	}
	if _, err := io.WriteString(s.mdw, text); err != nil {
		s.err = err
	}
}

// ToolStart signals that a tool-use block has begun streaming.
func (s *StreamWriter) ToolStart(_ string) {
	s.spinner.Stop()
}

// Flush emits any buffered partial line and returns the first I/O error
// encountered during the stream (write or flush).
func (s *StreamWriter) Flush() error {
	if err := s.mdw.Flush(); err != nil && s.err == nil {
		s.err = err
	}
	return s.err
}
