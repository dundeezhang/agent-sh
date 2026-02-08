package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ShellContext gathers contextual information about the current environment.
type ShellContext struct {
	CWD            string
	OS             string
	Shell          string
	GitBranch      string
	GitStatus      string
	RecentCommands []string
}

// GatherContext collects the current shell context.
func GatherContext(includeGit bool) ShellContext {
	ctx := ShellContext{
		OS:    runtime.GOOS,
		Shell: os.Getenv("SHELL"),
	}

	if cwd, err := os.Getwd(); err == nil {
		ctx.CWD = cwd
	}

	if includeGit {
		ctx.GitBranch = runQuiet("git", "branch", "--show-current")
		if ctx.GitBranch != "" {
			ctx.GitStatus = runQuiet("git", "status", "--short")
		}
	}

	return ctx
}

// SystemPrompt builds the system prompt from the context.
func (c ShellContext) SystemPrompt() string {
	var sb strings.Builder

	sb.WriteString("You are an AI agent running inside agent-sh, an AI-powered terminal shell.\n")
	sb.WriteString("You have direct access to the user's system through tools. When the user asks you to do something, DO IT using your tools — don't just explain how.\n\n")

	sb.WriteString("## Tools\n")
	sb.WriteString("- **bash**: Run any shell command — git, grep, find, make, npm, curl, docker, etc.\n")
	sb.WriteString("- **read_file**: Read file contents with line numbers.\n")
	sb.WriteString("- **write_file**: Create or overwrite files (auto-creates parent dirs).\n")
	sb.WriteString("- **edit_file**: Find-and-replace edits on existing files.\n")
	sb.WriteString("- **search**: Regex search across files (ripgrep).\n")
	sb.WriteString("- **glob**: Find files matching glob patterns.\n\n")

	sb.WriteString("## Environment\n")
	fmt.Fprintf(&sb, "- Working directory: %s\n", c.CWD)
	fmt.Fprintf(&sb, "- OS: %s\n", c.OS)
	fmt.Fprintf(&sb, "- Shell: %s\n", c.Shell)

	if c.GitBranch != "" {
		fmt.Fprintf(&sb, "- Git branch: %s\n", c.GitBranch)
		if c.GitStatus != "" {
			fmt.Fprintf(&sb, "- Git status:\n%s\n", c.GitStatus)
		}
	}

	if len(c.RecentCommands) > 0 {
		sb.WriteString("\n## Recent shell commands\n")
		for _, cmd := range c.RecentCommands {
			fmt.Fprintf(&sb, "- %s\n", cmd)
		}
	}

	sb.WriteString("\n## Guidelines\n")
	sb.WriteString("- **Take action.** When asked to commit, search, edit, build, or deploy — use your tools to do it immediately.\n")
	sb.WriteString("- For git operations: run git status first to understand the state, then proceed with add/commit/push/etc.\n")
	sb.WriteString("- For finding things: use search (regex across files) or glob (find files by pattern) or bash with grep/find.\n")
	sb.WriteString("- Read files before editing them to understand the current state.\n")
	sb.WriteString("- For multi-step tasks, work through them step by step, checking results as you go.\n")
	sb.WriteString("- When a command fails, read the error and try a different approach.\n")
	sb.WriteString("- Prefer non-interactive bash commands (no vim, no less, no interactive prompts).\n")
	sb.WriteString("- Be concise. Show what you're doing briefly, not lengthy explanations.\n")

	return sb.String()
}

func runQuiet(name string, args ...string) string {
	var stdout bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}
