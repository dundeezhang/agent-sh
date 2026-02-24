package shell

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRC_NonExistentFile(t *testing.T) {
	s := &Shell{}
	err := s.loadRC("/tmp/agent-sh-test-nonexistent-rc-file")
	if err != nil {
		t.Errorf("loadRC with non-existent file should return nil, got %v", err)
	}
}

func TestLoadRC_EmptyPath(t *testing.T) {
	s := &Shell{}
	err := s.loadRC("")
	if err != nil {
		t.Errorf("loadRC with empty path should return nil, got %v", err)
	}
}

func TestLoadRC_ExportSetsEnv(t *testing.T) {
	dir := t.TempDir()
	rc := filepath.Join(dir, "init.sh")
	os.WriteFile(rc, []byte("export AGENT_SH_TEST_VAR=hello\n"), 0644)

	// Clean up env after test.
	defer os.Unsetenv("AGENT_SH_TEST_VAR")

	s := &Shell{}
	err := s.loadRC(rc)
	if err != nil {
		t.Fatalf("loadRC returned error: %v", err)
	}

	got := os.Getenv("AGENT_SH_TEST_VAR")
	if got != "hello" {
		t.Errorf("AGENT_SH_TEST_VAR = %q, want %q", got, "hello")
	}
}

func TestLoadRC_CommentsAndBlankLines(t *testing.T) {
	dir := t.TempDir()
	rc := filepath.Join(dir, "init.sh")
	content := `# This is a comment
export AGENT_SH_TEST_A=1

# Another comment

export AGENT_SH_TEST_B=2
`
	os.WriteFile(rc, []byte(content), 0644)

	defer os.Unsetenv("AGENT_SH_TEST_A")
	defer os.Unsetenv("AGENT_SH_TEST_B")

	s := &Shell{}
	err := s.loadRC(rc)
	if err != nil {
		t.Fatalf("loadRC returned error: %v", err)
	}

	if got := os.Getenv("AGENT_SH_TEST_A"); got != "1" {
		t.Errorf("AGENT_SH_TEST_A = %q, want %q", got, "1")
	}
	if got := os.Getenv("AGENT_SH_TEST_B"); got != "2" {
		t.Errorf("AGENT_SH_TEST_B = %q, want %q", got, "2")
	}
}

func TestLoadRC_MultipleExports(t *testing.T) {
	dir := t.TempDir()
	rc := filepath.Join(dir, "init.sh")
	os.WriteFile(rc, []byte("export AGENT_SH_TEST_X=foo AGENT_SH_TEST_Y=bar\n"), 0644)

	defer os.Unsetenv("AGENT_SH_TEST_X")
	defer os.Unsetenv("AGENT_SH_TEST_Y")

	s := &Shell{}
	err := s.loadRC(rc)
	if err != nil {
		t.Fatalf("loadRC returned error: %v", err)
	}

	if got := os.Getenv("AGENT_SH_TEST_X"); got != "foo" {
		t.Errorf("AGENT_SH_TEST_X = %q, want %q", got, "foo")
	}
	if got := os.Getenv("AGENT_SH_TEST_Y"); got != "bar" {
		t.Errorf("AGENT_SH_TEST_Y = %q, want %q", got, "bar")
	}
}

func TestLoadRC_ShellCommand(t *testing.T) {
	dir := t.TempDir()
	rc := filepath.Join(dir, "init.sh")
	marker := filepath.Join(dir, "marker")
	content := "touch " + marker + "\n"
	os.WriteFile(rc, []byte(content), 0644)

	s := &Shell{}
	err := s.loadRC(rc)
	if err != nil {
		t.Fatalf("loadRC returned error: %v", err)
	}

	if _, err := os.Stat(marker); err != nil {
		t.Errorf("shell command did not execute: marker file not found")
	}
}

func TestLoadRC_ErrorsDoNotStopProcessing(t *testing.T) {
	dir := t.TempDir()
	rc := filepath.Join(dir, "init.sh")
	content := `export AGENT_SH_TEST_BEFORE=yes
false
export AGENT_SH_TEST_AFTER=yes
`
	os.WriteFile(rc, []byte(content), 0644)

	defer os.Unsetenv("AGENT_SH_TEST_BEFORE")
	defer os.Unsetenv("AGENT_SH_TEST_AFTER")

	s := &Shell{}
	err := s.loadRC(rc)

	// Should have an error from the "false" command.
	if err == nil {
		t.Fatal("loadRC should return error for failing command")
	}

	// Both exports should still have been processed.
	if got := os.Getenv("AGENT_SH_TEST_BEFORE"); got != "yes" {
		t.Errorf("AGENT_SH_TEST_BEFORE = %q, want %q", got, "yes")
	}
	if got := os.Getenv("AGENT_SH_TEST_AFTER"); got != "yes" {
		t.Errorf("AGENT_SH_TEST_AFTER = %q, want %q", got, "yes")
	}
}

func TestLoadRC_CdChangesDirectory(t *testing.T) {
	dir := t.TempDir()
	// Resolve symlinks so comparison works on macOS where /var -> /private/var.
	dir, _ = filepath.EvalSymlinks(dir)

	rc := filepath.Join(dir, "init.sh")
	os.WriteFile(rc, []byte("cd "+dir+"\n"), 0644)

	// Save and restore working directory.
	orig, _ := os.Getwd()
	defer os.Chdir(orig)

	s := &Shell{}
	err := s.loadRC(rc)
	if err != nil {
		t.Fatalf("loadRC returned error: %v", err)
	}

	got, _ := os.Getwd()
	got, _ = filepath.EvalSymlinks(got)
	if got != dir {
		t.Errorf("working directory = %q, want %q", got, dir)
	}
}
