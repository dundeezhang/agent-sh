package shell

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"golang.org/x/term"
)

// abbreviateHome replaces the home directory prefix with ~.
func abbreviateHome(dir string) string {
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(dir, home) {
		return "~" + dir[len(home):]
	}
	return dir
}

// gitBranch returns the current git branch name, or an empty string
// if not in a git repository or on error.
func gitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// formatPrompt expands escape sequences in the prompt format string.
//
// Supported sequences:
//
//	%d  current working directory (home abbreviated as ~)
//	%u  current username
//	%h  hostname
//	%t  current time (HH:MM:SS)
//	%?  last command exit code
//	%g  current git branch
//	%%  literal percent sign
func formatPrompt(format string, lastExitCode int) string {
	var b strings.Builder
	b.Grow(len(format))

	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 >= len(format) {
			b.WriteByte(format[i])
			continue
		}

		// We have a '%' and there's at least one more character.
		i++
		switch format[i] {
		case 'd':
			dir, err := os.Getwd()
			if err != nil {
				dir = "?"
			}
			b.WriteString(abbreviateHome(dir))
		case 'u':
			if u, err := user.Current(); err == nil {
				b.WriteString(u.Username)
			}
		case 'h':
			if h, err := os.Hostname(); err == nil {
				b.WriteString(h)
			}
		case 't':
			b.WriteString(time.Now().Format("15:04:05"))
		case '?':
			b.WriteString(fmt.Sprintf("%d", lastExitCode))
		case 'g':
			b.WriteString(gitBranch())
		case '%':
			b.WriteByte('%')
		default:
			// Unknown sequence: preserve as-is.
			b.WriteByte('%')
			b.WriteByte(format[i])
		}
	}

	return b.String()
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
