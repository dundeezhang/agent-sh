package shell

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

// viMode represents the current vi editing mode.
type viMode int

const (
	viInsert viMode = iota
	viNormal
)

// viEditor implements a vi-style line editor that reads raw bytes from stdin.
type viEditor struct {
	fd      int
	line    []rune
	pos     int
	mode    viMode
	history *History
	histIdx int    // current position in history browsing (-1 = live line)
	saved   string // saved live line while browsing history

	// pending is set when a key needs a second key (e.g. 'd' waiting for 'd').
	pending rune

	// completeFn is the tab-completion callback, matching term.Terminal's signature.
	completeFn func(line string, pos int, key rune) (string, int, bool)
}

func newViEditor(fd int, history *History) *viEditor {
	return &viEditor{
		fd:      fd,
		history: history,
		mode:    viInsert,
		histIdx: -1,
	}
}

// modeIndicator returns the mode string for the prompt.
func (v *viEditor) modeIndicator() string {
	if v.mode == viNormal {
		return "\033[1;33m[N]\033[0m "
	}
	return "\033[1;32m[I]\033[0m "
}

// readLine reads a single line of input with vi-style editing.
// The promptStr should be the raw prompt (without mode indicator).
func (v *viEditor) readLine(promptStr string) (string, error) {
	v.line = nil
	v.pos = 0
	v.mode = viInsert
	v.pending = 0
	v.histIdx = -1
	v.saved = ""

	v.redraw(promptStr)

	buf := make([]byte, 64)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return "", err
		}

		for i := 0; i < n; {
			b := buf[i]
			i++

			// Handle escape sequences (arrow keys, etc.)
			if b == 0x1b && i+1 < n && buf[i] == '[' {
				i++ // consume '['
				if i < n {
					seq := buf[i]
					i++
					done, line, err := v.handleEscapeSeq(seq)
					if err != nil {
						return "", err
					}
					if done {
						return line, nil
					}
					v.redraw(promptStr)
					continue
				}
			}

			done, line, err := v.handleByte(b)
			if err != nil {
				return "", err
			}
			if done {
				return line, nil
			}
			v.redraw(promptStr)
		}
	}
}

// handleEscapeSeq processes an ANSI escape sequence character.
// Returns (done, line, error).
func (v *viEditor) handleEscapeSeq(seq byte) (bool, string, error) {
	switch seq {
	case 'A': // Up arrow - history previous
		v.historyPrev()
	case 'B': // Down arrow - history next
		v.historyNext()
	case 'C': // Right arrow
		if v.pos < len(v.line) {
			v.pos++
		}
	case 'D': // Left arrow
		if v.pos > 0 {
			v.pos--
		}
	case 'H': // Home
		v.pos = 0
	case 'F': // End
		v.pos = len(v.line)
	}
	return false, "", nil
}

// handleByte processes a single byte of input.
// Returns (done, line, error).
func (v *viEditor) handleByte(b byte) (bool, string, error) {
	if v.mode == viInsert {
		return v.handleInsert(b)
	}
	return v.handleNormal(b)
}

// handleInsert processes a byte in insert mode.
func (v *viEditor) handleInsert(b byte) (bool, string, error) {
	switch b {
	case 0x1b: // Escape -> normal mode
		v.mode = viNormal
		// In vi, entering normal mode positions cursor on the last typed char.
		if v.pos > 0 && v.pos == len(v.line) {
			v.pos--
		}
		v.pending = 0
		return false, "", nil

	case '\r', '\n': // Enter
		// Move to a new line before returning.
		fmt.Fprint(os.Stdout, "\r\n")
		return true, string(v.line), nil

	case 0x03: // Ctrl-C: clear line
		v.line = nil
		v.pos = 0
		fmt.Fprint(os.Stdout, "\r\n")
		return false, "", nil

	case 0x04: // Ctrl-D: EOF if line is empty
		if len(v.line) == 0 {
			return true, "", io.EOF
		}
		// Otherwise delete char at cursor
		if v.pos < len(v.line) {
			v.line = append(v.line[:v.pos], v.line[v.pos+1:]...)
		}
		return false, "", nil

	case 0x7f, 0x08: // Backspace
		if v.pos > 0 {
			v.line = append(v.line[:v.pos-1], v.line[v.pos:]...)
			v.pos--
		}
		return false, "", nil

	case '\t': // Tab completion
		if v.completeFn != nil {
			newLine, newPos, ok := v.completeFn(string(v.line), v.pos, '\t')
			if ok {
				v.line = []rune(newLine)
				v.pos = newPos
			}
		}
		return false, "", nil

	case 0x01: // Ctrl-A: beginning of line
		v.pos = 0
		return false, "", nil

	case 0x05: // Ctrl-E: end of line
		v.pos = len(v.line)
		return false, "", nil

	case 0x0b: // Ctrl-K: kill to end of line
		v.line = v.line[:v.pos]
		return false, "", nil

	case 0x15: // Ctrl-U: kill to beginning of line
		v.line = v.line[v.pos:]
		v.pos = 0
		return false, "", nil

	case 0x17: // Ctrl-W: delete previous word
		if v.pos > 0 {
			// Skip trailing spaces
			end := v.pos
			for v.pos > 0 && v.line[v.pos-1] == ' ' {
				v.pos--
			}
			// Delete to previous word boundary
			for v.pos > 0 && v.line[v.pos-1] != ' ' {
				v.pos--
			}
			v.line = append(v.line[:v.pos], v.line[end:]...)
		}
		return false, "", nil

	default:
		if b >= 0x20 && b < 0x7f {
			// Insert printable character
			r := rune(b)
			newLine := make([]rune, len(v.line)+1)
			copy(newLine, v.line[:v.pos])
			newLine[v.pos] = r
			copy(newLine[v.pos+1:], v.line[v.pos:])
			v.line = newLine
			v.pos++
		}
		return false, "", nil
	}
}

