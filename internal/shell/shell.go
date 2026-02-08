package shell

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// AgentHandler is called when the user triggers agent mode with @.
type AgentHandler func(input string)

// Shell is a standalone read-eval-execute loop.
type Shell struct {
	history      *History
	agentHandler AgentHandler
	oldState     *term.State
}

// New creates a new standalone Shell.
func New(history *History, agentHandler AgentHandler) *Shell {
	return &Shell{
		history:      history,
		agentHandler: agentHandler,
	}
}

// Run starts the shell REPL and blocks until exit.
func (s *Shell) Run() error {
	fd := int(os.Stdin.Fd())

	// Ignore job-control signals in the shell process so it can perform
	// terminal operations while child processes are in the foreground.
	signal.Ignore(syscall.SIGTTOU, syscall.SIGTTIN, syscall.SIGTSTP)

	// Put terminal in raw mode for term.Terminal line editing
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	s.oldState = oldState
	defer s.restore()

	t := term.NewTerminal(os.Stdin, prompt())
	// Wire stdout through the terminal so output is properly \r\n translated
	t.SetSize(termSize())

	// Handle terminal resize
	sigWinch := make(chan os.Signal, 1)
	signal.Notify(sigWinch, syscall.SIGWINCH)
	go func() {
		for range sigWinch {
			t.SetSize(termSize())
		}
	}()

	t.AutoCompleteCallback = func(line string, pos int, key rune) (string, int, bool) {
		if key != '\t' {
			return "", 0, false
		}
		wordStart, _, matches := completeWord(line, pos)
		if len(matches) == 0 {
			return "", 0, false
		}
		if len(matches) == 1 {
			completion := matches[0]
			// Append a space after the completion unless it's a directory.
			if !strings.HasSuffix(completion, "/") {
				completion += " "
			}
			newLine := line[:wordStart] + completion + line[pos:]
			newPos := wordStart + len(completion)
			return newLine, newPos, true
		}
		// Multiple matches: show list and complete to common prefix.
		w, _ := termSize()
		fmt.Fprintf(t, "%s", formatColumns(matches, w))
		cp := commonPrefix(matches)
		newLine := line[:wordStart] + cp + line[pos:]
		newPos := wordStart + len(cp)
		return newLine, newPos, true
	}

	for {
		line, err := t.ReadLine()
		if err != nil {
			// EOF (Ctrl-D) or error — exit
			fmt.Fprintln(t, "")
			return nil
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// @ detection
		if strings.HasPrefix(line, "@") {
			rest := line[1:]
			if strings.HasPrefix(rest, "@") {
				// @@ escape — strip one @ and execute as command
				line = rest
			} else {
				// Agent mode
				query := strings.TrimSpace(rest)
				if query != "" {
					s.history.Add(HistoryEntry{Command: line})
					s.restore()
					s.agentHandler(query)
					s.rawMode()
					t.SetPrompt(prompt())
				}
				continue
			}
		}

		// Builtins
		if s.handleBuiltin(line, t) {
			s.history.Add(HistoryEntry{Command: line})
			continue
		}

		// External command
		s.history.Add(HistoryEntry{Command: line})
		s.restore()
		s.execCommand(line)
		s.rawMode()
		t.SetPrompt(prompt())
	}
}
