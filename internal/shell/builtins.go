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
		if dir == "-" {
			// Switch to previous directory.
			dir = s.dirStack.OldPWD()
			if dir == "" {
				fmt.Fprintf(t, "cd: OLDPWD not set\r\n")
				return true
			}
		} else if dir == "" || dir == "~" {
			dir, _ = os.UserHomeDir()
		} else if strings.HasPrefix(dir, "~/") {
			home, _ := os.UserHomeDir()
			dir = home + dir[1:]
		}
		prev, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			fmt.Fprintf(t, "cd: %s\r\n", err)
		} else {
			s.dirStack.SetOldPWD(prev)
			if len(parts) > 1 && parts[1] == "-" {
				// Print new directory when using cd -.
				cwd, _ := os.Getwd()
				fmt.Fprintf(t, "%s\r\n", cwd)
			}
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

	case "pushd":
		if len(parts) < 2 {
			// No argument: swap current dir with top of stack.
			if err := s.dirStack.Swap(); err != nil {
				fmt.Fprintf(t, "%s\r\n", err)
			} else {
				fmt.Fprintf(t, "%s\r\n", strings.Join(s.dirStack.List(), " "))
			}
		} else {
			dir := parts[1]
			if dir == "~" {
				dir, _ = os.UserHomeDir()
			} else if strings.HasPrefix(dir, "~/") {
				home, _ := os.UserHomeDir()
				dir = home + dir[1:]
			}
			if err := s.dirStack.Push(dir); err != nil {
				fmt.Fprintf(t, "%s\r\n", err)
			} else {
				fmt.Fprintf(t, "%s\r\n", strings.Join(s.dirStack.List(), " "))
			}
		}
		t.SetPrompt(prompt())
		return true

	case "popd":
		if err := s.dirStack.Pop(); err != nil {
			fmt.Fprintf(t, "%s\r\n", err)
		} else {
			fmt.Fprintf(t, "%s\r\n", strings.Join(s.dirStack.List(), " "))
		}
		t.SetPrompt(prompt())
		return true

	case "dirs":
		fmt.Fprintf(t, "%s\r\n", strings.Join(s.dirStack.List(), " "))
		return true
	}

	return false
}
