package shell

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// prompt returns the shell prompt string showing CWD and, when non-zero,
// the last command's exit code in red.
func prompt(exitCode int) string {
	dir, err := os.Getwd()
	if err != nil {
		dir = "?"
	}
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(dir, home) {
		dir = "~" + dir[len(home):]
	}
	exitIndicator := ""
	if exitCode != 0 {
		exitIndicator = fmt.Sprintf(" \033[1;31m[%d]\033[0m", exitCode)
	}
	return fmt.Sprintf("\033[1;34magent-sh\033[0m %s%s\033[1;34m>\033[0m ", dir, exitIndicator)
}

// termSize returns the current terminal width and height.
func termSize() (int, int) {
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil || w <= 0 {
		w = 80
	}
	if err != nil || h <= 0 {
		h = 24
	}
	return w, h
}
