package shell

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// CompletionRegistry
// ---------------------------------------------------------------------------

func TestCompletionRegistry_RegisterAndLookup(t *testing.T) {
	r := NewCompletionRegistry()

	called := false
	r.Register("mycmd", func(args []string, cur string) []string {
		called = true
		return []string{"sub1", "sub2"}
	})

	c := r.Lookup("mycmd")
	if c == nil {
		t.Fatal("Lookup returned nil for registered command")
	}
	results := c(nil, "")
	if !called {
		t.Error("custom completer was not called")
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestCompletionRegistry_LookupMissing(t *testing.T) {
	r := NewCompletionRegistry()
	if c := r.Lookup("nonexistent"); c != nil {
		t.Error("expected nil for unregistered command")
	}
}

func TestCompletionRegistry_DefaultsRegistered(t *testing.T) {
	r := NewCompletionRegistry()
	for _, cmd := range []string{"git", "docker", "kubectl", "k"} {
		if r.Lookup(cmd) == nil {
			t.Errorf("default completer for %q not registered", cmd)
		}
	}
}

// ---------------------------------------------------------------------------
// Environment variable completion
// ---------------------------------------------------------------------------

func TestCompleteEnvVar(t *testing.T) {
	// Set a unique env var so we can look for it.
	t.Setenv("AGENTSH_TEST_VAR_ABC", "1")

	matches := completeEnvVar("$AGENTSH_TEST_VAR")
	if len(matches) == 0 {
		t.Fatal("expected at least one match")
	}
	found := false
	for _, m := range matches {
		if m == "$AGENTSH_TEST_VAR_ABC" {
			found = true
		}
	}
	if !found {
		t.Errorf("$AGENTSH_TEST_VAR_ABC not found in matches: %v", matches)
	}
}

func TestCompleteEnvVar_NoPrefix(t *testing.T) {
	matches := completeEnvVar("noprefix")
	if len(matches) != 0 {
		t.Errorf("expected no matches for non-$ prefix, got %v", matches)
	}
}

func TestCompleteEnvVar_DollarOnly(t *testing.T) {
	// "$" alone should match all env vars.
	matches := completeEnvVar("$")
	if len(matches) == 0 {
		t.Error("expected matches for bare $")
	}
	// All matches must start with $.
	for _, m := range matches {
		if m[0] != '$' {
			t.Errorf("match %q does not start with $", m)
		}
	}
}

// ---------------------------------------------------------------------------
// Git completer
// ---------------------------------------------------------------------------

func TestCompleteGit_Subcommands(t *testing.T) {
	matches := completeGit(nil, "co")
	want := []string{"commit", "config"}
	if !equalSorted(matches, want) {
		t.Errorf("git subcommand completion for 'co': got %v, want %v", matches, want)
	}
}

func TestCompleteGit_SubcommandEmpty(t *testing.T) {
	matches := completeGit(nil, "")
	if len(matches) != len(gitSubcommands) {
		t.Errorf("got %d matches, want %d", len(matches), len(gitSubcommands))
	}
}

func TestCompleteGit_Flags(t *testing.T) {
	matches := completeGit([]string{"commit"}, "--am")
	if len(matches) != 1 || matches[0] != "--amend" {
		t.Errorf("git commit --am: got %v, want [--amend]", matches)
	}
}

func TestCompleteGit_FlagsAllForSubcommand(t *testing.T) {
	matches := completeGit([]string{"log"}, "--")
	want := []string{"--all", "--graph", "--oneline", "--patch", "--stat"}
	if !equalSorted(matches, want) {
		t.Errorf("git log --: got %v, want %v", matches, want)
	}
}

func TestCompleteGit_StashSubcommands(t *testing.T) {
	matches := completeGit([]string{"stash"}, "p")
	want := []string{"pop", "push"}
	if !equalSorted(matches, want) {
		t.Errorf("git stash p: got %v, want %v", matches, want)
	}
}

func TestCompleteGit_UnknownSubcommandFlag(t *testing.T) {
	// An unknown subcommand with a flag prefix should return nil.
	matches := completeGit([]string{"unknown"}, "--f")
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %v", matches)
	}
}

// ---------------------------------------------------------------------------
// Docker completer
// ---------------------------------------------------------------------------

func TestCompleteDocker_Subcommands(t *testing.T) {
	matches := completeDocker(nil, "ru")
	if len(matches) != 1 || matches[0] != "run" {
		t.Errorf("docker ru: got %v, want [run]", matches)
	}
}

func TestCompleteDocker_SubcommandEmpty(t *testing.T) {
	matches := completeDocker(nil, "")
	if len(matches) != len(dockerSubcommands) {
		t.Errorf("got %d matches, want %d", len(matches), len(dockerSubcommands))
	}
}

// ---------------------------------------------------------------------------
// Kubectl completer
// ---------------------------------------------------------------------------

func TestCompleteKubectl_Subcommands(t *testing.T) {
	matches := completeKubectl(nil, "ge")
	if len(matches) != 1 || matches[0] != "get" {
		t.Errorf("kubectl ge: got %v, want [get]", matches)
	}
}

func TestCompleteKubectl_SubcommandEmpty(t *testing.T) {
	matches := completeKubectl(nil, "")
	if len(matches) != len(kubectlSubcommands) {
		t.Errorf("got %d matches, want %d", len(matches), len(kubectlSubcommands))
	}
}

// ---------------------------------------------------------------------------
// Integration: completeWordWithRegistry
// ---------------------------------------------------------------------------

func TestCompleteWord_GitSubcommand(t *testing.T) {
	reg := NewCompletionRegistry()
	line := "git st"
	wordStart, prefix, matches := completeWordWithRegistry(line, len(line), reg)
	if wordStart != 4 {
		t.Errorf("wordStart = %d, want 4", wordStart)
	}
	if prefix != "st" {
		t.Errorf("prefix = %q, want %q", prefix, "st")
	}
	// Should include at least "stash", "status".
	found := map[string]bool{}
	for _, m := range matches {
		found[m] = true
	}
	for _, want := range []string{"stash", "status"} {
		if !found[want] {
			t.Errorf("missing %q in matches %v", want, matches)
		}
	}
}

func TestCompleteWord_GitFlags(t *testing.T) {
	reg := NewCompletionRegistry()
	line := "git commit --am"
	wordStart, prefix, matches := completeWordWithRegistry(line, len(line), reg)
	if wordStart != 11 {
		t.Errorf("wordStart = %d, want 11", wordStart)
	}
	if prefix != "--am" {
		t.Errorf("prefix = %q, want %q", prefix, "--am")
	}
	if len(matches) != 1 || matches[0] != "--amend" {
		t.Errorf("matches = %v, want [--amend]", matches)
	}
}

func TestCompleteWord_EnvVar(t *testing.T) {
	t.Setenv("AGENTSH_TEST_COMPLETE", "yes")

	reg := NewCompletionRegistry()
	line := "echo $AGENTSH_TEST_C"
	wordStart, prefix, matches := completeWordWithRegistry(line, len(line), reg)
	if wordStart != 5 {
		t.Errorf("wordStart = %d, want 5", wordStart)
	}
	if prefix != "$AGENTSH_TEST_C" {
		t.Errorf("prefix = %q, want %q", prefix, "$AGENTSH_TEST_C")
	}
	found := false
	for _, m := range matches {
		if m == "$AGENTSH_TEST_COMPLETE" {
			found = true
		}
	}
	if !found {
		t.Errorf("$AGENTSH_TEST_COMPLETE not found in %v", matches)
	}
}

func TestCompleteWord_FallbackToFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "myfile.txt"), nil, 0644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	reg := NewCompletionRegistry()
	line := "cat myf"
	_, _, matches := completeWordWithRegistry(line, len(line), reg)
	if len(matches) != 1 || matches[0] != "myfile.txt" {
		t.Errorf("expected file fallback, got %v", matches)
	}
}

