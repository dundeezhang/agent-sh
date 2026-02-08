package shell

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/term"
)

// execCommand runs an external command with sh -c and returns the exit code.
func (s *Shell) execCommand(line string) int {
	cmd := exec.Command("sh", "-c", line)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Give the child its own process group and make it the foreground group
	// so interactive programs (vim, less, etc.) can read from the terminal.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Foreground: true,
		Ctty:       int(os.Stdin.Fd()),
	}

	err := cmd.Run()

	// Restore the shell as the foreground process group.
	restoreForeground(int(os.Stdin.Fd()))

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		// Non-ExitError (e.g. command not found at OS level) → 127.
		return 127
	}
	return 0
}

// restoreForeground gives the terminal's foreground process group back to the shell.
func restoreForeground(fd int) {
	pgrp := int32(syscall.Getpgrp())
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&pgrp)))
}

// restore puts the terminal back to cooked mode.
func (s *Shell) restore() {
	if s.oldState != nil {
		term.Restore(int(os.Stdin.Fd()), s.oldState)
	}
}

// rawMode puts the terminal back into raw mode (after command execution).
func (s *Shell) rawMode() {
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err == nil {
		s.oldState = state
	}
}
