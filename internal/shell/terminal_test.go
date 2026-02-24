package shell

import (
	"fmt"
	"os"
	"os/user"
	"strings"
	"testing"
	"time"
)

func TestFormatPrompt_DirectoryExpansion(t *testing.T) {
	result := formatPrompt("%d", 0)
	cwd, _ := os.Getwd()
	expected := abbreviateHome(cwd)
	if result != expected {
		t.Errorf("%%d: got %q, want %q", result, expected)
	}
}

func TestFormatPrompt_Username(t *testing.T) {
	result := formatPrompt("%u", 0)
	u, err := user.Current()
	if err != nil {
		t.Skip("cannot get current user")
	}
	if result != u.Username {
		t.Errorf("%%u: got %q, want %q", result, u.Username)
	}
}

func TestFormatPrompt_Hostname(t *testing.T) {
	result := formatPrompt("%h", 0)
	h, err := os.Hostname()
	if err != nil {
		t.Skip("cannot get hostname")
	}
	if result != h {
		t.Errorf("%%h: got %q, want %q", result, h)
	}
}

func TestFormatPrompt_Time(t *testing.T) {
	before := time.Now().Format("15:04")
	result := formatPrompt("%t", 0)
	// Just check it starts with the current HH:MM (seconds may tick over).
	if !strings.HasPrefix(result, before) {
		t.Errorf("%%t: got %q, expected prefix %q", result, before)
	}
}

func TestFormatPrompt_ExitCode(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{127, "127"},
		{255, "255"},
	}
	for _, tc := range tests {
		result := formatPrompt("%?", tc.code)
		if result != tc.want {
			t.Errorf("%%? with code %d: got %q, want %q", tc.code, result, tc.want)
		}
	}
}

func TestFormatPrompt_LiteralPercent(t *testing.T) {
	result := formatPrompt("100%%", 0)
	if result != "100%" {
		t.Errorf("%%%% escape: got %q, want %q", result, "100%")
	}
}

func TestFormatPrompt_UnknownSequence(t *testing.T) {
	result := formatPrompt("%z", 0)
	if result != "%z" {
		t.Errorf("unknown sequence: got %q, want %q", result, "%z")
	}
}

func TestFormatPrompt_PlainText(t *testing.T) {
	result := formatPrompt("hello world", 0)
	if result != "hello world" {
		t.Errorf("plain text: got %q, want %q", result, "hello world")
	}
}

func TestFormatPrompt_MixedSequences(t *testing.T) {
	result := formatPrompt("[%?] %d> ", 42)
	cwd, _ := os.Getwd()
	expected := fmt.Sprintf("[42] %s> ", abbreviateHome(cwd))
	if result != expected {
		t.Errorf("mixed: got %q, want %q", result, expected)
	}
}

func TestFormatPrompt_TrailingPercent(t *testing.T) {
	// A trailing '%' with no following character should be preserved.
	result := formatPrompt("test%", 0)
	if result != "test%" {
		t.Errorf("trailing %%: got %q, want %q", result, "test%")
	}
}

func TestFormatPrompt_DefaultFormat(t *testing.T) {
	// Verify the default format produces output that contains "agent-sh"
	// and the current directory, matching the original hardcoded prompt.
	defaultFmt := "\033[1;34magent-sh\033[0m %d\033[1;34m>\033[0m "
	result := formatPrompt(defaultFmt, 0)
	if !strings.Contains(result, "agent-sh") {
		t.Errorf("default format should contain 'agent-sh', got %q", result)
	}
	cwd, _ := os.Getwd()
	abbrev := abbreviateHome(cwd)
	if !strings.Contains(result, abbrev) {
		t.Errorf("default format should contain CWD %q, got %q", abbrev, result)
	}
}

func TestAbbreviateHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}
	tests := []struct {
		input string
		want  string
	}{
		{home, "~"},
		{home + "/foo/bar", "~/foo/bar"},
		{"/tmp/other", "/tmp/other"},
	}
	for _, tc := range tests {
		got := abbreviateHome(tc.input)
		if got != tc.want {
			t.Errorf("abbreviateHome(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