func TestCompleteWord_CustomCompleter(t *testing.T) {
	reg := NewCompletionRegistry()
	reg.Register("mycli", func(args []string, cur string) []string {
		return filterPrefix([]string{"serve", "build", "test"}, cur)
	})

	line := "mycli se"
	_, prefix, matches := completeWordWithRegistry(line, len(line), reg)
	if prefix != "se" {
		t.Errorf("prefix = %q, want %q", prefix, "se")
	}
	if len(matches) != 1 || matches[0] != "serve" {
		t.Errorf("matches = %v, want [serve]", matches)
	}
}

func TestCompleteWord_NilRegistry(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.txt"), nil, 0644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	// With nil registry, should still fall back to file completion.
	line := "git fi"
	_, _, matches := completeWordWithRegistry(line, len(line), nil)
	if len(matches) != 1 || matches[0] != "file.txt" {
		t.Errorf("expected file fallback with nil registry, got %v", matches)
	}
}

// ---------------------------------------------------------------------------
// filterPrefix helper
// ---------------------------------------------------------------------------

func TestFilterPrefix(t *testing.T) {
	in := []string{"add", "apply", "branch", "checkout"}
	got := filterPrefix(in, "a")
	want := []string{"add", "apply"}
	if !equalSorted(got, want) {
		t.Errorf("filterPrefix: got %v, want %v", got, want)
	}
}

func TestFilterPrefix_Empty(t *testing.T) {
	in := []string{"a", "b", "c"}
	got := filterPrefix(in, "")
	if len(got) != 3 {
		t.Errorf("filterPrefix empty prefix: got %d, want 3", len(got))
	}
}

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

func equalSorted(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac, bc := make([]string, len(a)), make([]string, len(b))
	copy(ac, a)
	copy(bc, b)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}
