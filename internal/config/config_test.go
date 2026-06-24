package config

import (
	"testing"
)

func TestAutoErrorHelpEnabled_DefaultTrue(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Shell.AutoErrorHelpEnabled() {
		t.Error("AutoErrorHelpEnabled should default to true when not set")
	}
}

func TestAutoErrorHelpEnabled_ExplicitTrue(t *testing.T) {
	v := true
	c := ShellConfig{AutoErrorHelp: &v}
	if !c.AutoErrorHelpEnabled() {
		t.Error("AutoErrorHelpEnabled should return true when explicitly set to true")
	}
}

func TestAutoErrorHelpEnabled_ExplicitFalse(t *testing.T) {
	v := false
	c := ShellConfig{AutoErrorHelp: &v}
	if c.AutoErrorHelpEnabled() {
		t.Error("AutoErrorHelpEnabled should return false when explicitly set to false")
	}
}
