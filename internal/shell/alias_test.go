package shell

import (
	"testing"
)

func TestAliasMap_SetAndGet(t *testing.T) {
	a := NewAliasMap()
	a.Set("ll", "ls -la")

	val, ok := a.Get("ll")
	if !ok {
		t.Fatal("alias 'll' not found after Set")
	}
	if val != "ls -la" {
		t.Errorf("Get(\"ll\") = %q, want %q", val, "ls -la")
	}

	_, ok = a.Get("nonexistent")
	if ok {
		t.Error("Get(\"nonexistent\") returned ok=true")
	}
}

func TestAliasMap_Remove(t *testing.T) {
	a := NewAliasMap()
	a.Set("ll", "ls -la")

	if !a.Remove("ll") {
		t.Error("Remove(\"ll\") returned false")
	}

	_, ok := a.Get("ll")
	if ok {
		t.Error("alias 'll' still exists after Remove")
	}

	if a.Remove("nonexistent") {
		t.Error("Remove(\"nonexistent\") returned true")
	}
}

func TestAliasMap_List(t *testing.T) {
	a := NewAliasMap()
	a.Set("ll", "ls -la")
	a.Set("gs", "git status")
	a.Set("k", "kubectl")

	list := a.List()
	if len(list) != 3 {
		t.Fatalf("List() returned %d entries, want 3", len(list))
	}

	// Should be sorted by name.
	want := []struct{ Name, Value string }{
		{"gs", "git status"},
		{"k", "kubectl"},
		{"ll", "ls -la"},
	}
	for i, entry := range list {
		if entry.Name != want[i].Name || entry.Value != want[i].Value {
			t.Errorf("List()[%d] = {%q, %q}, want {%q, %q}",
				i, entry.Name, entry.Value, want[i].Name, want[i].Value)
		}
	}
}

func TestAliasMap_Names(t *testing.T) {
	a := NewAliasMap()
	a.Set("ll", "ls -la")
	a.Set("gs", "git status")

	names := a.Names()
	if len(names) != 2 {
		t.Fatalf("Names() returned %d entries, want 2", len(names))
	}
	if names[0] != "gs" || names[1] != "ll" {
		t.Errorf("Names() = %v, want [gs ll]", names)
	}
}

func TestAliasMap_Expand(t *testing.T) {
	a := NewAliasMap()
	a.Set("ll", "ls -la")
	a.Set("gs", "git status")
	a.Set("k", "kubectl")

	tests := []struct {
		input string
		want  string
	}{
		// Simple alias expansion.
		{"ll", "ls -la"},
		{"ll /tmp", "ls -la /tmp"},
		{"gs", "git status"},
		{"k get pods", "kubectl get pods"},

		// No alias — no change.
		{"ls -la", "ls -la"},
		{"echo hello", "echo hello"},

		// Only first word is expanded.
		{"echo ll", "echo ll"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := a.Expand(tt.input)
			if got != tt.want {
				t.Errorf("Expand(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAliasMap_Expand_Recursive(t *testing.T) {
	a := NewAliasMap()
	a.Set("k", "kubectl")
	a.Set("kgp", "k get pods")

	got := a.Expand("kgp -n default")
	want := "kubectl get pods -n default"
	if got != want {
		t.Errorf("Expand(\"kgp -n default\") = %q, want %q", got, want)
	}
}

func TestAliasMap_Expand_LoopDetection(t *testing.T) {
	a := NewAliasMap()
	// Create a cycle: a -> b -> a
	a.Set("a", "b")
	a.Set("b", "a")

	got := a.Expand("a hello")
	// Should stop when it detects the loop. After a->b, b->a would loop,
	// so it stops at "a hello".
	want := "a hello"
	if got != want {
		t.Errorf("Expand(\"a hello\") with cycle = %q, want %q", got, want)
	}
}

func TestAliasMap_Expand_SelfAlias(t *testing.T) {
	a := NewAliasMap()
	// Self-referencing alias: ls -> ls --color=auto
	a.Set("ls", "ls --color=auto")

	got := a.Expand("ls /tmp")
	want := "ls --color=auto /tmp"
	if got != want {
		t.Errorf("Expand(\"ls /tmp\") = %q, want %q", got, want)
	}

	// After first expansion, "ls" is in seen, so it won't expand again.
}

func TestAliasMap_Overwrite(t *testing.T) {
	a := NewAliasMap()
	a.Set("ll", "ls -la")
	a.Set("ll", "ls -lah")

	val, ok := a.Get("ll")
	if !ok {
		t.Fatal("alias 'll' not found after overwrite")
	}
	if val != "ls -lah" {
		t.Errorf("Get(\"ll\") = %q, want %q", val, "ls -lah")
	}
}
