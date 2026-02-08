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
			Role: "user",
			Content: []provider.ContentBlock{
				{Type: "text", Text: query},
			},
		},
	}

	var toolCallRecords []ToolCallRecord
	var lastAssistantText string

	for turn := 0; turn < maxTurns; turn++ {
		a.renderer.StartSpinner("Thinking...")

		// MarkdownWriter is intentionally created per turn: it tracks whether
		// we are inside a fenced code block (inCode) and buffers partial lines,
		// so reusing one across turns would let stale state from a previous
		// response corrupt the rendering of the next.
		mdw := render.NewMarkdownWriter(os.Stdout)

		var writeErr error
		resp, err := a.provider.Stream(ctx, provider.StreamParams{
			Model:     a.model,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     a.registry.Tools(),
			MaxTokens: 8192,
			OnTextDelta: func(text string) {
				a.renderer.StopSpinner()
				if _, werr := fmt.Fprint(mdw, text); werr != nil && writeErr == nil {
					writeErr = werr
				}
			},
			OnToolStart: func(name string) {
				a.renderer.StopSpinner()
			},
		})

		if ferr := mdw.Flush(); ferr != nil && writeErr == nil {
			writeErr = ferr
		}
		a.renderer.StopSpinner()

		if writeErr != nil {
			a.renderer.Error(fmt.Sprintf("Error writing output: %s", writeErr))
			return buildResult(query, toolCallRecords, lastAssistantText)
		}

		if err != nil {
			a.renderer.Error(fmt.Sprintf("Error: %s", err))
			return buildResult(query, toolCallRecords, lastAssistantText)
		}

		// Capture assistant text
		if t := extractText(resp.Content); t != "" {
			lastAssistantText = t
		}

		// Build assistant message from response
		assistantMsg := provider.Message{
			Role:    "assistant",
			Content: resp.Content,
		}
		messages = append(messages, assistantMsg)

		// If end of turn, we're done
		if resp.StopReason != "tool_use" {
			_, _ = fmt.Fprintln(os.Stdout)
			a.renderer.Usage(resp.Usage)
			return buildResult(query, toolCallRecords, lastAssistantText)
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
				cmd, _ := tu.Input["command"].(string)
				cmd = strings.TrimSpace(cmd)
				if cmd == "" {
					result = tools.ToolResult{Content: "Empty command.", IsError: true}
				} else if !isReadOnlyBash(cmd) && !a.confirmBash(cmd) {
					result = tools.ToolResult{Content: "User denied this command.", IsError: true}
				} else {
					result = a.registry.Execute(tu.Name, tu.Input)
				}
			} else {
				result = a.registry.Execute(tu.Name, tu.Input)
			}

			a.renderer.ToolResult(tu.Name, result.Content, result.IsError)

			toolCallRecords = append(toolCallRecords, ToolCallRecord{
				Tool:    tu.Name,
				Input:   extractToolInput(tu.Name, tu.Input),
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

		// Add tool results as user message
		messages = append(messages, provider.Message{
			Role:    "user",
			Content: toolResults,
		})
	}

	a.renderer.Error(fmt.Sprintf("Agent stopped after %d turns (safety limit)", maxTurns))
	return buildResult(query, toolCallRecords, lastAssistantText)
}

// readOnlyCommands are bash commands that only read state and never modify it.
var readOnlyCommands = map[string]bool{
	"cat": true, "head": true, "tail": true, "less": true, "more": true,
	"ls": true, "dir": true, "tree": true, "file": true, "stat": true,
	"wc": true, "du": true, "df": true,
	"grep": true, "rg": true, "ag": true, "ack": true,
	// find omitted: -delete and -exec can be destructive.
	"fd": true, "locate": true, "which": true, "whereis": true,
	"pwd": true, "whoami": true, "id": true, "hostname": true, "uname": true,
	"date": true, "uptime": true, "env": true, "printenv": true,
	"echo": true, "printf": true,
	"diff": true, "cmp": true, "md5sum": true, "shasum": true,
	"git": true, "go": true,
	"jq": true, "yq": true, "xmllint": true,
	"man": true, "help": true, "type": true,
	"ps": true, "top": true, "htop": true, "pgrep": true,
	"lsof": true, "netstat": true, "ss": true,
}

// readOnlyGitSubcommands are git subcommands that only read state.
var readOnlyGitSubcommands = map[string]bool{
	// branch, tag, remote omitted: can create/delete with args.
	"status": true, "log": true, "diff": true, "show": true,
	"blame": true, "shortlog": true,
	"describe": true, "rev-parse": true, "ls-files": true, "ls-tree": true,
	"cat-file": true, "reflog": true,
}

// mutatingGoSubcommands are go subcommands that modify state.
var mutatingGoSubcommands = map[string]bool{
	"install": true, "get": true, "generate": true, "clean": true,
	"mod": true, "fmt": true,
}

// isReadOnlyBash returns true if a bash command is safe to run without
// user confirmation (read-only, no side effects).
func isReadOnlyBash(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}

	// Reject commands with unquoted redirections or command substitutions.
	if hasUnsafeRedirects(cmd) {
		return false
	}

	// Strip leading env vars (FOO=bar cmd ...) and sudo.
	i := 0
	for i < len(parts) && strings.Contains(parts[i], "=") {
		i++
	}
	if i >= len(parts) {
		return false
	}
	if parts[i] == "sudo" {
		i++
		if i >= len(parts) {
			return false
		}
	}

	base := parts[i]

	// Pipelines/chains: every command in the pipeline must be read-only.
	if strings.ContainsAny(cmd, "|;&") {
		// Split on shell operators and check each segment.
		segments := splitShellSegments(cmd)
		for _, seg := range segments {
			if !isReadOnlyBash(seg) {
				return false
			}
		}
		return true
	}

	if !readOnlyCommands[base] {
		return false
	}

	// git: check subcommand.
	if base == "git" && i+1 < len(parts) {
		return readOnlyGitSubcommands[parts[i+1]]
	}

	// go: only read-only subcommands (build, test, vet, list, version, env, doc, fmt).
	if base == "go" && i+1 < len(parts) {
		return !mutatingGoSubcommands[parts[i+1]]
	}

	return true
}