// handleNormal processes a byte in normal mode.
func (v *viEditor) handleNormal(b byte) (bool, string, error) {
	// Handle pending operator (e.g. 'd' waiting for 'd')
	if v.pending != 0 {
		result := v.handlePending(b)
		v.pending = 0
		return result, "", nil
	}

	switch b {
	case '\r', '\n': // Enter
		fmt.Fprint(os.Stdout, "\r\n")
		return true, string(v.line), nil

	case 0x03: // Ctrl-C
		v.line = nil
		v.pos = 0
		v.mode = viInsert
		fmt.Fprint(os.Stdout, "\r\n")
		return false, "", nil

	case 0x04: // Ctrl-D
		if len(v.line) == 0 {
			return true, "", io.EOF
		}
		return false, "", nil

	// Movement
	case 'h': // left
		if v.pos > 0 {
			v.pos--
		}
	case 'l': // right
		if v.pos < len(v.line)-1 {
			v.pos++
		}
	case '0': // beginning of line
		v.pos = 0
	case '$': // end of line
		if len(v.line) > 0 {
			v.pos = len(v.line) - 1
		}
	case '^': // first non-space
		v.pos = 0
		for v.pos < len(v.line) && v.line[v.pos] == ' ' {
			v.pos++
		}

	case 'w': // next word
		v.pos = v.nextWord()
	case 'b': // previous word
		v.pos = v.prevWord()
	case 'e': // end of word
		v.pos = v.endOfWord()

	// Editing
	case 'x': // delete char at cursor
		if len(v.line) > 0 && v.pos < len(v.line) {
			v.line = append(v.line[:v.pos], v.line[v.pos+1:]...)
			if v.pos >= len(v.line) && v.pos > 0 {
				v.pos = len(v.line) - 1
			}
		}
	case 'X': // delete char before cursor
		if v.pos > 0 {
			v.line = append(v.line[:v.pos-1], v.line[v.pos:]...)
			v.pos--
		}
	case 'r': // replace single char (needs next byte)
		v.pending = 'r'
	case 'd': // delete operator (needs second key)
		v.pending = 'd'
	case 'D': // delete to end of line
		v.line = v.line[:v.pos]
		if v.pos > 0 {
			v.pos--
		}
	case 'C': // change to end of line
		v.line = v.line[:v.pos]
		v.mode = viInsert
	case 'S', 'c': // 'S' substitutes entire line; 'c' is change operator
		if b == 'S' {
			v.line = nil
			v.pos = 0
			v.mode = viInsert
		} else {
			v.pending = 'c'
		}
	case 's': // substitute char at cursor
		if v.pos < len(v.line) {
			v.line = append(v.line[:v.pos], v.line[v.pos+1:]...)
		}
		v.mode = viInsert

	// Mode switching
	case 'i': // insert at cursor
		v.mode = viInsert
	case 'I': // insert at beginning
		v.pos = 0
		v.mode = viInsert
	case 'a': // append after cursor
		if len(v.line) > 0 {
			v.pos++
		}
		v.mode = viInsert
	case 'A': // append at end
		v.pos = len(v.line)
		v.mode = viInsert

	// History
	case 'k': // previous history
		v.historyPrev()
	case 'j': // next history
		v.historyNext()

	case 0x1b: // Escape (noop in normal mode)
		// noop
	}

	return false, "", nil
}

