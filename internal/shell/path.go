package shell

import (
	"os"
	"strings"
)

// expandHome resolves a leading ~ or ~/ in p to the user's home directory.
// "" and "~" both expand to the home directory itself. Returns p unchanged
// when there is no tilde prefix or the home directory cannot be determined.
func expandHome(p string) string {
	if p != "" && p != "~" && !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == "" || p == "~" {
		return home
	}
	return home + p[1:]
}

// collapseHome is the inverse of expandHome: replaces a leading home path
// with "~" for display.
func collapseHome(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || !strings.HasPrefix(p, home) {
		return p
	}
	return "~" + p[len(home):]
}
