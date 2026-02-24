package shell

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// loadRC reads the RC file at path and executes each line. Lines starting
// with # and empty lines are skipped. export and cd commands are handled
// as builtins; everything else is executed via sh -c. Errors are collected
// and returned together so that a single bad line does not prevent the rest
// of the file from being processed. If the file does not exist, loadRC
// returns nil.
func (s *Shell) loadRC(path string) error {
	if path == "" {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("rc: %w", err)
	}
	defer f.Close()

	var errs []error
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle export as a builtin so environment variables are set
		// in the current process rather than a subshell.
		parts := strings.Fields(line)
		if parts[0] == "export" {
			for _, arg := range parts[1:] {
				if idx := strings.Index(arg, "="); idx >= 0 {
					os.Setenv(arg[:idx], arg[idx+1:])
				}
			}
			continue
		}

		// Handle cd as a builtin.
		if parts[0] == "cd" {
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
				errs = append(errs, fmt.Errorf("rc:%d: cd: %w", lineNum, err))
			}
			continue
		}

		// Execute everything else as a shell command. We use a simple
		// exec here (no foreground process group) because the terminal
		// is not yet in raw mode during RC loading.
		cmd := exec.Command("sh", "-c", line)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			code := 1
			if errors.As(err, &exitErr) {
				code = exitErr.ExitCode()
			}
			errs = append(errs, fmt.Errorf("rc:%d: %q exited with code %d", lineNum, line, code))
		}
	}

	if err := scanner.Err(); err != nil {
		errs = append(errs, fmt.Errorf("rc: reading file: %w", err))
	}

	return errors.Join(errs...)
}
