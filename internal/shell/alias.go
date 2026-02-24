package shell

import (
	"sort"
	"strings"
	"sync"
)

// AliasMap is a thread-safe map of command aliases.
type AliasMap struct {
	mu      sync.RWMutex
	aliases map[string]string
}

// NewAliasMap creates a new empty AliasMap.
func NewAliasMap() *AliasMap {
	return &AliasMap{
		aliases: make(map[string]string),
	}
}

// Set defines an alias. name is the alias and value is the expansion.
func (a *AliasMap) Set(name, value string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.aliases[name] = value
}

// Get returns the expansion for an alias and whether it exists.
func (a *AliasMap) Get(name string) (string, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	v, ok := a.aliases[name]
	return v, ok
}

// Remove deletes an alias. It returns true if the alias existed.
func (a *AliasMap) Remove(name string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.aliases[name]
	if ok {
		delete(a.aliases, name)
	}
	return ok
}

// List returns a sorted copy of all defined aliases.
func (a *AliasMap) List() []struct{ Name, Value string } {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]struct{ Name, Value string }, 0, len(a.aliases))
	for k, v := range a.aliases {
		result = append(result, struct{ Name, Value string }{k, v})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Names returns a sorted slice of all alias names.
func (a *AliasMap) Names() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	names := make([]string, 0, len(a.aliases))
	for k := range a.aliases {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Expand performs alias expansion on the first word of a command line.
// It recursively expands aliases (up to a depth limit) with loop detection
// to avoid infinite expansion.
func (a *AliasMap) Expand(line string) string {
	const maxDepth = 10

	a.mu.RLock()
	defer a.mu.RUnlock()

	seen := make(map[string]bool)
	for range maxDepth {
		parts := strings.SplitN(line, " ", 2)
		first := parts[0]
		rest := ""
		if len(parts) > 1 {
			rest = parts[1]
		}

		if seen[first] {
			break
		}

		expansion, ok := a.aliases[first]
		if !ok {
			break
		}

		seen[first] = true
		if rest != "" {
			line = expansion + " " + rest
		} else {
			line = expansion
		}
	}

	return line
}
