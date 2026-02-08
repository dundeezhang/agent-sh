package shell

import "sync"

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
