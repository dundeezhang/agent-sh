package shell

import (
	"os"
	"sort"
	"strings"
)

// Completer returns completions for a command given the arguments parsed so far
// and the partial word currently being typed.
type Completer func(args []string, currentArg string) []string

// CompletionRegistry maps command names to their Completer functions.
type CompletionRegistry struct {
	completers map[string]Completer
}

// NewCompletionRegistry creates a registry pre-loaded with built-in completers.
func NewCompletionRegistry() *CompletionRegistry {
	r := &CompletionRegistry{
		completers: make(map[string]Completer),
	}
	r.registerDefaults()
	return r
}

// Register adds (or replaces) a completer for the given command name.
func (r *CompletionRegistry) Register(command string, c Completer) {
	r.completers[command] = c
}

// Lookup returns the completer for a command, or nil if none is registered.
func (r *CompletionRegistry) Lookup(command string) Completer {
	return r.completers[command]
}

// registerDefaults wires up completers for well-known commands.
func (r *CompletionRegistry) registerDefaults() {
	r.Register("git", completeGit)
	r.Register("docker", completeDocker)
	r.Register("kubectl", completeKubectl)
	r.Register("k", completeKubectl)
}

// ---------------------------------------------------------------------------
// Environment variable completion
// ---------------------------------------------------------------------------

// completeEnvVar returns variable names matching the prefix (with $).
// The prefix is expected to start with "$".
func completeEnvVar(prefix string) []string {
	if !strings.HasPrefix(prefix, "$") {
		return nil
	}
	namePrefix := prefix[1:] // strip the leading $

	var matches []string
	for _, env := range os.Environ() {
		name, _, _ := strings.Cut(env, "=")
		if strings.HasPrefix(name, namePrefix) {
			matches = append(matches, "$"+name)
		}
	}
	sort.Strings(matches)
	return matches
}

// ---------------------------------------------------------------------------
// Git completer
// ---------------------------------------------------------------------------

var gitSubcommands = []string{
	"add", "bisect", "branch", "checkout", "cherry-pick", "clone",
	"commit", "config", "diff", "fetch", "init", "log", "merge",
	"mv", "pull", "push", "rebase", "remote", "reset", "restore",
	"rm", "show", "stash", "status", "switch", "tag",
}

var gitSubcommandFlags = map[string][]string{
	"add":    {"--all", "--dry-run", "--force", "--patch", "--verbose"},
	"branch": {"--all", "--delete", "--list", "--move", "--remotes", "--verbose"},
	"checkout": {
		"--branch", "--detach", "--force", "--merge", "--quiet", "--track",
	},
	"commit": {
		"--all", "--amend", "--message", "--no-edit", "--signoff", "--verbose",
	},
	"diff": {
		"--cached", "--name-only", "--name-status", "--stat", "--staged",
	},
	"log": {
		"--all", "--graph", "--oneline", "--patch", "--stat",
	},
	"merge":  {"--abort", "--continue", "--no-commit", "--no-ff", "--squash"},
	"pull":   {"--rebase", "--no-rebase", "--ff-only", "--verbose"},
	"push":   {"--all", "--delete", "--force", "--set-upstream", "--tags", "--verbose"},
	"rebase": {"--abort", "--continue", "--interactive", "--onto", "--skip"},
	"remote": {"--verbose"},
	"reset":  {"--hard", "--mixed", "--soft"},
	"stash":  {"apply", "clear", "drop", "list", "pop", "push", "show"},
	"tag":    {"--annotate", "--delete", "--force", "--list", "--message"},
}

func completeGit(args []string, currentArg string) []string {
	// args contains tokens after "git" and before the current partial word.
	// Example: "git commit --am" -> args=["commit"], currentArg="--am"

	if len(args) == 0 {
		// Completing the subcommand.
		return filterPrefix(gitSubcommands, currentArg)
	}

	subcmd := args[0]

	// If the current argument starts with "-", complete flags for the subcommand.
	if strings.HasPrefix(currentArg, "-") {
		if flags, ok := gitSubcommandFlags[subcmd]; ok {
			return filterPrefix(flags, currentArg)
		}
		return nil
	}

	// For "git stash", stash sub-subcommands are stored as flags list.
	if subcmd == "stash" && len(args) == 1 {
		if subs, ok := gitSubcommandFlags["stash"]; ok {
			return filterPrefix(subs, currentArg)
		}
	}

	// Fall back to file completion for everything else.
	return nil
}

// ---------------------------------------------------------------------------
// Docker completer
// ---------------------------------------------------------------------------

var dockerSubcommands = []string{
	"attach", "build", "compose", "container", "cp", "create", "exec",
	"image", "images", "info", "inspect", "kill", "logs", "network",
	"pause", "port", "ps", "pull", "push", "rename", "restart", "rm",
	"rmi", "run", "start", "stop", "system", "tag", "top", "unpause",
	"volume",
}

func completeDocker(args []string, currentArg string) []string {
	if len(args) == 0 {
		return filterPrefix(dockerSubcommands, currentArg)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Kubectl completer
// ---------------------------------------------------------------------------

var kubectlSubcommands = []string{
	"annotate", "api-resources", "apply", "attach", "auth", "autoscale",
	"cluster-info", "config", "cordon", "cp", "create", "delete",
	"describe", "diff", "drain", "edit", "exec", "explain", "expose",
	"get", "label", "logs", "patch", "plugin", "port-forward", "proxy",
	"replace", "rollout", "run", "scale", "set", "taint", "top",
	"uncordon", "version", "wait",
}

func completeKubectl(args []string, currentArg string) []string {
	if len(args) == 0 {
		return filterPrefix(kubectlSubcommands, currentArg)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// filterPrefix returns elements of candidates that start with prefix,
// preserving the original order. An empty prefix matches everything.
func filterPrefix(candidates []string, prefix string) []string {
	var out []string
	for _, c := range candidates {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}
