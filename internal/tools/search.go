package tools

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/dundeezhang/agent-sh/internal/provider"
)

func (r *Registry) registerSearch() {
	r.register("search", provider.Tool{
		Name:        "search",
		Description: "Search for a regex pattern in files using ripgrep (rg). Falls back to grep if rg is not installed. Returns matching lines with file paths and line numbers.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "The regex pattern to search for",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory or file to search in (default: current directory)",
				},
				"glob": map[string]interface{}{
					"type":        "string",
					"description": "File glob pattern to filter (e.g. '*.go', '*.js')",
				},
			},
			"required": []string{"pattern"},
		},
	}, executeSearch)
}

func (r *Registry) registerGlob() {
	r.register("glob", provider.Tool{
		Name:        "glob",
		Description: "Find files matching a glob pattern. Supports ** for recursive matching (e.g. '**/*.go'). Returns matching file paths.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "The glob pattern to match (e.g. '**/*.go', 'src/**/*.ts')",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Base directory (default: current directory)",
				},
			},
			"required": []string{"pattern"},
		},
	}, executeGlob)
}

func executeSearch(input map[string]interface{}) ToolResult {
	pattern, ok := input["pattern"].(string)
	if !ok || pattern == "" {
		return ToolResult{Content: "pattern is required", IsError: true}
	}

	searchPath := "."
	if p, ok := input["path"].(string); ok && p != "" {
		searchPath = p
	}

	globFilter := ""
	if g, ok := input["glob"].(string); ok && g != "" {
		globFilter = g
	}

	// Try ripgrep first
	args := []string{"-n", "--no-heading", "--color=never"}
	if globFilter != "" {
		args = append(args, "--glob", globFilter)
	}
	args = append(args, pattern, searchPath)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("rg", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if rg is not found — fall back to grep
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return executeSearchGrep(pattern, searchPath, globFilter)
		}
		// Exit code 1 means no matches (not an error)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return ToolResult{Content: "no matches found"}
		}
		if stderr.Len() > 0 {
			return ToolResult{Content: stderr.String(), IsError: true}
		}
	}

	output := stdout.String()
	if output == "" {
		return ToolResult{Content: "no matches found"}
	}
	return ToolResult{Content: output}
}

func executeSearchGrep(pattern, path, globFilter string) ToolResult {
	args := []string{"-rn", "--color=never"}
	if globFilter != "" {
		args = append(args, "--include", globFilter)
	}
	args = append(args, pattern, path)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("grep", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return ToolResult{Content: "no matches found"}
		}
		if stderr.Len() > 0 {
			return ToolResult{Content: stderr.String(), IsError: true}
		}
	}

	output := stdout.String()
	if output == "" {
		return ToolResult{Content: "no matches found"}
	}
	return ToolResult{Content: output}
}

func executeGlob(input map[string]interface{}) ToolResult {
	pattern, ok := input["pattern"].(string)
	if !ok || pattern == "" {
		return ToolResult{Content: "pattern is required", IsError: true}
	}

	basePath := "."
	if p, ok := input["path"].(string); ok && p != "" {
		basePath = p
	}

	fullPattern := filepath.Join(basePath, pattern)
	matches, err := doublestar.FilepathGlob(fullPattern)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("glob error: %s", err), IsError: true}
	}

	if len(matches) == 0 {
		return ToolResult{Content: "no files matched"}
	}

	return ToolResult{Content: strings.Join(matches, "\n")}
}
