package shell

import (
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

// SearchBackward searches history backward (from most recent to oldest) for
// an entry containing query. startIdx is the index into entries[] to start
// searching from (exclusive — the search begins at startIdx-1). Pass
// len(entries) to start from the most recent entry.  Returns the matching
// entry, its index in entries[], and whether a match was found.
func (h *History) SearchBackward(query string, startIdx int) (string, int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if query == "" || len(h.entries) == 0 {
		return "", 0, false
	}
	if startIdx > len(h.entries) {
		startIdx = len(h.entries)
	}
	for i := startIdx - 1; i >= 0; i-- {
		if strings.Contains(h.entries[i], query) {
			return h.entries[i], i, true
		}
	}
	return "", 0, false
}

// Len returns the number of history entries.
func (h *History) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.entries)
}
