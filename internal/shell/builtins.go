package shell

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// builtinNames lists every command that handleBuiltin recognises.
// Keep sorted for readability; order does not affect behaviour.
var builtinNames = map[string]bool{
	".":       true,
	"cd":      true,
	"env":     true,
	"exit":    true,
	"export":  true,
	"history": true,
	"pwd":     true,
	"source":  true,
	"type":    true,
	"unset":   true,
	"which":   true,
}

// handleBuiltin returns true if the line was a builtin command.
func (s *Shell) handleBuiltin(line string, t *term.Terminal) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}

	switch parts[0] {
	case "exit":
		s.restore()
		os.Exit(0)
		return true

	case "cd":
		dir := ""
		if len(parts) > 1 {
			dir = parts[1]
		}
		if err := os.Chdir(expandHome(dir)); err != nil {
			fmt.Fprintf(t, "cd: %s\r\n", err)
		}
		t.SetPrompt(prompt())
		return true

	case "export":
		if len(parts) < 2 {
			return true
		}
		for _, arg := range parts[1:] {
			if idx := strings.Index(arg, "="); idx >= 0 {
				os.Setenv(arg[:idx], arg[idx+1:])
			}
		}
		return true

	case "env":
		for _, e := range os.Environ() {
			fmt.Fprintf(t, "%s\r\n", e)
		}
		return true

	case "history":
		for i, cmd := range s.history.Recent(0) {
			fmt.Fprintf(t, "%4d  %s\r\n", i+1, cmd)
		}
		return true

	case "pwd":
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(t, "pwd: %s\r\n", err)
		} else {
			fmt.Fprintf(t, "%s\r\n", dir)
		}
		return true

	case "unset":
		for _, name := range parts[1:] {
			if err := os.Unsetenv(name); err != nil {
				fmt.Fprintf(t, "unset: %s\r\n", err)
			}
		}
		return true

	case "type":
		for _, name := range parts[1:] {
			if builtinNames[name] {
				fmt.Fprintf(t, "%s is a shell builtin\r\n", name)
			} else if path, err := exec.LookPath(name); err == nil {
				fmt.Fprintf(t, "%s is %s\r\n", name, path)
			} else {
				fmt.Fprintf(t, "type: %s: not found\r\n", name)
			}
		}
		return true

	case "which":
		for _, name := range parts[1:] {
			if path, err := exec.LookPath(name); err == nil {
				fmt.Fprintf(t, "%s\r\n", path)
			} else {
				fmt.Fprintf(t, "which: %s: not found\r\n", name)
			}
		}
		return true

	case "source", ".":
		if len(parts) < 2 {
			fmt.Fprintf(t, "%s: filename argument required\r\n", parts[0])
			return true
		}
		s.sourceFile(parts[1], t)
		return true
	}

	return false
}

// sourceFile reads a file and executes each line in the current shell context.
func (s *Shell) sourceFile(path string, t *term.Terminal) {
	// Expand ~ at the start.
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		if home != "" {
			path = home + path[1:]
		}
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(t, "source: %s\r\n", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Try as a builtin first; otherwise execute as an external command.
		if !s.handleBuiltin(line, t) {
			s.restore()
			s.execCommand(line)
			s.rawMode()
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(t, "source: read error: %s\r\n", err)
	}
}
