package shell

import (
	"fmt"
	"os"
)

// DirStack manages a directory stack for pushd/popd and tracks the previous
// working directory (OLDPWD) for cd -.
type DirStack struct {
	stack  []string
	oldpwd string
}

// NewDirStack returns an initialised DirStack. The OLDPWD is set to the
// current working directory so that the first "cd -" has a sensible target.
func NewDirStack() *DirStack {
	cwd, _ := os.Getwd()
	return &DirStack{oldpwd: cwd}
}

// Push saves the current directory on the stack, changes to dir, and updates
// OLDPWD. It returns the new stack representation (current dir + stack).
func (ds *DirStack) Push(dir string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("pushd: %w", err)
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("pushd: %w", err)
	}

	ds.stack = append(ds.stack, cwd)
	ds.oldpwd = cwd
	return nil
}

// Pop removes the top entry from the stack, changes to that directory, and
// updates OLDPWD.
func (ds *DirStack) Pop() error {
	if len(ds.stack) == 0 {
		return fmt.Errorf("popd: directory stack empty")
	}

	// Pop the top (most recently pushed) entry.
	top := ds.stack[len(ds.stack)-1]
	ds.stack = ds.stack[:len(ds.stack)-1]

	cwd, _ := os.Getwd()

	if err := os.Chdir(top); err != nil {
		return fmt.Errorf("popd: %w", err)
	}

	ds.oldpwd = cwd
	return nil
}

// Swap swaps the current directory with the top of the stack (pushd with no
// args behaviour).
func (ds *DirStack) Swap() error {
	if len(ds.stack) == 0 {
		return fmt.Errorf("pushd: no other directory")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("pushd: %w", err)
	}

	top := ds.stack[len(ds.stack)-1]

	if err := os.Chdir(top); err != nil {
		return fmt.Errorf("pushd: %w", err)
	}

	ds.stack[len(ds.stack)-1] = cwd
	ds.oldpwd = cwd
	return nil
}

// List returns a human-readable representation of the directory stack. The
// current directory is always shown first, followed by the stack entries from
// top (most recent) to bottom.
func (ds *DirStack) List() []string {
	cwd, _ := os.Getwd()
	dirs := []string{cwd}
	// Append from top of stack (most recent push) to bottom.
	for i := len(ds.stack) - 1; i >= 0; i-- {
		dirs = append(dirs, ds.stack[i])
	}
	return dirs
}

// SetOldPWD records the previous working directory.
func (ds *DirStack) SetOldPWD(dir string) {
	ds.oldpwd = dir
}

// OldPWD returns the previous working directory.
func (ds *DirStack) OldPWD() string {
	return ds.oldpwd
}
