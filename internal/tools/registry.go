package tools

import (
	"fmt"

	"github.com/dundeezhang/agent-sh/internal/provider"
)

const maxOutputSize = 100 * 1024 // 100KB

// ToolResult is the result of executing a tool.
type ToolResult struct {
	Content string
	IsError bool
}

// ToolFunc is the function signature for tool executors.
type ToolFunc func(input map[string]interface{}) ToolResult

// Registry holds all available tools.
type Registry struct {
	tools    map[string]ToolFunc
	schemas  map[string]provider.Tool
	ordered  []string // preserve insertion order for deterministic output
}

// NewRegistry creates a registry with all built-in tools.
func NewRegistry() *Registry {
	r := &Registry{
		tools:   make(map[string]ToolFunc),
		schemas: make(map[string]provider.Tool),
	}
	r.registerBash()
	r.registerReadFile()
	r.registerWriteFile()
	r.registerEditFile()
	r.registerSearch()
	r.registerGlob()
	return r
}

func (r *Registry) register(name string, schema provider.Tool, fn ToolFunc) {
	r.tools[name] = fn
	r.schemas[name] = schema
	r.ordered = append(r.ordered, name)
}

// Execute runs a tool by name with the given input.
func (r *Registry) Execute(name string, input map[string]interface{}) ToolResult {
	fn, ok := r.tools[name]
	if !ok {
		return ToolResult{Content: fmt.Sprintf("unknown tool: %s", name), IsError: true}
	}
	result := fn(input)
	// Truncate output if too large
	if len(result.Content) > maxOutputSize {
		result.Content = result.Content[:maxOutputSize] + "\n... [output truncated]"
	}
	return result
}

// Tools returns the tool definitions for the provider.
func (r *Registry) Tools() []provider.Tool {
	tools := make([]provider.Tool, 0, len(r.ordered))
	for _, name := range r.ordered {
		tools = append(tools, r.schemas[name])
	}
	return tools
}
