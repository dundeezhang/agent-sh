package shell

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompleteCommand_Vim(t *testing.T) {
	matches := completeCommand("vim")
	if len(matches) == 0 {
		t.Fatal("completeCommand(\"vim\") returned no matches")
	}
	found := false
	for _, m := range matches {
		if m == "vim" {
			found = true
		}
		t.Logf("match: %s", m)
	}
	if !found {
		t.Error("\"vim\" not in matches")
	}
}

func TestCompleteCommand_EmptyPrefix(t *testing.T) {
	matches := completeCommand("")
	if len(matches) != 0 {
		t.Errorf("completeCommand(\"\") returned %d matches, want 0", len(matches))
	}
}

func TestCompleteCommand_Builtins(t *testing.T) {
	matches := completeCommand("ex")
	found := map[string]bool{}
	for _, m := range matches {
		found[m] = true
	}
	if !found["exit"] {
		t.Error("\"exit\" not in matches for prefix \"ex\"")
	}
	if !found["export"] {
		t.Error("\"export\" not in matches for prefix \"ex\"")
	}
}

func TestCompleteWord_CommandPosition(t *testing.T) {
	wordStart, prefix, matches := completeWord("vim", 3)
	if wordStart != 0 {
		t.Errorf("wordStart = %d, want 0", wordStart)
	}
	if prefix != "vim" {
		t.Errorf("prefix = %q, want %q", prefix, "vim")
	}
	if len(matches) == 0 {
		t.Fatal("no matches for \"vim\" in command position")
	}
	t.Logf("matches: %v", matches)
}

func TestCompleteWord_ArgPosition(t *testing.T) {
	// Create a temp dir with known files
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), nil, 0644)
	os.WriteFile(filepath.Join(dir, "help.txt"), nil, 0644)
	os.Mkdir(filepath.Join(dir, "hidden"), 0755)

	// cd into it for the test
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	line := "cat hel"
	wordStart, prefix, matches := completeWord(line, len(line))
	if wordStart != 4 {
		t.Errorf("wordStart = %d, want 4", wordStart)
	}
	if prefix != "hel" {
		t.Errorf("prefix = %q, want %q", prefix, "hel")
	}
	if len(matches) != 2 {
		t.Fatalf("got %d matches %v, want 2", len(matches), matches)
	}
}

func TestCommonPrefix(t *testing.T) {
	tests := []struct {
		in   []string
		want string
	}{
		{[]string{"vim", "vimdiff", "vimtutor"}, "vim"},
		{[]string{"hello", "help"}, "hel"},
		{[]string{"abc"}, "abc"},
		{nil, ""},
	}
	for _, tt := range tests {
		got := commonPrefix(tt.in)
		if got != tt.want {
			t.Errorf("commonPrefix(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatColumns(t *testing.T) {
	out := formatColumns([]string{"vim", "vimdiff", "vimtutor"}, 80)
	if out == "" {
		t.Fatal("formatColumns returned empty string")
	}
	t.Logf("output:\n%s", out)
}
