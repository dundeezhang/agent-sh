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
		Description: "Search for a regex pattern in files using ripgrep (rg). Returns matching lines with file paths and line numbers. Requires ripgrep to be installed.",
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
	pattern, _ := input["pattern"].(string)
	if pattern == "" {
		return ToolResult{Content: "pattern is required", IsError: true}
	}
	searchPath, _ := input["path"].(string)
	if searchPath == "" {
		searchPath = "."
	}
	globFilter, _ := input["glob"].(string)

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
		var execErr *exec.Error
		if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
			return ToolResult{Content: "search requires ripgrep (rg) to be installed", IsError: true}
		}
		// Exit code 1 means no matches — not an error.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return ToolResult{Content: "no matches found"}
		}
		if stderr.Len() > 0 {
			return ToolResult{Content: stderr.String(), IsError: true}
		}
		return ToolResult{Content: err.Error(), IsError: true}
	}

	output := stdout.String()
	if output == "" {
		return ToolResult{Content: "no matches found"}
	}
	return ToolResult{Content: output}
}

func executeGlob(input map[string]interface{}) ToolResult {
	pattern, _ := input["pattern"].(string)
	if pattern == "" {
		return ToolResult{Content: "pattern is required", IsError: true}
	}
	basePath, _ := input["path"].(string)
	if basePath == "" {
		basePath = "."
	}

	matches, err := doublestar.FilepathGlob(filepath.Join(basePath, pattern))
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("glob error: %s", err), IsError: true}
	}
	if len(matches) == 0 {
		return ToolResult{Content: "no files matched"}
	}
	return ToolResult{Content: strings.Join(matches, "\n")}
}
