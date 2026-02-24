package shell

import (
	"bufio"
	"os"
	"path/filepath"
	"sync"
)

// History is a thread-safe command history with optional file persistence.
type History struct {
	mu       sync.Mutex
	entries  []string
	size     int
	filePath string // empty means in-memory only
}

// NewHistory creates a new History. If filePath is non-empty, existing history
// is loaded from disk and new entries are appended to the file. The parent
// directory is created automatically if it does not exist.
func NewHistory(size int, filePath string) *History {
	if size <= 0 {
		size = 20
	}
	h := &History{
		entries:  make([]string, 0, size),
		size:     size,
		filePath: filePath,
	}
	if filePath != "" {
		h.load()
	}
	return h
}

// Add records a command. Consecutive duplicates are collapsed. If a file
// path is configured the entry is also appended to disk. Space-prefixed
// commands should be filtered by the caller before calling Add.
func (h *History) Add(cmd string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Skip consecutive duplicates.
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == cmd {
		return
	}

	if len(h.entries) >= h.size {
		h.entries = h.entries[1:]
	}
	h.entries = append(h.entries, cmd)

	if h.filePath != "" {
		h.appendToFile(cmd)
	}
}

// Recent returns the last n commands (0 means all).
func (h *History) Recent(n int) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if n <= 0 || n > len(h.entries) {
		n = len(h.entries)
	}
	start := len(h.entries) - n
	result := make([]string, n)
	copy(result, h.entries[start:])
	return result
}

// load reads the history file and populates the in-memory ring buffer.
// Only the last h.size entries are kept. Must be called before any concurrent
// access (i.e. from NewHistory).
func (h *History) load() {
	f, err := os.Open(h.filePath)
	if err != nil {
		return // file may not exist yet
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if len(h.entries) >= h.size {
			h.entries = h.entries[1:]
		}
		h.entries = append(h.entries, line)
	}
}

// appendToFile writes a single command to the history file.
func (h *History) appendToFile(cmd string) {
	dir := filepath.Dir(h.filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(h.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(cmd + "\n")
}

// DefaultHistoryPath returns the XDG-compliant default path for the history
// file: ~/.local/share/agent-sh/history.
func DefaultHistoryPath() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "agent-sh", "history")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "agent-sh", "history")
}
