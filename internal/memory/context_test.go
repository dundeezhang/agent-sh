package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRender(t *testing.T) {
	ts := time.Date(2025, 6, 15, 14, 30, 45, 0, time.UTC)
	ctx := &Context{
		Timestamp: ts,
		CWD:       "/home/user/project",
		Query:     "list all Go files",
		ToolCalls: []ToolCallSummary{
			{Tool: "Bash", Input: "find . -name '*.go'"},
			{Tool: "Read", Input: "main.go", IsError: true},
		},
		Summary: "Found 12 Go files in the project.",
	}

	got := Render(ctx)

	// Check header with timestamp.
	if want := "## Previous interaction (14:30:45)\n"; !strings.Contains(got, want) {
		t.Errorf("missing header; got:\n%s", got)
	}
	// Check query line.
	if want := "**Query:** list all Go files\n"; !strings.Contains(got, want) {
		t.Errorf("missing query line; got:\n%s", got)
	}
	// Check actions line includes both tool calls.
	if want := "Bash(`find . -name '*.go'`)"; !strings.Contains(got, want) {
		t.Errorf("missing first tool call; got:\n%s", got)
	}
	if want := "Read(`main.go`)"; !strings.Contains(got, want) {
		t.Errorf("missing second tool call; got:\n%s", got)
	}
	// Check result line.
	if want := "**Result:** Found 12 Go files in the project.\n"; !strings.Contains(got, want) {
		t.Errorf("missing result line; got:\n%s", got)
	}
}

func TestRenderNoToolCalls(t *testing.T) {
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := &Context{
		Timestamp: ts,
		Query:     "hello",
		Summary:   "greeted user",
	}

	got := Render(ctx)

	if strings.Contains(got, "**Actions:**") {
		t.Errorf("should not contain Actions line when no tool calls; got:\n%s", got)
	}
	if !strings.Contains(got, "**Result:** greeted user") {
		t.Errorf("missing result line; got:\n%s", got)
	}
}

func TestRenderNoSummary(t *testing.T) {
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := &Context{
		Timestamp: ts,
		Query:     "hello",
	}

	got := Render(ctx)

	if strings.Contains(got, "**Result:**") {
		t.Errorf("should not contain Result line when summary is empty; got:\n%s", got)
	}
}

func TestRenderNil(t *testing.T) {
	got := Render(nil)
	if got != "" {
		t.Errorf("Render(nil) = %q; want empty string", got)
	}
}

func TestPathFor(t *testing.T) {
	got := pathFor("/base/cache", "/home/user/project")
	want := filepath.Join("/base/cache", "home/user/project", "context.json")
	if got != want {
		t.Errorf("pathFor = %q; want %q", got, want)
	}
}

func TestWriteReadDelete(t *testing.T) {
	// Use a deeply nested unique temp directory as the fake CWD.
	// Write/Read/Delete all route through CacheDir() which is the real user
	// cache directory, but since the CWD is a unique temp path the files
	// will land in a unique subdirectory that won't conflict with anything.
	fakeCWD := t.TempDir()

	ts := time.Date(2025, 7, 4, 12, 0, 0, 0, time.UTC)
	original := &Context{
		Timestamp: ts,
		CWD:       fakeCWD,
		Query:     "run tests",
		ToolCalls: []ToolCallSummary{
			{Tool: "Bash", Input: "go test ./..."},
		},
		Summary: "All tests passed.",
	}

	// Write the context.
	if err := Write(fakeCWD, original); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Ensure the file was actually created on disk.
	baseDir, err := CacheDir()
	if err != nil {
		t.Fatalf("CacheDir: %v", err)
	}
	p := pathFor(baseDir, fakeCWD)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("context file not found at %s: %v", p, err)
	}

	// Clean up on-disk file when the test finishes.
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Dir(p))
	})

	// Read it back.
	got, err := Read(fakeCWD)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got == nil {
		t.Fatal("Read returned nil; expected context")
	}

	// Verify fields.
	if !got.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp = %v; want %v", got.Timestamp, original.Timestamp)
	}
	if got.CWD != original.CWD {
		t.Errorf("CWD = %q; want %q", got.CWD, original.CWD)
	}
	if got.Query != original.Query {
		t.Errorf("Query = %q; want %q", got.Query, original.Query)
	}
	if got.Summary != original.Summary {
		t.Errorf("Summary = %q; want %q", got.Summary, original.Summary)
	}
	if len(got.ToolCalls) != len(original.ToolCalls) {
		t.Fatalf("ToolCalls length = %d; want %d", len(got.ToolCalls), len(original.ToolCalls))
	}
	if got.ToolCalls[0].Tool != original.ToolCalls[0].Tool {
		t.Errorf("ToolCalls[0].Tool = %q; want %q", got.ToolCalls[0].Tool, original.ToolCalls[0].Tool)
	}
	if got.ToolCalls[0].Input != original.ToolCalls[0].Input {
		t.Errorf("ToolCalls[0].Input = %q; want %q", got.ToolCalls[0].Input, original.ToolCalls[0].Input)
	}

	// Delete it.
	if err := Delete(fakeCWD); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify Read now returns nil.
	got, err = Read(fakeCWD)
	if err != nil {
		t.Fatalf("Read after Delete: %v", err)
	}
	if got != nil {
		t.Errorf("Read after Delete returned non-nil: %+v", got)
	}
}

func TestReadNonExistent(t *testing.T) {
	// Use a path that will never have a context file.
	fakeCWD := filepath.Join(t.TempDir(), "nonexistent", "deeply", "nested", "path")

	got, err := Read(fakeCWD)
	if err != nil {
		t.Fatalf("Read non-existent: %v", err)
	}
	if got != nil {
		t.Errorf("Read non-existent returned non-nil: %+v", got)
	}
}

