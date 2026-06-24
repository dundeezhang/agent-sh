package shell

import (
	"bufio"
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
	history       *History
	agentHandler  AgentHandler
	oldState      *term.State
	autoErrorHelp bool
}

// New creates a new standalone Shell.
func New(history *History, agentHandler AgentHandler, autoErrorHelp bool) *Shell {
	return &Shell{
		history:       history,
		agentHandler:  agentHandler,
		autoErrorHelp: autoErrorHelp,
	}
}

// runAgent restores cooked mode, calls the agent handler, then re-enters
// raw mode and refreshes the prompt. This pattern is repeated in several
// places in the REPL so it's factored out here.
func (s *Shell) runAgent(t *term.Terminal, input string) {
	s.restore()
	s.agentHandler(input)
	s.rawMode()
	t.SetPrompt(prompt())
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
					s.history.Add(line)
					s.runAgent(t, query)
				}
				continue
			}
		}

		// Builtins
		if s.handleBuiltin(line, t) {
			s.history.Add(line)
			continue
		}

		s.history.Add(line)

		if !forceBash {
			switch classifyInput(line) {
			case ClassAgent:
				s.runAgent(t, line)
				continue
			case ClassUnsure:
				s.runAgent(t, "[The user typed something ambiguous — it might be a question, a request, or a mistyped command. Respond conversationally. Make your best assumption about what they mean, but do NOT use any tools. Just reply in plain text.]\n\n"+line)
				continue
			}
		}

		// Execute as shell command; exit-127 fallback sends to agent.
		s.restore()
		result := s.execCommand(line)
		if result.exitCode == 127 && !forceBash {
			fmt.Fprintf(os.Stderr, "command not found, asking AI...\n")
			s.agentHandler(line)
		} else if s.shouldOfferErrorHelp(result) {
			if s.promptErrorHelp(result.exitCode) {
				query := formatErrorQuery(line, result.exitCode, result.stderr)
				s.agentHandler(query)
			}
		}
		s.rawMode()
		t.SetPrompt(prompt())
	}
}

// shouldOfferErrorHelp decides whether to offer AI help for a failed command.
func (s *Shell) shouldOfferErrorHelp(r execResult) bool {
	if !s.autoErrorHelp {
		return false
	}
	if r.exitCode == 0 {
		return false
	}
	// Exit code 130 = SIGINT (Ctrl-C); the user intentionally interrupted.
	if r.exitCode == 130 {
		return false
	}
	// Exit code 127 is already handled by the command-not-found fallback.
	if r.exitCode == 127 {
		return false
	}
	// Exit code 1 with empty stderr is common for grep no-match, test
	// assertions, and other expected non-error failures.
	if r.exitCode == 1 && strings.TrimSpace(r.stderr) == "" {
		return false
	}
	// Need some stderr output to provide useful context to the agent.
	if strings.TrimSpace(r.stderr) == "" {
		return false
	}
	return true
}

// promptErrorHelp prints a subtle prompt asking the user whether they want
// AI help and reads a single-character response from stdin.
func (s *Shell) promptErrorHelp(exitCode int) bool {
	fmt.Fprintf(os.Stderr, "\033[2mCommand failed (exit %d). Ask AI for help? [y/N]\033[0m ", exitCode)
	reader := bufio.NewReader(os.Stdin)
	b, err := reader.ReadByte()
	fmt.Fprintln(os.Stderr) // move to next line
	if err != nil {
		return false
	}
	return b == 'y' || b == 'Y'
}

// formatErrorQuery builds the query string sent to the agent when the user
// requests help with a failed command.
func formatErrorQuery(command string, exitCode int, stderr string) string {
	return fmt.Sprintf(
		"The command `%s` failed with exit code %d. Error output:\n```\n%s\n```\nHelp me understand and fix this error.",
		command, exitCode, strings.TrimSpace(stderr),
	)
}
