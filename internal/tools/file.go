package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dundeezhang/agent-sh/internal/provider"
)

func (r *Registry) registerReadFile() {
	r.register("read_file", provider.Tool{
		Name:        "read_file",
		Description: "Read the contents of a file. Returns the file content with line numbers. Supports optional offset and limit for large files.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to read",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Line number to start reading from (1-based, default 1)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of lines to read (default: all)",
				},
			},
			"required": []string{"path"},
		},
	}, executeReadFile)
}

func (r *Registry) registerWriteFile() {
	r.register("write_file", provider.Tool{
		Name:        "write_file",
		Description: "Create or overwrite a file with the given content. Parent directories are created automatically.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to write",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
	}, executeWriteFile)
}

func (r *Registry) registerEditFile() {
	r.register("edit_file", provider.Tool{
		Name:        "edit_file",
		Description: "Perform a find-and-replace edit on a file. The old_string must match exactly (including whitespace). Only the first occurrence is replaced unless replace_all is true.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to edit",
				},
				"old_string": map[string]interface{}{
					"type":        "string",
					"description": "The exact string to find and replace",
				},
				"new_string": map[string]interface{}{
					"type":        "string",
					"description": "The replacement string",
				},
				"replace_all": map[string]interface{}{
					"type":        "boolean",
					"description": "Replace all occurrences (default false)",
				},
			},
			"required": []string{"path", "old_string", "new_string"},
		},
	}, executeEditFile)
}

func executeReadFile(input map[string]interface{}) ToolResult {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return ToolResult{Content: "path is required", IsError: true}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("error reading file: %s", err), IsError: true}
	}

	lines := strings.Split(string(data), "\n")

	offset := 1
	if o, ok := input["offset"].(float64); ok && o > 0 {
		offset = int(o)
	}

	limit := len(lines)
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	// Adjust to 0-based index
	start := offset - 1
	if start < 0 {
		start = 0
	}
	if start > len(lines) {
		start = len(lines)
	}
	end := start + limit
	if end > len(lines) {
		end = len(lines)
	}

	var sb strings.Builder
	lineNumWidth := len(strconv.Itoa(end))
	for i := start; i < end; i++ {
		fmt.Fprintf(&sb, "%*d\t%s\n", lineNumWidth, i+1, lines[i])
	}

	return ToolResult{Content: sb.String()}
}

func executeWriteFile(input map[string]interface{}) ToolResult {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return ToolResult{Content: "path is required", IsError: true}
	}
	content, _ := input["content"].(string)

	// Create parent directories
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ToolResult{Content: fmt.Sprintf("error creating directories: %s", err), IsError: true}
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return ToolResult{Content: fmt.Sprintf("error writing file: %s", err), IsError: true}
	}

	return ToolResult{Content: fmt.Sprintf("wrote %d bytes to %s", len(content), path)}
}

func executeEditFile(input map[string]interface{}) ToolResult {
	path, ok := input["path"].(string)
	if !ok || path == "" {
		return ToolResult{Content: "path is required", IsError: true}
	}
	oldStr, ok := input["old_string"].(string)
	if !ok {
		return ToolResult{Content: "old_string is required", IsError: true}
	}
	newStr, _ := input["new_string"].(string)
	replaceAll, _ := input["replace_all"].(bool)

	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("error reading file: %s", err), IsError: true}
	}

	content := string(data)
	if !strings.Contains(content, oldStr) {
		return ToolResult{Content: "old_string not found in file", IsError: true}
	}

	var newContent string
	if replaceAll {
		newContent = strings.ReplaceAll(content, oldStr, newStr)
	} else {
		newContent = strings.Replace(content, oldStr, newStr, 1)
	}

	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return ToolResult{Content: fmt.Sprintf("error writing file: %s", err), IsError: true}
	}

	return ToolResult{Content: fmt.Sprintf("edited %s", path)}
}
