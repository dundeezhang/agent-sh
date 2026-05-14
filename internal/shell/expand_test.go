package shell

import "testing"

// helper to build a history with the given commands.
func historyWith(cmds ...string) *History {
	h := NewHistory(100)
	for _, c := range cmds {
		h.Add(c)
	}
	return h
}

func TestExpandHistory_DoubleBank(t *testing.T) {
	h := historyWith("ls -la", "echo hello")

	got, changed, err := ExpandHistory("!!", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got != "echo hello" {
		t.Errorf("got %q, want %q", got, "echo hello")
	}
}

func TestExpandHistory_DoubleBangInline(t *testing.T) {
	h := historyWith("cat file.txt")

	got, changed, err := ExpandHistory("sudo !!", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got != "sudo cat file.txt" {
		t.Errorf("got %q, want %q", got, "sudo cat file.txt")
	}
}

func TestExpandHistory_LastArg(t *testing.T) {
	h := historyWith("git commit -m 'initial'")

	got, changed, err := ExpandHistory("echo !$", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got != "echo initial" {
		t.Errorf("got %q, want %q", got, "echo initial")
	}
}

func TestExpandHistory_LastArgSimple(t *testing.T) {
	h := historyWith("cp foo.txt bar.txt")

	got, changed, err := ExpandHistory("vim !$", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got != "vim bar.txt" {
		t.Errorf("got %q, want %q", got, "vim bar.txt")
	}
}

func TestExpandHistory_AbsoluteIndex(t *testing.T) {
	h := historyWith("first", "second", "third")

	got, changed, err := ExpandHistory("!2", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got != "second" {
		t.Errorf("got %q, want %q", got, "second")
	}
}

func TestExpandHistory_RelativeIndex(t *testing.T) {
	h := historyWith("first", "second", "third")

	got, changed, err := ExpandHistory("!-2", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got != "second" {
		t.Errorf("got %q, want %q", got, "second")
	}
}

func TestExpandHistory_PrefixSearch(t *testing.T) {
	h := historyWith("git status", "make build", "git log --oneline")

	got, changed, err := ExpandHistory("!git", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got != "git log --oneline" {
		t.Errorf("got %q, want %q", got, "git log --oneline")
	}
}

func TestExpandHistory_QuickSubst(t *testing.T) {
	h := historyWith("echo hello world")

	got, changed, err := ExpandHistory("^hello^goodbye", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got != "echo goodbye world" {
		t.Errorf("got %q, want %q", got, "echo goodbye world")
	}
}

func TestExpandHistory_QuickSubstTrailingCaret(t *testing.T) {
	h := historyWith("echo hello world")

	got, changed, err := ExpandHistory("^hello^goodbye^", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got != "echo goodbye world" {
		t.Errorf("got %q, want %q", got, "echo goodbye world")
	}
}

func TestExpandHistory_NoExpansion(t *testing.T) {
	h := historyWith("ls -la")

	tests := []string{
		"echo hello",
		"ls -la",
		"git status",
		"FOO=bar",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got, changed, err := ExpandHistory(input, h)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if changed {
				t.Fatalf("expected changed=false, got expanded=%q", got)
			}
			if got != input {
				t.Errorf("got %q, want %q", got, input)
			}
		})
	}
}

func TestExpandHistory_InsideQuotes(t *testing.T) {
	h := historyWith("ls -la")

	tests := []struct {
		name  string
		input string
	}{
		{"single quotes", "echo '!!'"},
		{"double quotes", `echo "!!"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := ExpandHistory(tt.input, h)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if changed {
				t.Fatalf("expected no expansion inside quotes, got %q", got)
			}
			if got != tt.input {
				t.Errorf("got %q, want %q", got, tt.input)
			}
		})
	}
}

func TestExpandHistory_EmptyHistory(t *testing.T) {
	h := NewHistory(10)

	_, _, err := ExpandHistory("!!", h)
	if err == nil {
		t.Fatal("expected error for !! on empty history")
	}

	_, _, err = ExpandHistory("!$", h)
	if err == nil {
		t.Fatal("expected error for !$ on empty history")
	}

	_, _, err = ExpandHistory("!1", h)
	if err == nil {
		t.Fatal("expected error for !1 on empty history")
	}
}

func TestExpandHistory_OutOfRange(t *testing.T) {
	h := historyWith("only one")

	_, _, err := ExpandHistory("!999", h)
	if err == nil {
		t.Fatal("expected error for out-of-range !999")
	}

	_, _, err = ExpandHistory("!-5", h)
	if err == nil {
		t.Fatal("expected error for out-of-range !-5")
	}
}

func TestExpandHistory_PrefixNotFound(t *testing.T) {
	h := historyWith("ls -la", "echo hello")

	_, _, err := ExpandHistory("!zzz", h)
	if err == nil {
		t.Fatal("expected error for prefix not found")
	}
}

func TestExpandHistory_QuickSubstBadPattern(t *testing.T) {
	h := historyWith("echo hello")

	// Empty old string
	_, _, err := ExpandHistory("^^new", h)
	if err == nil {
		t.Fatal("expected error for empty substitution pattern")
	}

	// Old string not found in last command
	_, _, err = ExpandHistory("^notfound^new", h)
	if err == nil {
		t.Fatal("expected error for substitution of string not in last command")
	}
}

func TestExpandHistory_MultipleExpansions(t *testing.T) {
	h := historyWith("echo hello")

	got, changed, err := ExpandHistory("!! && !!", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if got != "echo hello && echo hello" {
		t.Errorf("got %q, want %q", got, "echo hello && echo hello")
	}
}

func TestExpandHistory_BangSpace(t *testing.T) {
	// "! " should not trigger expansion.
	h := historyWith("ls -la")

	got, changed, err := ExpandHistory("echo ! done", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false for '! '")
	}
	if got != "echo ! done" {
		t.Errorf("got %q, want %q", got, "echo ! done")
	}
}

func TestExpandHistory_TrailingBang(t *testing.T) {
	// A trailing "!" should not trigger expansion.
	h := historyWith("ls -la")

	got, changed, err := ExpandHistory("echo wow!", h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false for trailing !")
	}
	if got != "echo wow!" {
		t.Errorf("got %q, want %q", got, "echo wow!")
	}
}

func TestExpandHistory_QuickSubstNoSecondCaret(t *testing.T) {
	h := historyWith("echo hello")

	// ^old without second ^ — should not expand.
	got, changed, _ := ExpandHistory("^nosecondcaret", h)
	if changed {
		t.Fatalf("expected no expansion, got %q", got)
	}
}

func TestHistoryHelpers(t *testing.T) {
	h := historyWith("first", "second", "third")

	t.Run("Len", func(t *testing.T) {
		if n := h.Len(); n != 3 {
			t.Errorf("Len() = %d, want 3", n)
		}
	})

	t.Run("Last", func(t *testing.T) {
		cmd, err := h.Last()
		if err != nil {
			t.Fatal(err)
		}
		if cmd != "third" {
			t.Errorf("Last() = %q, want %q", cmd, "third")
		}
	})

	t.Run("Get", func(t *testing.T) {
		cmd, err := h.Get(2)
		if err != nil {
			t.Fatal(err)
		}
		if cmd != "second" {
			t.Errorf("Get(2) = %q, want %q", cmd, "second")
		}
	})

	t.Run("Get out of range", func(t *testing.T) {
		_, err := h.Get(0)
		if err == nil {
			t.Fatal("expected error for Get(0)")
		}
		_, err = h.Get(99)
		if err == nil {
			t.Fatal("expected error for Get(99)")
		}
	})

	t.Run("GetFromEnd", func(t *testing.T) {
		cmd, err := h.GetFromEnd(1)
		if err != nil {
			t.Fatal(err)
		}
		if cmd != "third" {
			t.Errorf("GetFromEnd(1) = %q, want %q", cmd, "third")
		}
		cmd, err = h.GetFromEnd(3)
		if err != nil {
			t.Fatal(err)
		}
		if cmd != "first" {
			t.Errorf("GetFromEnd(3) = %q, want %q", cmd, "first")
		}
	})

	t.Run("Search", func(t *testing.T) {
		cmd, err := h.Search("sec")
		if err != nil {
			t.Fatal(err)
		}
		if cmd != "second" {
			t.Errorf("Search(\"sec\") = %q, want %q", cmd, "second")
		}
	})

	t.Run("Search not found", func(t *testing.T) {
		_, err := h.Search("zzz")
		if err == nil {
			t.Fatal("expected error for Search(\"zzz\")")
		}
	})
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"ls -la", []string{"ls", "-la"}},
		{"echo 'hello world'", []string{"echo", "hello world"}},
		{`echo "hello world"`, []string{"echo", "hello world"}},
		{"git commit -m 'initial'", []string{"git", "commit", "-m", "initial"}},
		{"  spaced  out  ", []string{"spaced", "out"}},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitArgs(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitArgs(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitArgs(%q)[%d] = %q, want %q",
						tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLastArg(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{"ls -la /tmp", "/tmp"},
		{"echo hello", "hello"},
		{"git commit -m 'initial commit'", "initial commit"},
		{"solo", "solo"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := lastArg(tt.cmd)
			if got != tt.want {
				t.Errorf("lastArg(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}
