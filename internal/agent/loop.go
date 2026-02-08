package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dundeezhang/agent-sh/internal/provider"
	"github.com/dundeezhang/agent-sh/internal/render"
	"github.com/dundeezhang/agent-sh/internal/tools"
)

const maxTurns = 25

// Agent orchestrates the conversation loop with the LLM.
type Agent struct {
	provider   provider.Provider
	model      string
	registry   *tools.Registry
	renderer   *render.Renderer
	includeGit bool
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
func (a *Agent) Run(query string, recentCommands []string) {
	ctx := context.Background()
	shellCtx := GatherContext(a.includeGit)
	shellCtx.RecentCommands = recentCommands
	systemPrompt := shellCtx.SystemPrompt()

	messages := []provider.Message{
		{
			Role: "user",
			Content: []provider.ContentBlock{
				{Type: "text", Text: query},
			},
		},
	}

	for turn := 0; turn < maxTurns; turn++ {
		a.renderer.StartSpinner("Thinking...")

		resp, err := a.provider.Stream(ctx, provider.StreamParams{
			Model:     a.model,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     a.registry.Tools(),
			MaxTokens: 8192,
			OnTextDelta: func(text string) {
				a.renderer.StopSpinner()
				fmt.Fprint(os.Stdout, text)
			},
			OnToolStart: func(name string) {
				a.renderer.StopSpinner()
			},
		})

		a.renderer.StopSpinner()

		if err != nil {
			a.renderer.Error(fmt.Sprintf("Error: %s", err))
			return
		}

		// Build assistant message from response
		assistantMsg := provider.Message{
			Role:    "assistant",
			Content: resp.Content,
		}
		messages = append(messages, assistantMsg)

		// If end of turn, we're done
		if resp.StopReason != "tool_use" {
			fmt.Fprintln(os.Stdout)
			a.renderer.Usage(resp.Usage)
			return
		}

		// Execute tool calls
		var toolResults []provider.ContentBlock
		for _, block := range resp.Content {
			if block.Type != "tool_use" || block.ToolUse == nil {
				continue
			}

			tu := block.ToolUse
			a.renderer.ToolCall(tu.Name, tu.Input)

			var result tools.ToolResult
			if tu.Name == "bash" {
				if cmd, ok := tu.Input["command"].(string); ok && !a.confirmBash(cmd) {
					result = tools.ToolResult{Content: "User denied this command.", IsError: true}
				} else {
					result = a.registry.Execute(tu.Name, tu.Input)
				}
			} else {
				result = a.registry.Execute(tu.Name, tu.Input)
			}

			a.renderer.ToolResult(tu.Name, result.Content, result.IsError)

			toolResults = append(toolResults, provider.ContentBlock{
				Type: "tool_result",
				ToolResult: &provider.ToolResult{
					ToolUseID: tu.ID,
					Content:   result.Content,
					IsError:   result.IsError,
				},
			})
		}

		// Add tool results as user message
		messages = append(messages, provider.Message{
			Role:    "user",
			Content: toolResults,
		})
	}

	a.renderer.Error(fmt.Sprintf("Agent stopped after %d turns (safety limit)", maxTurns))
}

// confirmBash prompts the user to approve a bash command. Returns true if approved.
func (a *Agent) confirmBash(cmd string) bool {
	fmt.Fprintf(os.Stdout, "\033[1;33mRun:\033[0m %s\n", cmd)
	fmt.Fprintf(os.Stdout, "\033[2m[y/N]\033[0m ")

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes"
}
