package shell

import (
	"os/exec"
	"strings"
)

// InputClass represents the classification of user input.
type InputClass int

const (
	// ClassCommand means the input should be executed as a shell command.
	ClassCommand InputClass = iota
	// ClassAgent means the input should be handled by the AI agent.
	ClassAgent
	// ClassUnsure means the classifier cannot confidently decide;
	// the shell should ask the user.
	ClassUnsure
)

// String returns the name of the InputClass constant.
func (c InputClass) String() string {
	switch c {
	case ClassCommand:
		return "ClassCommand"
	case ClassAgent:
		return "ClassAgent"
	case ClassUnsure:
		return "ClassUnsure"
	default:
		return "???"
	}
}

// functionWords are words that almost never appear as bare command arguments.
// If any appears among 2+ remaining args after a known command, the input
// is likely natural language.
var functionWords = map[string]bool{
	"the": true, "a": true, "an": true, "this": true, "that": true,
	"my": true, "your": true, "its": true, "our": true, "their": true,
	"it": true, "me": true, "i": true, "you": true, "we": true, "they": true,
	"in": true, "on": true, "at": true, "for": true, "to": true,
	"from": true, "with": true, "of": true, "about": true,
	"all": true, "not": true, "and": true, "or": true, "but": true,
	"is": true, "are": true, "was": true, "do": true, "does": true,
	"can": true, "will": true, "would": true, "should": true,
	"how": true, "what": true, "why": true, "where": true, "when": true,
	"who": true, "which": true,
	"please": true, "help": true,
}

// shellKeywords are shell language keywords and builtins that indicate
// the input is a command, not natural language.
var shellKeywords = map[string]bool{
	"echo": true, "printf": true, "test": true, "true": true, "false": true,
	"for": true, "while": true, "until": true, "if": true, "then": true,
	"else": true, "elif": true, "fi": true, "do": true, "done": true,
	"case": true, "esac": true, "select": true,
	"source": true, ".": true,
	"alias": true, "unalias": true,
	"set": true, "unset": true, "shift": true,
	"read": true, "eval": true, "exec": true,
	"trap": true, "wait": true, "kill": true,
	"type": true, "which": true, "command": true,
	"declare": true, "local": true, "readonly": true,
	"return": true, "break": true, "continue": true,
	"pushd": true, "popd": true, "dirs": true,
	"time": true, "times": true,
	"ulimit": true, "umask": true,
	"bg": true, "fg": true, "jobs": true,
	"let": true, "getopts": true,
	"sudo": true,
}

// classifyInput decides whether a line of input is a shell command or
// natural language that should be sent to the agent.
func classifyInput(line string) InputClass {
	// Shell operators → command.
	if containsShellOperator(line) {
		return ClassCommand
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ClassCommand
	}

	first := parts[0]

	// First word is a path → command.
	if strings.HasPrefix(first, "/") || strings.HasPrefix(first, "./") ||
		strings.HasPrefix(first, "~/") || first == "~" {
		return ClassCommand
	}

	// First word contains "=" → command (variable assignment).
	if strings.Contains(first, "=") {
		return ClassCommand
	}

	// First word is a shell builtin/keyword.
	// echo/printf take arbitrary text — always command.
	// Others get the same NL check as PATH commands.
	if isShellBuiltin(first) {
		if first == "echo" || first == "printf" {
			return ClassCommand
		}
		args := parts[1:]
		if len(args) == 0 {
			return ClassCommand
		}
		if argsHaveShellPatterns(args) {
			return ClassCommand
		}
		if len(args) == 1 {
			return ClassCommand
		}
		if argsLookLikeSentence(args) {
			return ClassAgent
		}
		return ClassCommand
	}

	// First word found in PATH.
	if _, err := exec.LookPath(first); err == nil {
		args := parts[1:]
		if len(args) == 0 {
			return ClassCommand
		}
		// Has flags, paths, or globs → command.
		if argsHaveShellPatterns(args) {
			return ClassCommand
		}
		// Single arg → command (e.g. "make clean", "git status").
		if len(args) == 1 {
			return ClassCommand
		}
		// 2+ args: if any is a function word → agent.
		if argsLookLikeSentence(args) {
			return ClassAgent
		}
		return ClassCommand
	}

	// First word NOT in PATH.
	if len(parts) == 1 {
		// Single unknown word — could be a typo or a vague request.
		return ClassUnsure
	}

	args := parts[1:]

	// Args with flags, paths, or globs → probably a command (e.g. "gti -v").
	if argsHaveShellPatterns(args) {
		return ClassCommand
	}

	// Function words → clearly natural language.
	if argsLookLikeSentence(args) {
		return ClassAgent
	}

	// No clear signals → unsure, ask the user.
	return ClassUnsure
}

// containsShellOperator returns true if the line contains unquoted shell
// operators: |, >, >>, <, &&, ||, ;, $(), or backticks.
func containsShellOperator(line string) bool {
	inSingle := false
	inDouble := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case inSingle || inDouble:
			continue
		case ch == '|' || ch == '>' || ch == '<' || ch == ';' || ch == '`':
			return true
		case ch == '&' && i+1 < len(line) && line[i+1] == '&':
			return true
		case ch == '$' && i+1 < len(line) && line[i+1] == '(':
			return true
		}
	}
	return false
}

// isShellBuiltin returns true if word is a known shell builtin or keyword.
func isShellBuiltin(word string) bool {
	return shellKeywords[word]
}

// argsHaveShellPatterns returns true if any arg looks like a flag, path, or glob.
func argsHaveShellPatterns(args []string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return true
		}
		if strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, "./") || strings.HasPrefix(arg, "~/") {
			return true
		}
		if strings.ContainsAny(arg, "*?[") {
			return true
		}
	}
	return false
}

// argsLookLikeSentence returns true if any of the args is a common English
// function word, suggesting the input is natural language rather than a command.
func argsLookLikeSentence(args []string) bool {
	for _, arg := range args {
		if functionWords[strings.ToLower(arg)] {
			return true
		}
	}
	return false
}
