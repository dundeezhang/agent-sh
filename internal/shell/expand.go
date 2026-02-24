package shell

import (
	"strconv"
	"strings"
	"unicode"
)

// ExpandHistory performs bash-style history expansion on the input line.
// It returns the expanded string, whether any expansion occurred, and any
// error (e.g., referencing a non-existent history entry).
//
// Supported expansions:
//
//	^old^new      — substitute first occurrence of old with new in previous command
//	!!            — the previous command
//	!$            — the last argument of the previous command
//	!-n           — the command n entries back from the end
//	!n            — the command at 1-based index n
//	!string       — the most recent command starting with string
//
// Exclamation marks inside single or double quotes are not expanded.
func ExpandHistory(line string, history *History) (string, bool, error) {
	// Handle ^old^new substitution (must be at the start of the line).
	if strings.HasPrefix(line, "^") {
		return expandQuickSubst(line, history)
	}

	// Walk the line character by character, respecting quotes.
	var result strings.Builder
	changed := false
	inSingle := false
	inDouble := false

	i := 0
	for i < len(line) {
		ch := line[i]

		// Track quoting state.
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			result.WriteByte(ch)
			i++
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			result.WriteByte(ch)
			i++
			continue
		}

		// Inside quotes, don't expand.
		if inSingle || inDouble {
			result.WriteByte(ch)
			i++
			continue
		}

		// Not a history trigger.
		if ch != '!' {
			result.WriteByte(ch)
			i++
			continue
		}

		// We have '!' — look ahead.
		if i+1 >= len(line) {
			// Trailing '!' with nothing after — not an expansion.
			result.WriteByte(ch)
			i++
			continue
		}

		next := line[i+1]

		switch {
		case next == '!':
			// !! — previous command
			cmd, err := history.Last()
			if err != nil {
				return "", false, err
			}
			result.WriteString(cmd)
			changed = true
			i += 2

		case next == '$':
			// !$ — last argument of previous command
			cmd, err := history.Last()
			if err != nil {
				return "", false, err
			}
			arg := lastArg(cmd)
			result.WriteString(arg)
			changed = true
			i += 2

		case next == '-':
			// !-n — relative reference from end
			i += 2 // skip "!-"
			numStr, end := scanDigits(line, i)
			if numStr == "" {
				return "", false, &historyError{msg: "!-: bad number"}
			}
			n, _ := strconv.Atoi(numStr)
			cmd, err := history.GetFromEnd(n)
			if err != nil {
				return "", false, err
			}
			result.WriteString(cmd)
			changed = true
			i = end

		case next >= '0' && next <= '9':
			// !n — absolute index
			numStr, end := scanDigits(line, i+1)
			n, _ := strconv.Atoi(numStr)
			cmd, err := history.Get(n)
			if err != nil {
				return "", false, err
			}
			result.WriteString(cmd)
			changed = true
			i = end

		case next == ' ' || next == '\t' || next == '=':
			// "! " or "!\t" or "!=" — not an expansion, write literally.
			result.WriteByte(ch)
			i++

		default:
			// !string — prefix search
			prefix, end := scanWord(line, i+1)
			if prefix == "" {
				result.WriteByte(ch)
				i++
				continue
			}
			cmd, err := history.Search(prefix)
			if err != nil {
				return "", false, err
			}
			result.WriteString(cmd)
			changed = true
			i = end
		}
	}

	return result.String(), changed, nil
}

// expandQuickSubst handles ^old^new^ substitution on the previous command.
func expandQuickSubst(line string, history *History) (string, bool, error) {
	// Strip leading ^
	rest := line[1:]
	sepIdx := strings.Index(rest, "^")
	if sepIdx < 0 {
		// No second ^ — not a valid substitution.
		return line, false, nil
	}
	old := rest[:sepIdx]
	newPart := rest[sepIdx+1:]
	// Strip optional trailing ^
	newPart = strings.TrimSuffix(newPart, "^")

	if old == "" {
		return "", false, &historyError{msg: "^: bad substitution"}
	}

	cmd, err := history.Last()
	if err != nil {
		return "", false, err
	}

	if !strings.Contains(cmd, old) {
		return "", false, &historyError{msg: "substitution failed"}
	}

	expanded := strings.Replace(cmd, old, newPart, 1)
	return expanded, true, nil
}

// scanDigits reads consecutive digits starting at pos and returns the digit
// string and the position after the last digit.
func scanDigits(line string, pos int) (string, int) {
	start := pos
	for pos < len(line) && line[pos] >= '0' && line[pos] <= '9' {
		pos++
	}
	return line[start:pos], pos
}

// scanWord reads a "word" for !prefix matching: letters, digits, underscores,
// hyphens, dots, and slashes — stopping at whitespace or special characters.
func scanWord(line string, pos int) (string, int) {
	start := pos
	for pos < len(line) {
		r := rune(line[pos])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || r == '/' {
			pos++
		} else {
			break
		}
	}
	return line[start:pos], pos
}

// lastArg returns the last whitespace-delimited argument from a command string.
// It handles simple quoting (single and double quotes).
func lastArg(cmd string) string {
	fields := splitArgs(cmd)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

// splitArgs splits a command string by whitespace, respecting single and
// double quotes. Quotes are stripped from the returned tokens.
func splitArgs(cmd string) []string {
	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case (ch == ' ' || ch == '\t') && !inSingle && !inDouble:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// historyError is a simple error type for history expansion failures.
type historyError struct {
	msg string
}

func (e *historyError) Error() string {
	return e.msg
}
