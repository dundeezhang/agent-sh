package shell

import (
	"strings"
)

// ContinuationReason describes why a line needs continuation.
type ContinuationReason int

const (
	// ContinuationNone means the input is complete.
	ContinuationNone ContinuationReason = iota
	// ContinuationBackslash means the line ends with a trailing backslash.
	ContinuationBackslash
	// ContinuationQuote means there is an unclosed quote.
	ContinuationQuote
	// ContinuationBracket means there is an unclosed bracket/paren/brace.
	ContinuationBracket
	// ContinuationHeredoc means a here document was started but not closed.
	ContinuationHeredoc
)

// String returns a human-readable name for the continuation reason.
func (r ContinuationReason) String() string {
	switch r {
	case ContinuationNone:
		return "none"
	case ContinuationBackslash:
		return "backslash"
	case ContinuationQuote:
		return "quote"
	case ContinuationBracket:
		return "bracket"
	case ContinuationHeredoc:
		return "heredoc"
	default:
		return "unknown"
	}
}

// needsContinuation checks whether the accumulated input is incomplete and
// requires additional lines. It returns the reason for continuation and,
// for heredocs, the delimiter that must appear to close the document.
func needsContinuation(input string) (ContinuationReason, string) {
	// Check for heredoc start on the last logical line.
	if reason, delim := detectHeredoc(input); reason == ContinuationHeredoc {
		return reason, delim
	}

	// Check for trailing backslash (after trimming trailing whitespace).
	trimmed := strings.TrimRight(input, " \t")
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '\\' {
		// Make sure the backslash itself is not escaped (odd number of
		// trailing backslashes means continuation).
		n := countTrailing(trimmed, '\\')
		if n%2 == 1 {
			return ContinuationBackslash, ""
		}
	}

	// Walk the entire input to track quoting and bracket depth.
	inSingle := false
	inDouble := false
	depth := 0 // net depth of ( [ {

	for i := 0; i < len(input); i++ {
		ch := input[i]

		// Inside single quotes, only a closing single quote matters.
		if inSingle {
			if ch == '\'' {
				inSingle = false
			}
			continue
		}

		// Inside double quotes, handle escapes and closing quote.
		if inDouble {
			if ch == '\\' && i+1 < len(input) {
				i++ // skip escaped character
				continue
			}
			if ch == '"' {
				inDouble = false
			}
			continue
		}

		// Outside any quoting.
		switch ch {
		case '\\':
			if i+1 < len(input) {
				i++ // skip escaped character
			}
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		}
	}

	if inSingle || inDouble {
		return ContinuationQuote, ""
	}
	if depth > 0 {
		return ContinuationBracket, ""
	}

	return ContinuationNone, ""
}

// detectHeredoc scans the input for an unmatched here document redirection
// (<<DELIM or <<-DELIM or <<'DELIM'). It looks at the last occurrence
// of a heredoc operator and checks whether the delimiter has been seen
// on a subsequent line.
func detectHeredoc(input string) (ContinuationReason, string) {
	lines := strings.Split(input, "\n")

	// Scan all lines for heredoc operators; track the last one found.
	lastDelim := ""
	lastDelimLine := -1

	for i, line := range lines {
		if delim := extractHeredocDelimiter(line); delim != "" {
			lastDelim = delim
			lastDelimLine = i
		}
	}

	if lastDelim == "" {
		return ContinuationNone, ""
	}

	// Check if any line after the heredoc operator matches the delimiter.
	for i := lastDelimLine + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == lastDelim {
			return ContinuationNone, ""
		}
	}

	return ContinuationHeredoc, lastDelim
}

// extractHeredocDelimiter looks for a <<[-]DELIM pattern in a single line
// and returns the delimiter, or "" if none found. It handles quoted
// delimiters like <<'EOF' and <<"EOF".
func extractHeredocDelimiter(line string) string {
	idx := strings.Index(line, "<<")
	if idx < 0 {
		return ""
	}
	rest := line[idx+2:]

	// Skip optional '-' for <<- (tab-stripped heredocs).
	if len(rest) > 0 && rest[0] == '-' {
		rest = rest[1:]
	}

	// Skip leading whitespace between << and delimiter.
	rest = strings.TrimLeft(rest, " \t")

	if len(rest) == 0 {
		return ""
	}

	// Ignore <<< (here-string, not heredoc).
	if rest[0] == '<' {
		return ""
	}

	// Quoted delimiter: <<'DELIM' or <<"DELIM".
	if rest[0] == '\'' || rest[0] == '"' {
		quote := rest[0]
		end := strings.IndexByte(rest[1:], quote)
		if end < 0 {
			return ""
		}
		delim := rest[1 : 1+end]
		if delim == "" {
			return ""
		}
		return delim
	}

	// Unquoted delimiter: read until whitespace or end of string.
	var delim strings.Builder
	for _, ch := range rest {
		if ch == ' ' || ch == '\t' || ch == ';' || ch == '&' || ch == '|' || ch == ')' {
			break
		}
		delim.WriteRune(ch)
	}
	result := delim.String()
	if result == "" {
		return ""
	}
	return result
}

// isHereDocComplete checks whether the heredoc body (represented as
// multiple lines) contains the closing delimiter on its own line.
func isHereDocComplete(lines []string, delimiter string) bool {
	for _, line := range lines {
		if strings.TrimSpace(line) == delimiter {
			return true
		}
	}
	return false
}

// countTrailing counts how many consecutive occurrences of ch appear at
// the end of s.
func countTrailing(s string, ch byte) int {
	n := 0
	for i := len(s) - 1; i >= 0 && s[i] == ch; i-- {
		n++
	}
	return n
}
