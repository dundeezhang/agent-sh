package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAI implements Provider using the OpenAI API.
type OpenAI struct {
	client *openai.Client
}

// NewOpenAI creates a new OpenAI provider.
func NewOpenAI(apiKey string, baseURL string) *OpenAI {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	return &OpenAI{client: openai.NewClientWithConfig(config)}
}

func (o *OpenAI) resolveModel(model string) string {
	switch model {
	case "gpt4", "gpt-4":
		return openai.GPT4o
	case "gpt4o", "gpt-4o":
		return openai.GPT4o
	case "gpt4o-mini", "gpt-4o-mini":
		return openai.GPT4oMini
	default:
		return model
	}
}

func (o *OpenAI) Stream(ctx context.Context, params StreamParams) (*Response, error) {
	model := o.resolveModel(params.Model)
	maxTokens := params.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}

	// Build messages
	messages := []openai.ChatCompletionMessage{}
	if params.System != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: params.System,
		})
	}

	for _, msg := range params.Messages {
		oaiMsg := openai.ChatCompletionMessage{
			Role: msg.Role,
		}

		// Check for tool results (function role messages)
		hasToolResults := false
		for _, cb := range msg.Content {
			if cb.Type == "tool_result" && cb.ToolResult != nil {
				hasToolResults = true
				break
			}
		}

		if hasToolResults {
			// Send each tool result as a separate message
			for _, cb := range msg.Content {
				if cb.Type == "tool_result" && cb.ToolResult != nil {
					messages = append(messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    cb.ToolResult.Content,
						ToolCallID: cb.ToolResult.ToolUseID,
					})
				}
			}
			continue
		}

		// Regular message: collect text and tool calls
		for _, cb := range msg.Content {
			switch cb.Type {
			case "text":
				oaiMsg.Content = cb.Text
			case "tool_use":
				if cb.ToolUse != nil {
					inputJSON, _ := json.Marshal(cb.ToolUse.Input)
					oaiMsg.ToolCalls = append(oaiMsg.ToolCalls, openai.ToolCall{
						ID:   cb.ToolUse.ID,
						Type: openai.ToolTypeFunction,
						Function: openai.FunctionCall{
							Name:      cb.ToolUse.Name,
							Arguments: string(inputJSON),
						},
					})
				}
			}
		}
		messages = append(messages, oaiMsg)
	}

	// Build tools
	var tools []openai.Tool
	for _, t := range params.Tools {
		schemaJSON, _ := json.Marshal(t.InputSchema)
		var schemaMap map[string]interface{}
		_ = json.Unmarshal(schemaJSON, &schemaMap)

		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schemaMap,
			},
		})
	}

	req := openai.ChatCompletionRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: maxTokens,
		Stream:    true,
	}
	if len(tools) > 0 {
		req.Tools = tools
	}

	stream, err := o.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai stream: %w", err)
	}
	defer stream.Close()

	resp := &Response{}
	var textContent string
	toolCalls := make(map[int]*ToolUse) // index → tool use
	toolArgs := make(map[int]string)    // index → accumulated JSON

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("openai stream recv: %w", err)
		}

		for _, choice := range chunk.Choices {
			delta := choice.Delta

			// Text content
			if delta.Content != "" {
				textContent += delta.Content
				if params.OnTextDelta != nil {
					params.OnTextDelta(delta.Content)
				}
			}

			// Tool calls
			for _, tc := range delta.ToolCalls {
				idx := 0
				if tc.Index != nil {
					idx = *tc.Index
				}

				if tc.ID != "" {
					toolCalls[idx] = &ToolUse{
						ID:    tc.ID,
						Name:  tc.Function.Name,
						Input: make(map[string]interface{}),
					}
					toolArgs[idx] = ""
					if params.OnToolStart != nil {
						params.OnToolStart(tc.Function.Name)
					}
				}
				if tc.Function.Arguments != "" {
					toolArgs[idx] += tc.Function.Arguments
				}
			}

			// Check finish reason
			if choice.FinishReason != "" {
				switch choice.FinishReason {
				case openai.FinishReasonToolCalls:
					resp.StopReason = "tool_use"
				default:
					resp.StopReason = "end_turn"
				}
			}
		}
	}

	// Build response content
	if textContent != "" {
		resp.Content = append(resp.Content, ContentBlock{
			Type: "text",
			Text: textContent,
		})
	}

	for idx, tu := range toolCalls {
		if args, ok := toolArgs[idx]; ok && args != "" {
			_ = json.Unmarshal([]byte(args), &tu.Input)
		}
		resp.Content = append(resp.Content, ContentBlock{
			Type:    "tool_use",
			ToolUse: tu,
		})
	}

	return resp, nil
}
