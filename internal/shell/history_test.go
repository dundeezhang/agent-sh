package shell

import "testing"

func TestHistorySearchBackward(t *testing.T) {
	h := NewHistory(100)
	h.Add("ls -la")
	h.Add("git status")
	h.Add("go build ./...")
	h.Add("git commit -m 'fix'")
	h.Add("ls /tmp")

	// Search for "git" starting from end — should find most recent "git" entry.
	entry, idx, ok := h.SearchBackward("git", h.Len())
	if !ok {
		t.Fatal("expected to find a match for 'git'")
	}
	if entry != "git commit -m 'fix'" {
		t.Errorf("expected 'git commit -m fix', got %q", entry)
	}
	if idx != 3 {
		t.Errorf("expected index 3, got %d", idx)
	}

	// Continue searching from that index — should find older "git" entry.
	entry, idx, ok = h.SearchBackward("git", idx)
	if !ok {
		t.Fatal("expected to find an older match for 'git'")
	}
	if entry != "git status" {
		t.Errorf("expected 'git status', got %q", entry)
	}
	if idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}

	// Continue searching — no more matches.
	_, _, ok = h.SearchBackward("git", idx)
	if ok {
		t.Error("expected no more matches for 'git'")
	}
}

func TestHistorySearchBackwardNoMatch(t *testing.T) {
	h := NewHistory(100)
	h.Add("ls -la")
	h.Add("pwd")

	_, _, ok := h.SearchBackward("git", h.Len())
	if ok {
		t.Error("expected no match for 'git'")
	}
}

func TestHistorySearchBackwardEmptyQuery(t *testing.T) {
	h := NewHistory(100)
	h.Add("ls")

	_, _, ok := h.SearchBackward("", h.Len())
	if ok {
		t.Error("expected no match for empty query")
	}
}

func TestHistorySearchBackwardEmptyHistory(t *testing.T) {
	h := NewHistory(100)

	_, _, ok := h.SearchBackward("ls", h.Len())
	if ok {
		t.Error("expected no match in empty history")
	}
}

func TestHistoryLen(t *testing.T) {
	h := NewHistory(100)
	if h.Len() != 0 {
		t.Errorf("expected len 0, got %d", h.Len())
	}
	h.Add("ls")
	h.Add("pwd")
	if h.Len() != 2 {
		t.Errorf("expected len 2, got %d", h.Len())
	}
}