// splitShellSegments splits a command on unquoted |, &&, ||, ; operators.
func splitShellSegments(cmd string) []string {
	var segments []string
	var current strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
			current.WriteByte(ch)
		case ch == '"' && !inSingle:
			inDouble = !inDouble
			current.WriteByte(ch)
		case inSingle || inDouble:
			current.WriteByte(ch)
		case ch == '|' || ch == ';':
			segments = append(segments, current.String())
			current.Reset()
			// Skip || or |
			if ch == '|' && i+1 < len(cmd) && cmd[i+1] == '|' {
				i++
			}
		case ch == '&':
			if i+1 < len(cmd) && cmd[i+1] == '&' {
				segments = append(segments, current.String())
				current.Reset()
				i++
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		segments = append(segments, current.String())
	}
	return segments
}

// hasUnsafeRedirects returns true if cmd contains unquoted output/input
// redirections (>, >>, <), backticks, or $() command substitutions.
func hasUnsafeRedirects(cmd string) bool {
	inSingle, inDouble := false, false
	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case inSingle || inDouble:
			continue
		case ch == '>' || ch == '<' || ch == '`':
			return true
		case ch == '$' && i+1 < len(cmd) && cmd[i+1] == '(':
			return true
		}
	}
	return false
}

// confirmBash prompts the user to approve a bash command. Returns true if approved.
// Default is Yes (just press Enter). Typing "a" enables auto-approve for the session.
func (a *Agent) confirmBash(cmd string) bool {
	if a.autoApprove {
		fmt.Fprintf(os.Stdout, "\033[1;33mRun:\033[0m %s\n", cmd)
		return true
	}

	fmt.Fprintf(os.Stdout, "\033[1;33mRun:\033[0m %s\n", cmd)
	fmt.Fprintf(os.Stdout, "\033[2m[Y/n/a]\033[0m ")

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(strings.ToLower(line))

	switch answer {
	case "n", "no":
		return false
	case "a", "always":
		a.autoApprove = true
		return true
	default:
		// Enter or "y"/"yes" — approve
		return true
	}
}
