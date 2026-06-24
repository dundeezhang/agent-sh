package shell

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/term"
)

// execResult holds the result of a command execution.
type execResult struct {
	exitCode int
	stderr   string
}

// maxStderrCapture is the maximum number of bytes captured from stderr.
// This prevents excessive memory usage from commands that produce large
// amounts of error output.
const maxStderrCapture = 4096

// execCommand runs an external command with sh -c and returns the result
// including exit code and captured stderr output.
func (s *Shell) execCommand(line string) execResult {
	var stderrBuf bytes.Buffer
	limitedBuf := &limitedWriter{w: &stderrBuf, remaining: maxStderrCapture}

	cmd := exec.Command("sh", "-c", line)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, limitedBuf)

	// Give the child its own process group and make it the foreground group
	// so interactive programs (vim, less, etc.) can read from the terminal.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Foreground: true,
		Ctty:       int(os.Stdin.Fd()),
	}

	err := cmd.Run()

	// Restore the shell as the foreground process group.
	restoreForeground(int(os.Stdin.Fd()))

	captured := stderrBuf.String()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return execResult{exitCode: exitErr.ExitCode(), stderr: captured}
		}
		// Non-ExitError (e.g. command not found at OS level) → 127.
		return execResult{exitCode: 127, stderr: captured}
	}
	return execResult{exitCode: 0, stderr: captured}
}

// limitedWriter wraps a writer and stops writing after a byte limit is reached.
// It silently discards bytes beyond the limit to avoid unbounded memory growth.
type limitedWriter struct {
	w         io.Writer
	remaining int
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.remaining <= 0 {
		return len(p), nil // silently discard
	}
	if len(p) > lw.remaining {
		p = p[:lw.remaining]
	}
	n, err := lw.w.Write(p)
	lw.remaining -= n
	return n, err
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
