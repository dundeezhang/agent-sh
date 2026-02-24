package shell

import (
	"fmt"
	"strings"
	"sync"
)

// History is a thread-safe ring buffer of recent commands.
type History struct {
	mu      sync.Mutex
	entries []string
	size    int
}

func NewHistory(size int) *History {
	if size <= 0 {
		size = 20
	}
	return &History{
		entries: make([]string, 0, size),
		size:    size,
	}
}

func (h *History) Add(cmd string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.entries) >= h.size {
		h.entries = h.entries[1:]
	}
	h.entries = append(h.entries, cmd)
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

// Len returns the number of entries currently in history.
func (h *History) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.entries)
}

// Last returns the most recent command, or an error if history is empty.
func (h *History) Last() (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.entries) == 0 {
		return "", fmt.Errorf("no commands in history")
	}
	return h.entries[len(h.entries)-1], nil
}

// Get returns the command at the given 1-based index, or an error if
// the index is out of range.
func (h *History) Get(index int) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if index < 1 || index > len(h.entries) {
		return "", fmt.Errorf("!%d: event not found", index)
	}
	return h.entries[index-1], nil
}

// GetFromEnd returns the command n entries back from the end (1 = last).
func (h *History) GetFromEnd(n int) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if n < 1 || n > len(h.entries) {
		return "", fmt.Errorf("!-%d: event not found", n)
	}
	return h.entries[len(h.entries)-n], nil
}

// Search returns the most recent command starting with the given prefix,
// or an error if no match is found.
func (h *History) Search(prefix string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := len(h.entries) - 1; i >= 0; i-- {
		if strings.HasPrefix(h.entries[i], prefix) {
			return h.entries[i], nil
		}
	}
	return "", fmt.Errorf("!%s: event not found", prefix)
}
