package shell

import (
	"bytes"
	"strings"
	"testing"
)

func TestLimitedWriter_UnderLimit(t *testing.T) {
	var buf bytes.Buffer
	lw := &limitedWriter{w: &buf, remaining: 100}

	data := []byte("hello world")
	n, err := lw.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("wrote %d bytes, want %d", n, len(data))
	}
	if buf.String() != "hello world" {
		t.Errorf("got %q, want %q", buf.String(), "hello world")
	}
}

func TestLimitedWriter_ExactLimit(t *testing.T) {
	var buf bytes.Buffer
	lw := &limitedWriter{w: &buf, remaining: 5}

	n, err := lw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("wrote %d bytes, want 5", n)
	}
	if buf.String() != "hello" {
		t.Errorf("got %q, want %q", buf.String(), "hello")
	}
}

func TestLimitedWriter_OverLimit(t *testing.T) {
	var buf bytes.Buffer
	lw := &limitedWriter{w: &buf, remaining: 5}

	data := []byte("hello world")
	n, err := lw.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should report writing the full input length (to avoid io.ErrShortWrite for callers)
	// but only actually write up to the limit.
	if n != 5 {
		t.Errorf("wrote %d bytes, want 5", n)
	}
	if buf.String() != "hello" {
		t.Errorf("got %q, want %q", buf.String(), "hello")
	}
}

func TestLimitedWriter_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	lw := &limitedWriter{w: &buf, remaining: 10}

	lw.Write([]byte("hello"))
	lw.Write([]byte(" world!"))

	// Should capture "hello worl" (10 bytes) and discard the rest.
	if buf.String() != "hello worl" {
		t.Errorf("got %q, want %q", buf.String(), "hello worl")
	}
}

func TestLimitedWriter_ZeroRemaining(t *testing.T) {
	var buf bytes.Buffer
	lw := &limitedWriter{w: &buf, remaining: 0}

	n, err := lw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("wrote %d bytes, want 5 (silently discarded)", n)
	}
	if buf.Len() != 0 {
		t.Errorf("buffer should be empty, got %q", buf.String())
	}
}

func TestLimitedWriter_LargeData(t *testing.T) {
	var buf bytes.Buffer
	lw := &limitedWriter{w: &buf, remaining: 100}

	// Write 1000 bytes
	data := []byte(strings.Repeat("x", 1000))
	lw.Write(data)

	if buf.Len() != 100 {
		t.Errorf("buffer length = %d, want 100", buf.Len())
	}
}
