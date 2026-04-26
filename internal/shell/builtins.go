package shell

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

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
	}

	return false
}
