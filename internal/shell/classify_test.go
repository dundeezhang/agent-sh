package shell

import "testing"

func className(c InputClass) string {
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

func TestClassifyInput(t *testing.T) {
	tests := []struct {
		input string
		want  InputClass
	}{
		// Commands: executable in PATH with flags/paths/globs
		{"ls -la", ClassCommand},
		{"find . -name \"*.py\"", ClassCommand},
		{"make -j4 build", ClassCommand},
		{"git status", ClassCommand},
		{"git push origin main", ClassCommand},
		{"make clean", ClassCommand},

		// Commands: shell operators
		{"cat foo | grep bar", ClassCommand},
		{"echo hello > file.txt", ClassCommand},
		{"ls && pwd", ClassCommand},
		{"echo $(date)", ClassCommand},
		{"echo hello; echo world", ClassCommand},

		// Commands: first word is a path
		{"./script.sh", ClassCommand},
		{"/usr/bin/env python", ClassCommand},
		{"~/bin/tool", ClassCommand},

		// Commands: variable assignment
		{"FOO=bar", ClassCommand},

		// Commands: shell builtins
		{"echo hello world", ClassCommand},
		{"for i in 1 2 3", ClassCommand},
		{"if true", ClassCommand},
		{"source ~/.bashrc", ClassCommand},
		{"sudo apt update", ClassCommand},

		// Commands: unknown first word but args have shell patterns
		{"gti -v", ClassCommand},
		{"terraform ./main.tf", ClassCommand},

		// Agent: clear natural language (function words present)
		{"what files are here", ClassAgent},
		{"refactor the login page", ClassAgent},
		{"make this work", ClassAgent},
		{"find all python files", ClassAgent},
		{"explain this to me", ClassAgent},

		// Unsure: single unknown word — could be typo or vague request
		{"gti", ClassUnsure},
		{"asdfgh", ClassUnsure},
		{"facts", ClassUnsure},

		// Unsure: unknown first word, 2+ words, no clear signals
		{"gti status", ClassUnsure},
		{"terraform init", ClassUnsure},
		{"hello world", ClassUnsure},
		{"create project", ClassUnsure},
		{"describe structure", ClassUnsure},

		// Agent: empty → ClassCommand (edge case, handled by caller)
		{"", ClassCommand},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := classifyInput(tt.input)
			if got != tt.want {
				t.Errorf("classifyInput(%q) = %s, want %s", tt.input, className(got), className(tt.want))
			}
		})
	}
}

func TestContainsShellOperator(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"ls | grep foo", true},
		{"echo hello > file", true},
		{"ls && pwd", true},
		{"echo $(date)", true},
		{"echo hello; echo world", true},
		{"echo `date`", true},
		{"cat < file", true},

		// Operators inside quotes should be ignored
		{"echo 'hello | world'", false},
		{"echo \"hello | world\"", false},

		// No operators
		{"ls -la", false},
		{"git status", false},
		{"make clean", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := containsShellOperator(tt.input); got != tt.want {
				t.Errorf("containsShellOperator(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsShellBuiltin(t *testing.T) {
	builtins := []string{"echo", "for", "if", "source", "sudo", "eval", "read"}
	for _, b := range builtins {
		if !isShellBuiltin(b) {
			t.Errorf("isShellBuiltin(%q) = false, want true", b)
		}
	}

	nonBuiltins := []string{"ls", "git", "make", "python", "hello"}
	for _, nb := range nonBuiltins {
		if isShellBuiltin(nb) {
			t.Errorf("isShellBuiltin(%q) = true, want false", nb)
		}
	}
}

func TestArgsLookLikeSentence(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"this", "work"}, true},
		{[]string{"all", "python", "files"}, true},
		{[]string{"the", "login", "page"}, true},
		{[]string{"clean"}, false},
		{[]string{"push", "origin", "main"}, false},
		{[]string{"-la"}, false},
		{[]string{"status"}, false},
	}

	for _, tt := range tests {
		name := ""
		for i, a := range tt.args {
			if i > 0 {
				name += " "
			}
			name += a
		}
		t.Run(name, func(t *testing.T) {
			if got := argsLookLikeSentence(tt.args); got != tt.want {
				t.Errorf("argsLookLikeSentence(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestArgsHaveShellPatterns(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"-la"}, true},
		{[]string{"./foo"}, true},
		{[]string{"/usr/bin"}, true},
		{[]string{"~/bar"}, true},
		{[]string{"*.go"}, true},
		{[]string{"clean"}, false},
		{[]string{"push", "origin", "main"}, false},
	}

	for _, tt := range tests {
		name := ""
		for i, a := range tt.args {
			if i > 0 {
				name += " "
			}
			name += a
		}
		t.Run(name, func(t *testing.T) {
			if got := argsHaveShellPatterns(tt.args); got != tt.want {
				t.Errorf("argsHaveShellPatterns(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
