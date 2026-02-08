package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// readOnlyCommands are bash commands that only read state and never modify it.
var readOnlyCommands = map[string]bool{
	"cat": true, "head": true, "tail": true, "less": true, "more": true,
	"ls": true, "dir": true, "tree": true, "file": true, "stat": true,
	"wc": true, "du": true, "df": true,
	"grep": true, "rg": true, "ag": true, "ack": true,
	// find omitted: -delete and -exec can be destructive.
	"fd": true, "locate": true, "which": true, "whereis": true,
	"pwd": true, "whoami": true, "id": true, "hostname": true, "uname": true,
	"date": true, "uptime": true, "env": true, "printenv": true,
	"echo": true, "printf": true,
	"diff": true, "cmp": true, "md5sum": true, "shasum": true,
	"git": true, "go": true,
	"jq": true, "yq": true, "xmllint": true,
	"man": true, "help": true, "type": true,
	"ps": true, "top": true, "htop": true, "pgrep": true,
	"lsof": true, "netstat": true, "ss": true,
}

// readOnlyGitSubcommands are git subcommands that only read state.
var readOnlyGitSubcommands = map[string]bool{
	// branch, tag, remote omitted: can create/delete with args.
	"status": true, "log": true, "diff": true, "show": true,
	"blame": true, "shortlog": true,
	"describe": true, "rev-parse": true, "ls-files": true, "ls-tree": true,
	"cat-file": true, "reflog": true,
}

// mutatingGoSubcommands are go subcommands that modify state.
var mutatingGoSubcommands = map[string]bool{
	"install": true, "get": true, "generate": true, "clean": true,
	"mod": true, "fmt": true,
}

// isReadOnlyBash returns true if a bash command is safe to run without
// user confirmation (read-only, no side effects).
func isReadOnlyBash(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}

	// Reject commands with unquoted redirections or command substitutions.
	if hasUnsafeRedirects(cmd) {
		return false
	}

	// Strip leading env vars (FOO=bar cmd ...) and sudo.
	i := 0
	for i < len(parts) && strings.Contains(parts[i], "=") {
		i++
	}
	if i >= len(parts) {
		return false
	}
	if parts[i] == "sudo" {
		i++
		if i >= len(parts) {
			return false
		}
	}

	base := parts[i]

	// Pipelines/chains: every command in the pipeline must be read-only.
	// Only recurse when there are actually multiple segments to avoid
	// infinite recursion on quoted operators or a lone '&'.
	if strings.ContainsAny(cmd, "|;&") {
		segments := splitShellSegments(cmd)
		if len(segments) > 1 {
			for _, seg := range segments {
				if !isReadOnlyBash(seg) {
					return false
				}
			}
			return true
		}
	}

	if !readOnlyCommands[base] {
		return false
	}

	// git: check subcommand (skip leading flags like -C, --no-pager, etc.).
	if base == "git" {
		sub := ""
		for j := i + 1; j < len(parts); j++ {
			if !strings.HasPrefix(parts[j], "-") {
				sub = parts[j]
				break
			}
		}
		if sub == "" {
			return true // bare "git" with only flags is read-only
		}
		return readOnlyGitSubcommands[sub]
	}

	// go: only read-only subcommands (skip leading flags).
	if base == "go" {
		sub := ""
		for j := i + 1; j < len(parts); j++ {
			if !strings.HasPrefix(parts[j], "-") {
				sub = parts[j]
				break
			}
		}
		if sub == "" {
			return true // bare "go" with only flags is read-only
		}
		return !mutatingGoSubcommands[sub]
	}

	return true
}

// splitShellSegments splits a command on unquoted |, &&, ||, ; operators.
func splitShellSegments(cmd string) []string {
	var segments []string
	var current strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
			current.WriteByte(ch)
		case ch == '"' && !inSingle:
			inDouble = !inDouble
			current.WriteByte(ch)
		case inSingle || inDouble:
			current.WriteByte(ch)
		case ch == '|' || ch == ';':
			segments = append(segments, current.String())
			current.Reset()
			// Skip || or |
			if ch == '|' && i+1 < len(cmd) && cmd[i+1] == '|' {
				i++
			}
		case ch == '&':
			if i+1 < len(cmd) && cmd[i+1] == '&' {
				segments = append(segments, current.String())
				current.Reset()
				i++
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		segments = append(segments, current.String())
	}
	return segments
}

// hasUnsafeRedirects returns true if cmd contains unquoted output/input
// redirections (>, >>, <), backticks, or $() command substitutions.
// Single quotes suppress everything. Double quotes suppress redirects
// but NOT backticks or $() (which are expanded inside double quotes).
func hasUnsafeRedirects(cmd string) bool {
	inSingle, inDouble := false, false
	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case inSingle:
			continue
		case ch == '`':
			return true
		case ch == '$' && i+1 < len(cmd) && cmd[i+1] == '(':
			return true
		case inDouble:
			continue
		case ch == '>' || ch == '<':
			return true
		}
	}
	return false
}

// confirmBash prompts the user to approve a bash command. Returns true if approved.
// Default is Yes (just press Enter). Typing "a" enables auto-approve for the session.
func (a *Agent) confirmBash(cmd string) bool {
	fmt.Fprintf(os.Stdout, "\033[1;33mRun:\033[0m %s\n", cmd)

	if a.autoApprove {
		return true
	}

	fmt.Fprintf(os.Stdout, "\033[2m[Y/n/a]\033[0m ")

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(strings.ToLower(line))

	switch answer {
	case "n", "no":
		return false
	case "a", "always":
		a.autoApprove = true
		return true
	default:
		// Enter or "y"/"yes" — approve
		return true
	}
}
