package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Anthropic implements Provider using the Anthropic API.
type Anthropic struct {
	client *anthropic.Client
}

// NewAnthropic creates a new Anthropic provider.
// apiKey can be empty if ANTHROPIC_API_KEY env var is set.
func NewAnthropic(apiKey string) *Anthropic {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	client := anthropic.NewClient(opts...)
	return &Anthropic{client: &client}
}

func (a *Anthropic) resolveModel(model string) anthropic.Model {
	switch model {
	case "sonnet", "claude-sonnet":
		return anthropic.ModelClaudeSonnet4_5
	case "haiku", "claude-haiku":
		return anthropic.ModelClaudeHaiku4_5
	case "opus", "claude-opus":
		return anthropic.ModelClaudeOpus4_6
	default:
		return anthropic.Model(model)
	}
}

func (a *Anthropic) Stream(ctx context.Context, params StreamParams) (*Response, error) {
	model := a.resolveModel(params.Model)
	maxTokens := params.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}

	// Build messages
	messages := make([]anthropic.MessageParam, 0, len(params.Messages))
	for _, msg := range params.Messages {
		blocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.Content))
		for _, cb := range msg.Content {
			switch cb.Type {
			case "text":
				blocks = append(blocks, anthropic.NewTextBlock(cb.Text))
			case "tool_use":
				if cb.ToolUse != nil {
					inputJSON, err := json.Marshal(cb.ToolUse.Input)
					if err != nil {
						return nil, fmt.Errorf("anthropic: marshal tool input for %q: %w", cb.ToolUse.Name, err)
					}
					blocks = append(blocks, anthropic.ContentBlockParamUnion{
						OfToolUse: &anthropic.ToolUseBlockParam{
							ID:    cb.ToolUse.ID,
							Name:  cb.ToolUse.Name,
							Input: json.RawMessage(inputJSON),
						},
					})
				}
			case "tool_result":
				if cb.ToolResult != nil {
					blocks = append(blocks, anthropic.NewToolResultBlock(
						cb.ToolResult.ToolUseID,
						cb.ToolResult.Content,
						cb.ToolResult.IsError,
					))
				}
			}
		}
		messages = append(messages, anthropic.MessageParam{
			Role:    anthropic.MessageParamRole(msg.Role),
			Content: blocks,
		})
	}

	// Build tools
	tools := make([]anthropic.ToolUnionParam, 0, len(params.Tools))
	for _, t := range params.Tools {
		tools = append(tools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: t.InputSchema["properties"],
					Required:   toStringSlice(t.InputSchema["required"]),
				},
			},
		})
	}

	// Create the stream
	messageParams := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: int64(maxTokens),
		Messages:  messages,
	}
	if params.System != "" {
		messageParams.System = []anthropic.TextBlockParam{
			{Text: params.System},
		}
	}
	if len(tools) > 0 {
		messageParams.Tools = tools
	}

	stream := a.client.Messages.NewStreaming(ctx, messageParams)
	defer func() { _ = stream.Close() }()

	// Process the stream using Accumulate
	resp := &Response{}
	message := anthropic.Message{}
	var currentToolUse *ToolUse
	var toolInputJSON strings.Builder

	for stream.Next() {
		event := stream.Current()
		_ = message.Accumulate(event)

		switch event.Type {
		case "content_block_start":
			if event.ContentBlock.Type == "tool_use" {
				name := event.ContentBlock.Name
				currentToolUse = &ToolUse{
					ID:    event.ContentBlock.ID,
					Name:  name,
					Input: make(map[string]interface{}),
				}
				toolInputJSON.Reset()
				if params.OnToolStart != nil {
					params.OnToolStart(name)
				}
			}

		case "content_block_delta":
			if event.Delta.Type == "text_delta" {
				text := event.Delta.Text
				if params.OnTextDelta != nil {
					params.OnTextDelta(text)
				}
			} else if event.Delta.Type == "input_json_delta" {
				toolInputJSON.WriteString(event.Delta.PartialJSON)
			}

		case "content_block_stop":
			if currentToolUse != nil {
				if toolInputJSON.Len() > 0 {
					if err := json.Unmarshal([]byte(toolInputJSON.String()), &currentToolUse.Input); err != nil {
						return nil, fmt.Errorf("anthropic: malformed tool call JSON for %q: %w", currentToolUse.Name, err)
					}
				}
				resp.Content = append(resp.Content, ContentBlock{
					Type:    "tool_use",
					ToolUse: currentToolUse,
				})
				currentToolUse = nil
				toolInputJSON.Reset()
			}

		case "message_delta":
			if event.Delta.StopReason != "" {
				resp.StopReason = string(event.Delta.StopReason)
			}
			resp.Usage.OutputTokens = int(event.Usage.OutputTokens)
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("anthropic stream: %w", err)
	}

	// Collect text from the accumulated message
	for _, block := range message.Content {
		if block.Type == "text" {
			resp.Content = append([]ContentBlock{{
				Type: "text",
				Text: block.Text,
			}}, resp.Content...)
		}
	}
	resp.Usage.InputTokens = int(message.Usage.InputTokens)

	return resp, nil
}

// toStringSlice converts an interface{} to []string (for JSON schema "required" fields).
func toStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}
