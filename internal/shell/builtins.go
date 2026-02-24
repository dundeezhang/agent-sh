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
		if dir == "" || dir == "~" {
			dir, _ = os.UserHomeDir()
		} else if strings.HasPrefix(dir, "~/") {
			home, _ := os.UserHomeDir()
			dir = home + dir[1:]
		}
		if err := os.Chdir(dir); err != nil {
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

	case "alias":
		if len(parts) == 1 {
			// No args: list all aliases.
			for _, entry := range s.aliases.List() {
				fmt.Fprintf(t, "alias %s='%s'\r\n", entry.Name, entry.Value)
			}
			return true
		}
		// Parse each alias definition: name='value' or name=value
		for _, arg := range parts[1:] {
			idx := strings.Index(arg, "=")
			if idx < 0 {
				// Show a single alias.
				val, ok := s.aliases.Get(arg)
				if ok {
					fmt.Fprintf(t, "alias %s='%s'\r\n", arg, val)
				} else {
					fmt.Fprintf(t, "alias: %s: not found\r\n", arg)
				}
				continue
			}
			name := arg[:idx]
			value := arg[idx+1:]
			// Strip surrounding quotes if present.
			if len(value) >= 2 {
				if (value[0] == '\'' && value[len(value)-1] == '\'') ||
					(value[0] == '"' && value[len(value)-1] == '"') {
					value = value[1 : len(value)-1]
				}
			}
			if name == "" {
				fmt.Fprintf(t, "alias: invalid alias name\r\n")
				continue
			}
			s.aliases.Set(name, value)
		}
		return true

	case "unalias":
		if len(parts) < 2 {
			fmt.Fprintf(t, "unalias: usage: unalias name [name ...]\r\n")
			return true
		}
		for _, name := range parts[1:] {
			if !s.aliases.Remove(name) {
				fmt.Fprintf(t, "unalias: %s: not found\r\n", name)
			}
		}
		return true
	}

	return false
}
