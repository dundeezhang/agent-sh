package shell

import (
	"fmt"
	"os"
)

const keyCtrlR = 0x12 // Ctrl+R

// reverseSearch runs an interactive reverse-incremental-search loop, reading
// raw bytes directly from stdin and rendering a search prompt to stdout.
//
// It is designed to be called from within the AutoCompleteCallback of
// term.Terminal while the terminal lock is released.  On return the caller
// should replace the current input line with the result.
//
// Returns the selected command (or "" if cancelled) and whether a command
// was selected.
func (s *Shell) reverseSearch() (string, bool) {
	fd := int(os.Stdin.Fd())
	var query []byte
	matchIdx := s.history.Len() // start search from the end (most recent)
	match := ""
	found := false

	s.renderSearchPrompt(query, match, found)

	buf := make([]byte, 64)
	for {
		n, err := readRaw(fd, buf)
		if err != nil || n == 0 {
			// Read error — cancel search.
			s.clearSearchPrompt()
			return "", false
		}

		// Process each byte in the read buffer.
		for i := 0; i < n; i++ {
			b := buf[i]

			switch {
			case b == keyCtrlR:
				// Cycle to next (older) match.
				if len(query) > 0 && found {
					m, idx, ok := s.history.SearchBackward(string(query), matchIdx)
					if ok {
						match = m
						matchIdx = idx
						found = true
					}
					// If no older match, keep current match displayed.
				}

			case b == '\r' || b == '\n':
				// Accept current match.
				s.clearSearchPrompt()
				if found {
					return match, true
				}
				return "", false

			case b == 0x1b:
				// Escape — might be a standalone Esc or the start of an
				// escape sequence (e.g. arrow keys).  Consume any
				// remaining bytes in the sequence and cancel.
				for i+1 < n {
					i++
					c := buf[i]
					if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '~' {
						break
					}
				}
				s.clearSearchPrompt()
				return "", false

			case b == 0x03:
				// Ctrl+C — cancel.
				s.clearSearchPrompt()
				return "", false

			case b == 0x7f || b == 0x08:
				// Backspace — remove last character from query.
				if len(query) > 0 {
					query = query[:len(query)-1]
				}
				if len(query) == 0 {
					match = ""
					found = false
					matchIdx = s.history.Len()
				} else {
					// Re-search from the end with the shorter query.
					m, idx, ok := s.history.SearchBackward(string(query), s.history.Len())
					if ok {
						match = m
						matchIdx = idx
						found = true
					} else {
						match = ""
						found = false
						matchIdx = s.history.Len()
					}
				}

			case b >= 0x20 && b < 0x7f:
				// Printable ASCII — append to query and search.
				query = append(query, b)
				// Search from the current match position (or end) to
				// allow the user to keep narrowing.
				startFrom := s.history.Len()
				if found {
					startFrom = matchIdx + 1
				}
				m, idx, ok := s.history.SearchBackward(string(query), startFrom)
				if ok {
					match = m
					matchIdx = idx
					found = true
				} else {
					// No match for this query — indicate failure.
					match = ""
					found = false
				}

			default:
				// Ignore other control characters.
				continue
			}

			s.renderSearchPrompt(query, match, found)
		}
	}
}

// renderSearchPrompt writes the reverse-i-search prompt directly to stdout,
// overwriting the current terminal line.
func (s *Shell) renderSearchPrompt(query []byte, match string, found bool) {
	q := string(query)
	var line string
	if !found && len(query) > 0 {
		line = fmt.Sprintf("\r\033[K(failing reverse-i-search)`%s': ", q)
	} else {
		line = fmt.Sprintf("\r\033[K(reverse-i-search)`%s': %s", q, match)
	}
	os.Stdout.WriteString(line)
}

// clearSearchPrompt erases the search prompt line.
func (s *Shell) clearSearchPrompt() {
	os.Stdout.WriteString("\r\033[K")
}

// readRaw reads raw bytes from the given file descriptor.
func readRaw(fd int, buf []byte) (int, error) {
	return ignoringEINTR(fd, buf)
}

// ignoringEINTR retries the read if interrupted by a signal.
func ignoringEINTR(fd int, buf []byte) (int, error) {
	for {
		n, err := rawRead(fd, buf)
		if err != nil && isEINTR(err) {
			continue
		}
		return n, err
	}
}
