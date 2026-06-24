package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRCFilePath(t *testing.T) {
	path := RCFilePath()
	if path == "" {
		t.Fatal("RCFilePath() returned empty string")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	want := filepath.Join(home, ".config", "agent-sh", "init.sh")
	if path != want {
		t.Errorf("RCFilePath() = %q, want %q", path, want)
	}

	if !strings.HasSuffix(path, "init.sh") {
		t.Errorf("RCFilePath() should end with init.sh, got %q", path)
	}
}
