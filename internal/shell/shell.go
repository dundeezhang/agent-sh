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

	// Handle terminal resize; stop on exit to avoid goroutine leak.
	sigWinch := make(chan os.Signal, 1)
	signal.Notify(sigWinch, syscall.SIGWINCH)
	defer func() {
		signal.Stop(sigWinch)
		close(sigWinch)
	}()
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
		forceBash := false
		if strings.HasPrefix(line, "@") {
			rest := line[1:]
			if strings.HasPrefix(rest, "@") {
				// @@ escape — force bash, skip classification
				line = strings.TrimSpace(rest[1:])
				if line == "" {
					continue
				}
				forceBash = true
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

		s.history.Add(HistoryEntry{Command: line})

		if !forceBash {
			switch classifyInput(line) {
			case ClassAgent:
				s.restore()
				s.agentHandler(line)
				s.rawMode()
				t.SetPrompt(prompt())
				continue
			case ClassUnsure:
				s.restore()
				s.agentHandler("[The user typed something ambiguous — it might be a question, a request, or a mistyped command. Respond conversationally. Make your best assumption about what they mean, but do NOT use any tools. Just reply in plain text.]\n\n" + line)
				s.rawMode()
				t.SetPrompt(prompt())
				continue
			}
		}

		// Execute as shell command; exit-127 fallback sends to agent.
		s.restore()
		if exitCode := s.execCommand(line); exitCode == 127 && !forceBash {
			fmt.Fprintf(os.Stderr, "command not found, asking AI...\n")
			s.agentHandler(line)
		}
		s.rawMode()
		t.SetPrompt(prompt())
	}
}
