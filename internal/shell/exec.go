package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/term"
)

// execCommand runs an external command with sh -c and returns the exit code.
// If the command ends with "&", it is launched in the background.
func (s *Shell) execCommand(line string) int {
	if bg, cmd := parseBackground(line); bg {
		return s.execBackground(cmd)
	}
	return s.execForegroundCmd(line)
}

// parseBackground checks whether a command line ends with an unquoted "&"
// (but not "&&"). If so it returns true and the trimmed command.
func parseBackground(line string) (bool, string) {
	trimmed := strings.TrimRight(line, " \t")
	if !strings.HasSuffix(trimmed, "&") {
		return false, line
	}
	// Make sure it is not "&&".
	if strings.HasSuffix(trimmed, "&&") {
		return false, line
	}
	cmd := strings.TrimRight(trimmed[:len(trimmed)-1], " \t")
	if cmd == "" {
		return false, line
	}
	return true, cmd
}

// execForegroundCmd runs a command in the foreground with proper process group
// handling so interactive programs (vim, less, etc.) work correctly.
func (s *Shell) execForegroundCmd(line string) int {
	cmd := exec.Command("sh", "-c", line)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Give the child its own process group and make it the foreground group
	// so interactive programs can read from the terminal.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Foreground: true,
		Ctty:       int(os.Stdin.Fd()),
		Setpgid:    true,
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 127
	}

	// Track this as a job so the SIGTSTP handler can find it.
	j := s.jobs.Add(line, cmd, JobRunning)
	s.jobs.SetCurrent(j.ID)

	exitCode := s.waitForJob(j)

	s.jobs.SetCurrent(0)

	// If the job was stopped (Ctrl+Z), don't remove it.
	if j.Status == JobStopped {
		return 148 // 128 + 20 (SIGTSTP)
	}

	s.jobs.Remove(j.ID)
	return exitCode
}

// waitForJob waits for a foreground job to finish or be stopped.
// Returns the exit code (0 for stopped jobs).
func (s *Shell) waitForJob(j *Job) int {
	if j.Cmd == nil || j.Cmd.Process == nil {
		return 127
	}

	for {
		var ws syscall.WaitStatus
		pid, err := syscall.Wait4(j.Cmd.Process.Pid, &ws, syscall.WUNTRACED, nil)
		if err != nil {
			// ECHILD means the process has already been reaped.
			return 127
		}
		if pid != j.Cmd.Process.Pid {
			continue
		}

		if ws.Stopped() {
			j.Status = JobStopped
			// Restore the shell as the foreground process group.
			restoreForeground(int(os.Stdin.Fd()))
			fmt.Fprintf(os.Stderr, "\n[%d]+  Stopped                 %s\n", j.ID, j.Command)
			return 148
		}

		if ws.Exited() {
			// Restore the shell as the foreground process group.
			restoreForeground(int(os.Stdin.Fd()))
			return ws.ExitStatus()
		}

		if ws.Signaled() {
			// Restore the shell as the foreground process group.
			restoreForeground(int(os.Stdin.Fd()))
			return 128 + int(ws.Signal())
		}

		// For continued or other states, keep waiting.
	}
}

// execBackground launches a command in the background and adds it to the
// job table. Returns 0 immediately.
func (s *Shell) execBackground(line string) int {
	cmd := exec.Command("sh", "-c", line)
	cmd.Stdin = nil // background jobs don't get terminal input
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 127
	}

	j := s.jobs.Add(line, cmd, JobRunning)
	fmt.Fprintf(os.Stderr, "[%d] %d\n", j.ID, j.Pid())
	return 0
}

// bringToForeground makes a job the foreground process group, sends SIGCONT
// if it was stopped, and waits for it to finish or be stopped again.
func (s *Shell) bringToForeground(j *Job) int {
	if j.Cmd == nil || j.Cmd.Process == nil {
		return 127
	}

	fmt.Fprintf(os.Stderr, "%s\n", j.Command)

	// Make the job's process group the foreground group.
	pid := j.Cmd.Process.Pid
	pgid := int32(pid)
	fd := os.Stdin.Fd()
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&pgid)))

	// If stopped, resume it.
	if j.Status == JobStopped {
		j.Status = JobRunning
		_ = j.SendSignal(syscall.SIGCONT)
	}

	s.jobs.SetCurrent(j.ID)
	exitCode := s.waitForJob(j)
	s.jobs.SetCurrent(0)

	if j.Status == JobStopped {
		return 148
	}

	s.jobs.Remove(j.ID)
	return exitCode
}

// resumeInBackground sends SIGCONT to a stopped job so it continues running
// in the background.
func (s *Shell) resumeInBackground(j *Job) {
	if j.Status != JobStopped {
		fmt.Fprintf(os.Stderr, "bg: job %d is not stopped\n", j.ID)
		return
	}
	j.Status = JobRunning
	_ = j.SendSignal(syscall.SIGCONT)
	fmt.Fprintf(os.Stderr, "[%d]+ %s &\n", j.ID, j.Command)
}

// restoreForeground gives the terminal's foreground process group back to the shell.
func restoreForeground(fd int) {
	pgrp := int32(syscall.Getpgrp())
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&pgrp)))
}

// restore puts the terminal back to cooked mode.
func (s *Shell) restore() {
	if s.oldState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), s.oldState)
	}
}

// rawMode puts the terminal back into raw mode (after command execution).
func (s *Shell) rawMode() {
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err == nil {
		s.oldState = state
	}
}
