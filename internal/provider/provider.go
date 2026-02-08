package provider

import "context"

// Provider is the interface for LLM providers.
type Provider interface {
	// Stream sends messages to the LLM and streams the response.
	Stream(ctx context.Context, params StreamParams) (*Response, error)
}

// StreamParams configures a streaming request.
type StreamParams struct {
	Model       string
	System      string
	Messages    []Message
	Tools       []Tool
	MaxTokens   int
	OnTextDelta func(text string) // called for each text chunk
	OnToolStart func(name string) // called when a tool call starts
}

// Message represents a conversation message.
type Message struct {
	Role    string         `json:"role"` // "user" or "assistant"
	Content []ContentBlock `json:"content"`
}

// ContentBlock is a typed block within a message.
type ContentBlock struct {
	Type       string      `json:"type"` // "text", "tool_use", "tool_result"
	Text       string      `json:"text,omitempty"`
	ToolUse    *ToolUse    `json:"tool_use,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// ToolUse represents a tool invocation by the model.
type ToolUse struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// Tool describes a tool available to the model.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// Response is the complete response from a streaming call.
type Response struct {
	Content    []ContentBlock
	StopReason string // "end_turn" or "tool_use"
	Usage      Usage
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int
	OutputTokens int
}
