//go:build !windows

package shell

import (
	"errors"
	"syscall"
)

// rawRead performs a low-level read from the file descriptor.
func rawRead(fd int, buf []byte) (int, error) {
	return syscall.Read(fd, buf)
}

// isEINTR reports whether err is a syscall.EINTR error.
func isEINTR(err error) bool {
	return errors.Is(err, syscall.EINTR)
}
