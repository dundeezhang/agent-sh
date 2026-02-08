package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ToolCallSummary is a compact record of a single tool invocation.
type ToolCallSummary struct {
	Tool    string `json:"tool"`
	Input   string `json:"input"`
	IsError bool   `json:"is_error,omitempty"`
}

// Context holds the compact summary of one interaction.
type Context struct {
	Timestamp time.Time         `json:"timestamp"`
	CWD       string            `json:"cwd"`
	Query     string            `json:"query"`
	ToolCalls []ToolCallSummary `json:"tool_calls"`
	Summary   string            `json:"summary"`
}

// CacheDir returns the base cache directory for agent-sh context.
func CacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("cache dir: %w", err)
	}
	return filepath.Join(base, "agent-sh", "context"), nil
}

// pathFor maps a working directory to its context.json path inside the cache.
func pathFor(baseDir, cwd string) string {
	// Strip leading slash so filepath.Join works cleanly.
	rel := strings.TrimPrefix(cwd, "/")
	return filepath.Join(baseDir, rel, "context.json")
}

// Read loads the previous interaction context for cwd.
// Returns nil (no error) if no context exists.
func Read(cwd string) (*Context, error) {
	baseDir, err := CacheDir()
	if err != nil {
		return nil, err
	}
	p := pathFor(baseDir, cwd)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ctx Context
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, err
	}
	return &ctx, nil
}

// Write persists the interaction context for cwd.
func Write(cwd string, ctx *Context) error {
	baseDir, err := CacheDir()
	if err != nil {
		return err
	}
	p := pathFor(baseDir, cwd)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// Delete removes the context file for cwd. No error if it doesn't exist.
func Delete(cwd string) error {
	baseDir, err := CacheDir()
	if err != nil {
		return err
	}
	p := pathFor(baseDir, cwd)
	err = os.Remove(p)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Render formats a Context into a string suitable for the system prompt.
func Render(ctx *Context) string {
	if ctx == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Previous interaction")
	sb.WriteString(fmt.Sprintf(" (%s)\n", ctx.Timestamp.Format("15:04:05")))
	sb.WriteString(fmt.Sprintf("**Query:** %s\n", ctx.Query))

	if len(ctx.ToolCalls) > 0 {
		sb.WriteString("**Actions:** ")
		parts := make([]string, len(ctx.ToolCalls))
		for i, tc := range ctx.ToolCalls {
			parts[i] = fmt.Sprintf("%s(`%s`)", tc.Tool, tc.Input)
		}
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString("\n")
	}

	if ctx.Summary != "" {
		sb.WriteString(fmt.Sprintf("**Result:** %s\n", ctx.Summary))
	}
	return sb.String()
}
