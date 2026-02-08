package shell

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCompleteFile_DotSlashPrefix(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "script.sh"), nil, 0755)
	os.WriteFile(filepath.Join(dir, "setup.py"), nil, 0644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	matches := completeFile("./scr")
	if len(matches) != 1 || matches[0] != "./script.sh" {
		t.Errorf("completeFile(\"./scr\") = %v, want [\"./script.sh\"]", matches)
	}
}

func TestCompleteFile_DotDotSlashPrefix(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0755)
	os.WriteFile(filepath.Join(dir, "parent.txt"), nil, 0644)

	orig, _ := os.Getwd()
	os.Chdir(sub)
	defer os.Chdir(orig)

	matches := completeFile("../par")
	if len(matches) != 1 || matches[0] != "../parent.txt" {
		t.Errorf("completeFile(\"../par\") = %v, want [\"../parent.txt\"]", matches)
	}
}

func TestCompleteFile_AbsolutePathPrefix(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), nil, 0644)

	matches := completeFile(dir + "/te")
	want := dir + "/test.txt"
	if len(matches) != 1 || matches[0] != want {
		t.Errorf("completeFile(%q) = %v, want [%q]", dir+"/te", matches, want)
	}
}

func TestCompleteFile_TildePrefix(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("cannot determine home directory")
	}

	// Find an entry in home dir to use as a test target.
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) == 0 {
		t.Skip("cannot read home directory")
	}

	// Find a non-hidden entry to test with.
	var target os.DirEntry
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			target = e
			break
		}
	}
	if target == nil {
		t.Skip("no non-hidden entries in home directory")
	}

	name := target.Name()
	// Use enough of the name to likely get a unique match.
	partial := name
	if len(partial) > 3 {
		partial = partial[:3]
	}
	matches := completeFile("~/" + partial)

	// All matches must start with ~/, not the expanded home path.
	for _, m := range matches {
		if !strings.HasPrefix(m, "~/") {
			t.Errorf("match %q does not start with \"~/\"", m)
		}
		if strings.HasPrefix(m, home) {
			t.Errorf("match %q contains expanded home path", m)
		}
	}
}

func TestCompleteFile_NoSeparator(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.md"), nil, 0644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	matches := completeFile("read")
	if len(matches) != 1 || matches[0] != "readme.md" {
		t.Errorf("completeFile(\"read\") = %v, want [\"readme.md\"]", matches)
	}
}

func TestFormatColumns(t *testing.T) {
	out := formatColumns([]string{"vim", "vimdiff", "vimtutor"}, 80)
	if out == "" {
		t.Fatal("formatColumns returned empty string")
	}
	t.Logf("output:\n%s", out)
}
