package shell

import (
	"os"
	"path/filepath"
	"testing"
)

// realTempDir creates a temp directory and resolves symlinks so that path
// comparisons work on macOS (where /var -> /private/var).
func realTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

func TestDirStack_PushPop(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	dirA := realTempDir(t)
	dirB := realTempDir(t)

	os.Chdir(dirA)
	ds := NewDirStack()

	// Push dirB — should cd to dirB and save dirA on the stack.
	if err := ds.Push(dirB); err != nil {
		t.Fatalf("Push: %v", err)
	}
	cwd, _ := os.Getwd()
	if cwd != dirB {
		t.Fatalf("after Push, cwd = %q; want %q", cwd, dirB)
	}
	if ds.OldPWD() != dirA {
		t.Fatalf("after Push, OldPWD = %q; want %q", ds.OldPWD(), dirA)
	}

	// Pop — should cd back to dirA.
	if err := ds.Pop(); err != nil {
		t.Fatalf("Pop: %v", err)
	}
	cwd, _ = os.Getwd()
	if cwd != dirA {
		t.Fatalf("after Pop, cwd = %q; want %q", cwd, dirA)
	}

	// Pop on empty stack should return an error.
	if err := ds.Pop(); err == nil {
		t.Fatal("Pop on empty stack should error")
	}
}

func TestDirStack_Swap(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	dirA := realTempDir(t)
	dirB := realTempDir(t)

	os.Chdir(dirA)
	ds := NewDirStack()

	// Swap with empty stack should error.
	if err := ds.Swap(); err == nil {
		t.Fatal("Swap on empty stack should error")
	}

	// Push dirB, then swap — should go back to dirA.
	if err := ds.Push(dirB); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if err := ds.Swap(); err != nil {
		t.Fatalf("Swap: %v", err)
	}
	cwd, _ := os.Getwd()
	if cwd != dirA {
		t.Fatalf("after Swap, cwd = %q; want %q", cwd, dirA)
	}
}

func TestDirStack_List(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	dirA := realTempDir(t)
	dirB := realTempDir(t)
	dirC := realTempDir(t)

	os.Chdir(dirA)
	ds := NewDirStack()

	ds.Push(dirB)
	ds.Push(dirC)

	list := ds.List()
	// Expected: current (dirC), then dirB (top of stack), then dirA (bottom).
	if len(list) != 3 {
		t.Fatalf("List() returned %d entries; want 3", len(list))
	}
	if list[0] != dirC {
		t.Errorf("List()[0] = %q; want %q", list[0], dirC)
	}
	if list[1] != dirB {
		t.Errorf("List()[1] = %q; want %q", list[1], dirB)
	}
	if list[2] != dirA {
		t.Errorf("List()[2] = %q; want %q", list[2], dirA)
	}
}

func TestDirStack_OldPWD(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	dirA := realTempDir(t)
	os.Chdir(dirA)
	ds := NewDirStack()

	// Initial OldPWD should be the starting directory.
	if ds.OldPWD() != dirA {
		t.Fatalf("initial OldPWD = %q; want %q", ds.OldPWD(), dirA)
	}

	ds.SetOldPWD("/tmp")
	if ds.OldPWD() != "/tmp" {
		t.Fatalf("OldPWD after Set = %q; want %q", ds.OldPWD(), "/tmp")
	}
}
