package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewHistory_InMemory(t *testing.T) {
	h := NewHistory(5, "")
	h.Add("echo hello")
	h.Add("ls")
	got := h.Recent(0)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0] != "echo hello" || got[1] != "ls" {
		t.Fatalf("unexpected entries: %v", got)
	}
}

func TestHistory_ConsecutiveDuplicates(t *testing.T) {
	h := NewHistory(10, "")
	h.Add("ls")
	h.Add("ls")
	h.Add("ls")
	h.Add("pwd")
	h.Add("ls")
	got := h.Recent(0)
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(got), got)
	}
	want := []string{"ls", "pwd", "ls"}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("entry %d: got %q, want %q", i, got[i], w)
		}
	}
}

func TestHistory_RingBuffer(t *testing.T) {
	h := NewHistory(3, "")
	h.Add("a")
	h.Add("b")
	h.Add("c")
	h.Add("d")
	got := h.Recent(0)
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[0] != "b" || got[1] != "c" || got[2] != "d" {
		t.Fatalf("unexpected entries: %v", got)
	}
}

func TestHistory_Persistence(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "history")

	// Create history and add entries.
	h1 := NewHistory(10, fp)
	h1.Add("echo one")
	h1.Add("echo two")
	h1.Add("echo three")

	// Verify the file was written.
	data, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("reading history file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines in file, got %d: %v", len(lines), lines)
	}

	// Create a new history from the same file and verify entries are loaded.
	h2 := NewHistory(10, fp)
	got := h2.Recent(0)
	if len(got) != 3 {
		t.Fatalf("expected 3 entries after reload, got %d", len(got))
	}
	if got[0] != "echo one" || got[2] != "echo three" {
		t.Fatalf("unexpected entries after reload: %v", got)
	}

	// Add another entry and verify append.
	h2.Add("echo four")
	data, err = os.ReadFile(fp)
	if err != nil {
		t.Fatalf("reading history file after append: %v", err)
	}
	lines = strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines in file, got %d", len(lines))
	}
}

func TestHistory_PersistenceRespectsDuplicates(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "history")

	h := NewHistory(10, fp)
	h.Add("ls")
	h.Add("ls")
	h.Add("ls")

	data, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("reading history file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line in file (duplicates collapsed), got %d: %v", len(lines), lines)
	}
}

func TestHistory_LoadRespectsSize(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "history")

	// Write a file with more entries than the size limit.
	var content string
	for i := 0; i < 10; i++ {
		content += "cmd" + string(rune('0'+i)) + "\n"
	}
	if err := os.WriteFile(fp, []byte(content), 0o600); err != nil {
		t.Fatalf("writing history file: %v", err)
	}

	h := NewHistory(3, fp)
	got := h.Recent(0)
	if len(got) != 3 {
		t.Fatalf("expected 3 entries (size limit), got %d: %v", len(got), got)
	}
}

func TestDefaultHistoryPath(t *testing.T) {
	path := DefaultHistoryPath()
	if path == "" {
		t.Fatal("DefaultHistoryPath returned empty string")
	}
	if !strings.Contains(path, "agent-sh") {
		t.Errorf("path should contain 'agent-sh': %s", path)
	}
	if !strings.HasSuffix(path, "history") {
		t.Errorf("path should end with 'history': %s", path)
	}
}

func TestDefaultHistoryPath_XDGOverride(t *testing.T) {
	orig := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", orig)

	os.Setenv("XDG_DATA_HOME", "/tmp/xdg-test")
	path := DefaultHistoryPath()
	want := "/tmp/xdg-test/agent-sh/history"
	if path != want {
		t.Errorf("got %q, want %q", path, want)
	}
}
