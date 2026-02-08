package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/dundeezhang/agent-sh/internal/provider"
)

func (r *Registry) registerBash() {
	r.register("bash", provider.Tool{
		Name:        "bash",
		Description: "Execute a shell command via bash. Returns stdout and stderr. Use this for running commands, installing packages, git operations, etc.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The shell command to execute",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Timeout in seconds (default 120)",
				},
			},
			"required": []string{"command"},
		},
	}, executeBash)
}

func executeBash(input map[string]interface{}) ToolResult {
	command, ok := input["command"].(string)
	if !ok || command == "" {
		return ToolResult{Content: "command is required", IsError: true}
	}

	timeout := 120
	if t, ok := input["timeout"].(float64); ok && t > 0 {
		timeout = int(t)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	var output string
	if stdout.Len() > 0 {
		output = stdout.String()
	}
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ToolResult{
				Content: fmt.Sprintf("command timed out after %ds\n%s", timeout, output),
				IsError: true,
			}
		}
		if output == "" {
			output = err.Error()
		}
		return ToolResult{Content: output, IsError: true}
	}

	if output == "" {
		output = "(no output)"
	}
	return ToolResult{Content: output}
}
