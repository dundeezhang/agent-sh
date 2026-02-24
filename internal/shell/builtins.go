package shell

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// handleBuiltin returns true if the line was a builtin command.
func (s *Shell) handleBuiltin(line string, t *term.Terminal) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}

	switch parts[0] {
	case "exit":
		s.restore()
		os.Exit(0)
		return true

	case "cd":
		dir := ""
		if len(parts) > 1 {
			dir = parts[1]
		}
		if dir == "" || dir == "~" {
			dir, _ = os.UserHomeDir()
		} else if strings.HasPrefix(dir, "~/") {
			home, _ := os.UserHomeDir()
			dir = home + dir[1:]
		}
		if err := os.Chdir(dir); err != nil {
			fmt.Fprintf(t, "cd: %s\r\n", err)
		}
		t.SetPrompt(prompt())
		return true

	case "export":
		if len(parts) < 2 {
			return true
		}
		for _, arg := range parts[1:] {
			if idx := strings.Index(arg, "="); idx >= 0 {
				os.Setenv(arg[:idx], arg[idx+1:])
			}
		}
		return true

	case "env":
		for _, e := range os.Environ() {
			fmt.Fprintf(t, "%s\r\n", e)
		}
		return true

	case "history":
		for i, cmd := range s.history.Recent(0) {
			fmt.Fprintf(t, "%4d  %s\r\n", i+1, cmd)
		}
		return true

	case "jobs":
		s.builtinJobs(t)
		return true

	case "fg":
		s.builtinFg(parts, t)
		return true

	case "bg":
		s.builtinBg(parts, t)
		return true
	}

	return false
}

// builtinJobs lists all jobs with their status.
func (s *Shell) builtinJobs(t *term.Terminal) {
	jobs := s.jobs.List()
	if len(jobs) == 0 {
		return
	}
	for _, j := range jobs {
		indicator := " "
		// Mark the most recent job with "+".
		if recent := s.jobs.MostRecent(); recent != nil && j.ID == recent.ID {
			indicator = "+"
		}
		suffix := ""
		if j.Status == JobRunning {
			suffix = " &"
		}
		fmt.Fprintf(t, "[%d]%s  %-24s%s%s\r\n", j.ID, indicator, j.Status.String(), j.Command, suffix)
	}
}

// builtinFg brings a job to the foreground.
// Usage: fg [%n] or fg [n]
func (s *Shell) builtinFg(parts []string, t *term.Terminal) {
	j := s.resolveJobArg(parts)
	if j == nil {
		fmt.Fprintf(t, "fg: no current job\r\n")
		return
	}

	if j.Status == JobDone {
		fmt.Fprintf(t, "fg: job has already terminated\r\n")
		s.jobs.Remove(j.ID)
		return
	}

	// Leave raw mode, bring the job to the foreground, then re-enter raw mode.
	s.restore()
	s.bringToForeground(j)
	s.rawMode()
	t.SetPrompt(prompt())
}

// builtinBg resumes a stopped job in the background.
// Usage: bg [%n] or bg [n]
func (s *Shell) builtinBg(parts []string, t *term.Terminal) {
	j := s.resolveJobArg(parts)
	if j == nil {
		fmt.Fprintf(t, "bg: no current job\r\n")
		return
	}

	if j.Status != JobStopped {
		fmt.Fprintf(t, "bg: job %d is not stopped\r\n", j.ID)
		return
	}

	s.resumeInBackground(j)
}

// resolveJobArg parses the optional job argument from fg/bg commands.
// Accepts "%n" or "n". Returns the most recent job if no argument is given.
func (s *Shell) resolveJobArg(parts []string) *Job {
	if len(parts) < 2 {
		return s.jobs.MostRecent()
	}

	arg := parts[1]
	// Strip leading "%" if present.
	arg = strings.TrimPrefix(arg, "%")

	id, err := strconv.Atoi(arg)
	if err != nil {
		return nil
	}

	return s.jobs.Get(id)
}
