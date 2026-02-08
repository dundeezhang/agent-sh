package shell

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// builtins that the shell handles directly.
var shellBuiltins = []string{"cd", "env", "exit", "export", "history"}

// completeWord extracts the word at the cursor and returns possible completions.
// When the cursor is on the first token and it contains no path separator,
// it completes command names (PATH executables + builtins). Otherwise it
// completes file/directory names.
func completeWord(line string, pos int) (wordStart int, prefix string, matches []string) {
	// Find start of the current word (scan backwards from cursor for whitespace).
	left := line[:pos]
	wordStart = strings.LastIndexAny(left, " \t") + 1 // 0 if no space found
	prefix = left[wordStart:]

	// Determine if this is the command (first) word.
	isCommand := strings.TrimSpace(left[:wordStart]) == ""

	// If it's the command word and doesn't contain a path separator, complete commands.
	if isCommand && !strings.Contains(prefix, "/") && !strings.HasPrefix(prefix, "~") {
		matches = completeCommand(prefix)
		return wordStart, prefix, matches
	}

	matches = completeFile(prefix)
	return wordStart, prefix, matches
}

// completeCommand returns executables from $PATH and builtins matching prefix.
func completeCommand(prefix string) []string {
	if prefix == "" {
		return nil
	}

	seen := make(map[string]bool)
	var matches []string

	// Builtins first.
	for _, b := range shellBuiltins {
		if strings.HasPrefix(b, prefix) {
			seen[b] = true
			matches = append(matches, b)
		}
	}

	// Walk each directory in $PATH.
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		sort.Strings(matches)
		return matches
	}
	for _, dir := range filepath.SplitList(pathEnv) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if seen[name] || !strings.HasPrefix(name, prefix) {
				continue
			}
			typ := e.Type()
			if typ.IsDir() {
				continue
			}
			// Symlinks in PATH are almost certainly executables; skip the stat.
			if typ&os.ModeSymlink != 0 {
				seen[name] = true
				matches = append(matches, name)
				continue
			}
			// Regular file — check execute bit.
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0111 != 0 {
				seen[name] = true
				matches = append(matches, name)
			}
		}
	}

	sort.Strings(matches)
	return matches
}

// completeFile returns file/directory names matching prefix.
func completeFile(prefix string) []string {
	// Expand ~ at the start of the prefix for filesystem operations.
	expanded := prefix
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(expanded, "~") {
		expanded = home + expanded[1:]
	}

	// Extract the directory prefix from the original input by finding the
	// last path separator, preserving prefixes like ./, ../, /path/to/.
	var dirPrefix string
	var dir string
	var partial string

	if prefix == "" {
		dir = "."
		partial = ""
		dirPrefix = ""
	} else if strings.HasSuffix(prefix, "/") {
		dir = expanded
		partial = ""
		dirPrefix = prefix
	} else if idx := strings.LastIndex(prefix, "/"); idx >= 0 {
		dirPrefix = prefix[:idx+1]
		dir = filepath.Dir(expanded)
		partial = prefix[idx+1:]
	} else {
		dir = "."
		partial = prefix
		dirPrefix = ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var matches []string
	for _, e := range entries {
		name := e.Name()
		// Skip hidden files unless the user typed a dot prefix.
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(partial, ".") {
			continue
		}
		if strings.HasPrefix(name, partial) {
			display := dirPrefix + name
			if e.IsDir() {
				display += "/"
			}
			matches = append(matches, display)
		}
	}

	sort.Strings(matches)
	return matches
}

// commonPrefix returns the longest common prefix of all strings in matches.
func commonPrefix(matches []string) string {
	if len(matches) == 0 {
		return ""
	}
	cp := matches[0]
	for _, m := range matches[1:] {
		for i := range cp {
			if i >= len(m) || cp[i] != m[i] {
				cp = cp[:i]
				break
			}
		}
	}
	return cp
}

// formatColumns arranges names into columns fitting the given terminal width,
// similar to ls output.
func formatColumns(matches []string, width int) string {
	if len(matches) == 0 {
		return ""
	}

	// Find the widest entry.
	maxLen := 0
	for _, m := range matches {
		if len(m) > maxLen {
			maxLen = len(m)
		}
	}

	colWidth := maxLen + 2 // 2-space gap between columns
	if colWidth > width {
		colWidth = width
	}
	cols := width / colWidth
	if cols < 1 {
		cols = 1
	}

	var b strings.Builder
	for i, m := range matches {
		if i > 0 && i%cols == 0 {
			b.WriteByte('\n')
		}
		// Pad to column width (except for last column in a row).
		if (i+1)%cols == 0 || i == len(matches)-1 {
			b.WriteString(m)
		} else {
			b.WriteString(m)
			for j := len(m); j < colWidth; j++ {
				b.WriteByte(' ')
			}
		}
	}
	b.WriteByte('\n')
	return b.String()
}