// handlePending processes the second key of a two-key sequence.
func (v *viEditor) handlePending(b byte) bool {
	switch v.pending {
	case 'd':
		switch b {
		case 'd': // dd: delete entire line
			v.line = nil
			v.pos = 0
		case 'w': // dw: delete word
			end := v.nextWord()
			v.line = append(v.line[:v.pos], v.line[end:]...)
			if v.pos >= len(v.line) && v.pos > 0 {
				v.pos = len(v.line) - 1
			}
		case 'b': // db: delete backward word
			start := v.prevWord()
			v.line = append(v.line[:start], v.line[v.pos:]...)
			v.pos = start
		case '$': // d$: delete to end
			v.line = v.line[:v.pos]
			if v.pos > 0 {
				v.pos--
			}
		case '0': // d0: delete to beginning
			v.line = v.line[v.pos:]
			v.pos = 0
		}
	case 'c':
		switch b {
		case 'c': // cc: change entire line
			v.line = nil
			v.pos = 0
			v.mode = viInsert
		case 'w': // cw: change word
			end := v.nextWord()
			v.line = append(v.line[:v.pos], v.line[end:]...)
			v.mode = viInsert
		case 'b': // cb: change backward word
			start := v.prevWord()
			v.line = append(v.line[:start], v.line[v.pos:]...)
			v.pos = start
			v.mode = viInsert
		case '$': // c$: change to end
			v.line = v.line[:v.pos]
			v.mode = viInsert
		case '0': // c0: change to beginning
			v.line = v.line[v.pos:]
			v.pos = 0
			v.mode = viInsert
		}
	case 'r':
		// Replace character at cursor
		if b >= 0x20 && b < 0x7f && v.pos < len(v.line) {
			v.line[v.pos] = rune(b)
		}
	}
	return false
}

// nextWord returns the position of the start of the next word.
func (v *viEditor) nextWord() int {
	p := v.pos
	n := len(v.line)
	if p >= n {
		return p
	}
	// Skip current word characters
	for p < n && !unicode.IsSpace(v.line[p]) {
		p++
	}
	// Skip whitespace
	for p < n && unicode.IsSpace(v.line[p]) {
		p++
	}
	return p
}

// prevWord returns the position of the start of the previous word.
func (v *viEditor) prevWord() int {
	p := v.pos
	if p <= 0 {
		return 0
	}
	p--
	// Skip whitespace
	for p > 0 && unicode.IsSpace(v.line[p]) {
		p--
	}
	// Skip word characters
	for p > 0 && !unicode.IsSpace(v.line[p-1]) {
		p--
	}
	return p
}

// endOfWord returns the position of the end of the current/next word.
func (v *viEditor) endOfWord() int {
	p := v.pos
	n := len(v.line)
	if p >= n-1 {
		return max(0, n-1)
	}
	p++
	// Skip whitespace
	for p < n && unicode.IsSpace(v.line[p]) {
		p++
	}
	// Skip to end of word
	for p < n-1 && !unicode.IsSpace(v.line[p+1]) {
		p++
	}
	return p
}

// historyPrev navigates to the previous history entry.
func (v *viEditor) historyPrev() {
	entries := v.history.Recent(0)
	if len(entries) == 0 {
		return
	}
	if v.histIdx == -1 {
		v.saved = string(v.line)
		v.histIdx = len(entries) - 1
	} else if v.histIdx > 0 {
		v.histIdx--
	} else {
		return
	}
	v.line = []rune(entries[v.histIdx])
	v.pos = len(v.line)
	if v.mode == viNormal && v.pos > 0 {
		v.pos--
	}
}

// historyNext navigates to the next history entry.
func (v *viEditor) historyNext() {
	entries := v.history.Recent(0)
	if v.histIdx == -1 {
		return
	}
	if v.histIdx < len(entries)-1 {
		v.histIdx++
		v.line = []rune(entries[v.histIdx])
	} else {
		v.histIdx = -1
		v.line = []rune(v.saved)
	}
	v.pos = len(v.line)
	if v.mode == viNormal && v.pos > 0 {
		v.pos--
	}
}

// redraw clears the current line and redraws the prompt, mode indicator, and line content.
func (v *viEditor) redraw(promptStr string) {
	w, _ := termSize()

	// Build the display line: mode indicator + prompt + text
	indicator := v.modeIndicator()
	display := indicator + promptStr + string(v.line)

	// Calculate cursor position within the display.
	// The visible prompt length (without ANSI escapes).
	visIndicator := stripAnsi(indicator)
	visPrompt := stripAnsi(promptStr)
	cursorCol := len(visIndicator) + len(visPrompt) + v.pos

	// Clear and redraw. Use \r to go to column 0, then clear the line.
	fmt.Fprintf(os.Stdout, "\r\033[K%s", display)

	// Handle line wrapping: position cursor correctly.
	totalLen := len(visIndicator) + len(visPrompt) + len(v.line)
	_ = totalLen

	// Move cursor to the right position.
	// We're currently at the end of the line; move back to cursor position.
	cursorRow := cursorCol / w
	endRow := (len(visIndicator) + len(visPrompt) + len(v.line)) / w

	// Move up from end to cursor row
	if endRow > cursorRow {
		fmt.Fprintf(os.Stdout, "\033[%dA", endRow-cursorRow)
	}
	// Set column position
	col := cursorCol % w
	fmt.Fprintf(os.Stdout, "\r\033[%dC", col)
}

// stripAnsi removes ANSI escape sequences from a string to calculate visible length.
func stripAnsi(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		if r == 0x1b {
			inEsc = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
