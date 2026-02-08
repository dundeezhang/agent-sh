package shell

import "sync"

// HistoryEntry stores a command.
type HistoryEntry struct {
	Command string
}

// History is a thread-safe ring buffer of recent commands.
type History struct {
	mu      sync.Mutex
	entries []HistoryEntry
	size    int
}

func NewHistory(size int) *History {
	if size <= 0 {
		size = 20
	}
	return &History{
		entries: make([]HistoryEntry, 0, size),
		size:    size,
	}
}

func (h *History) Add(entry HistoryEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.entries) >= h.size {
		h.entries = h.entries[1:]
	}
	h.entries = append(h.entries, entry)
}

func (h *History) Recent(n int) []HistoryEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	if n <= 0 || n > len(h.entries) {
		n = len(h.entries)
	}
	start := len(h.entries) - n
	result := make([]HistoryEntry, n)
	copy(result, h.entries[start:])
	return result
}

// Commands returns all command strings for history navigation.
func (h *History) Commands() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	cmds := make([]string, len(h.entries))
	for i, e := range h.entries {
		cmds[i] = e.Command
	}
	return cmds
}
