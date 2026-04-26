package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/dundeezhang/agent-sh/internal/provider"
	"github.com/dundeezhang/agent-sh/internal/render"
	"github.com/dundeezhang/agent-sh/internal/tools"
)

const maxTurns = 25

// Agent orchestrates the conversation loop with the LLM.
type Agent struct {
	provider    provider.Provider
	model       string
	registry    *tools.Registry
	renderer    *render.Renderer
	includeGit  bool
	autoApprove bool
}

// New creates a new Agent.
func New(p provider.Provider, model string, registry *tools.Registry, renderer *render.Renderer, includeGit bool) *Agent {
	return &Agent{
		provider:   p,
		model:      model,
		registry:   registry,
		renderer:   renderer,
		includeGit: includeGit,
	}
}

// Run executes the agent loop for a single user query.
// previousContext is injected into the system prompt (may be empty).
// Returns an InteractionResult summarising what happened.
func (a *Agent) Run(query string, recentCommands []string, previousContext string) *InteractionResult {
	ctx := context.Background()
	shellCtx := GatherContext(a.includeGit)
	shellCtx.RecentCommands = recentCommands
	shellCtx.PreviousContext = previousContext
	systemPrompt := shellCtx.SystemPrompt()

	messages := []provider.Message{
		{
			Role:    "user",
			Content: []provider.ContentBlock{{Type: "text", Text: query}},
		},
	}

	var toolCallRecords []ToolCallRecord
	var lastAssistantText string

	for turn := 0; turn < maxTurns; turn++ {
		a.renderer.StartSpinner("Thinking...")
		sw := a.renderer.Stream()

		resp, err := a.provider.Stream(ctx, provider.StreamParams{
			Model:       a.model,
			System:      systemPrompt,
			Messages:    messages,
			Tools:       a.registry.Tools(),
			MaxTokens:   8192,
			OnTextDelta: sw.Text,
			OnToolStart: sw.ToolStart,
		})
		a.renderer.StopSpinner()

		if writeErr := sw.Flush(); writeErr != nil {
			a.renderer.Error(fmt.Sprintf("Error writing output: %s", writeErr))
			return buildResult(query, toolCallRecords, lastAssistantText)
		}
		if err != nil {
			a.renderer.Error(fmt.Sprintf("Error: %s", err))
			return buildResult(query, toolCallRecords, lastAssistantText)
		}

		if t := extractText(resp.Content); t != "" {
			lastAssistantText = t
		}

		messages = append(messages, provider.Message{
			Role:    "assistant",
			Content: resp.Content,
		})

		// End of turn — model produced a final response.
		if resp.StopReason != "tool_use" {
			a.renderer.Newline()
			a.renderer.Usage(resp.Usage)
			return buildResult(query, toolCallRecords, lastAssistantText)
		}

		toolResults := a.runToolCalls(resp.Content, &toolCallRecords)
		messages = append(messages, provider.Message{
			Role:    "user",
			Content: toolResults,
		})
	}

	a.renderer.Error(fmt.Sprintf("Agent stopped after %d turns (safety limit)", maxTurns))
	return buildResult(query, toolCallRecords, lastAssistantText)
}

// runToolCalls executes every tool_use block in content, rendering progress
// and collecting tool_result blocks plus a record for each call.
func (a *Agent) runToolCalls(content []provider.ContentBlock, records *[]ToolCallRecord) []provider.ContentBlock {
	var toolResults []provider.ContentBlock
	for _, block := range content {
		if block.Type != "tool_use" || block.ToolUse == nil {
			continue
		}
		tu := block.ToolUse

		a.renderer.ToolCall(tu.Name, tu.Input)
		result := a.executeTool(tu)
		a.renderer.ToolResult(tu.Name, result.Content, result.IsError)

		*records = append(*records, ToolCallRecord{
			Tool:    tu.Name,
			Input:   summarizeToolInput(tu.Name, tu.Input),
			IsError: result.IsError,
		})

		toolResults = append(toolResults, provider.ContentBlock{
			Type: "tool_result",
			ToolResult: &provider.ToolResult{
				ToolUseID: tu.ID,
				Content:   result.Content,
				IsError:   result.IsError,
			},
		})
	}
	return toolResults
}

// executeTool runs a single tool call, with bash-specific safety checks.
func (a *Agent) executeTool(tu *provider.ToolUse) tools.ToolResult {
	if tu.Name != "bash" {
		return a.registry.Execute(tu.Name, tu.Input)
	}

	cmd, _ := tu.Input["command"].(string)
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return tools.ToolResult{Content: "Empty command.", IsError: true}
	}
	if !isReadOnlyBash(cmd) && !a.confirmBash(cmd) {
		return tools.ToolResult{Content: "User denied this command.", IsError: true}
	}
	return a.registry.Execute(tu.Name, tu.Input)
}
