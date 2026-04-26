package shell

import (
	"os"
	"path/filepath"
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
// with "~" for display. Only collapses when p is exactly the home directory
// or a path inside it — never on a sibling path that merely shares a prefix
// (e.g. "/home/userfoo" must not become "~foo" when home is "/home/user").
func collapseHome(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	sep := string(filepath.Separator)
	if strings.HasPrefix(p, home+sep) {
		return "~" + p[len(home):]
	}
	return p
}
