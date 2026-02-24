package shell

import "testing"

func TestNeedsContinuation(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		reason ContinuationReason
	}{
		// No continuation needed.
		{"simple command", "ls -la", ContinuationNone},
		{"empty string", "", ContinuationNone},
		{"closed double quotes", `echo "hello world"`, ContinuationNone},
		{"closed single quotes", "echo 'hello world'", ContinuationNone},
		{"balanced parens", "echo $(date)", ContinuationNone},
		{"balanced braces", "echo ${HOME}", ContinuationNone},
		{"balanced brackets", "arr[0]=1", ContinuationNone},
		{"escaped backslash at end", `echo hello\\`, ContinuationNone},

		// Trailing backslash.
		{"trailing backslash", `echo hello \`, ContinuationBackslash},
		{"trailing backslash no space", `echo hello\`, ContinuationBackslash},
		{"trailing backslash with trailing spaces", "echo hello \\   ", ContinuationBackslash},
		{"triple backslash", `echo hello\\\`, ContinuationBackslash},

		// Unclosed quotes.
		{"unclosed double quote", `echo "hello`, ContinuationQuote},
		{"unclosed single quote", "echo 'hello", ContinuationQuote},
		{"escaped quote inside double", `echo "hello \"world`, ContinuationQuote},
		{"unclosed double after closed single", "echo 'done' \"not done", ContinuationQuote},

		// Unclosed brackets.
		{"unclosed paren", "echo $(date", ContinuationBracket},
		{"unclosed brace", "echo ${HOME", ContinuationBracket},
		{"unclosed bracket", "arr[0", ContinuationBracket},
		{"nested unclosed", "echo $((1 + $(date", ContinuationBracket},

		// Heredoc.
		{"heredoc start", "cat <<EOF", ContinuationHeredoc},
		{"heredoc dash", "cat <<-EOF", ContinuationHeredoc},
		{"heredoc quoted", "cat <<'EOF'", ContinuationHeredoc},
		{"heredoc double quoted", `cat <<"EOF"`, ContinuationHeredoc},
		{"heredoc with body not closed", "cat <<EOF\nhello world", ContinuationHeredoc},
		{"heredoc completed", "cat <<EOF\nhello world\nEOF", ContinuationNone},
		{"heredoc with indent completed", "cat <<END\n  some text\nEND", ContinuationNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, _ := needsContinuation(tt.input)
			if reason != tt.reason {
				t.Errorf("needsContinuation(%q) reason = %s, want %s", tt.input, reason, tt.reason)
			}
		})
	}
}

func TestNeedsContinuationHeredocDelimiter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		delim string
	}{
		{"EOF delimiter", "cat <<EOF", "EOF"},
		{"END delimiter", "cat <<END", "END"},
		{"quoted delimiter", "cat <<'MARKER'", "MARKER"},
		{"double quoted delimiter", `cat <<"MARKER"`, "MARKER"},
		{"dash delimiter", "cat <<-INDENT", "INDENT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, delim := needsContinuation(tt.input)
			if reason != ContinuationHeredoc {
				t.Fatalf("expected ContinuationHeredoc, got %s", reason)
			}
			if delim != tt.delim {
				t.Errorf("delimiter = %q, want %q", delim, tt.delim)
			}
		})
	}
}

func TestExtractHeredocDelimiter(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"cat <<EOF", "EOF"},
		{"cat <<-EOF", "EOF"},
		{"cat <<'EOF'", "EOF"},
		{`cat <<"EOF"`, "EOF"},
		{"cat << EOF", "EOF"},
		{"cat <<MARKER", "MARKER"},
		{"cat <<<word", ""},  // here-string, not heredoc
		{"echo hello", ""},   // no heredoc
		{"cat <<", ""},       // no delimiter
		{"cat <<EOF; echo", "EOF"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := extractHeredocDelimiter(tt.line)
			if got != tt.want {
				t.Errorf("extractHeredocDelimiter(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsHereDocComplete(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		delimiter string
		want      bool
	}{
		{"complete", []string{"hello", "world", "EOF"}, "EOF", true},
		{"not complete", []string{"hello", "world"}, "EOF", false},
		{"delimiter with whitespace", []string{"hello", "  EOF  "}, "EOF", true},
		{"delimiter in middle", []string{"EOF", "more text"}, "EOF", true},
		{"empty lines", []string{""}, "EOF", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHereDocComplete(tt.lines, tt.delimiter)
			if got != tt.want {
				t.Errorf("isHereDocComplete(%v, %q) = %v, want %v", tt.lines, tt.delimiter, got, tt.want)
			}
		})
	}
}

func TestCountTrailing(t *testing.T) {
	tests := []struct {
		s    string
		ch   byte
		want int
	}{
		{`hello\`, '\\', 1},
		{`hello\\`, '\\', 2},
		{`hello\\\`, '\\', 3},
		{"hello", '\\', 0},
		{"", '\\', 0},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := countTrailing(tt.s, tt.ch)
			if got != tt.want {
				t.Errorf("countTrailing(%q, %q) = %d, want %d", tt.s, tt.ch, got, tt.want)
			}
		})
	}
}

func TestContinuationReasonString(t *testing.T) {
	tests := []struct {
		reason ContinuationReason
		want   string
	}{
		{ContinuationNone, "none"},
		{ContinuationBackslash, "backslash"},
		{ContinuationQuote, "quote"},
		{ContinuationBracket, "bracket"},
		{ContinuationHeredoc, "heredoc"},
		{ContinuationReason(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.reason.String()
			if got != tt.want {
				t.Errorf("ContinuationReason(%d).String() = %q, want %q", tt.reason, got, tt.want)
			}
		})
	}
}
