package render

import (
	"bytes"
	"testing"
)

func TestRenderInline(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Bold
		{"**hello**", boldOn + "hello" + boldOff},
		{"say **bold** word", "say " + boldOn + "bold" + boldOff + " word"},
		// Italic
		{"*italic*", italicOn + "italic" + italicOff},
		{"say *italic* word", "say " + italicOn + "italic" + italicOff + " word"},
		// Bold+Italic
		{"***both***", boldItalicOn + "both" + biOff},
		// Inline code
		{"`code`", cyanOn + "code" + colorOff},
		{"run `ls -la` now", "run " + cyanOn + "ls -la" + colorOff + " now"},
		// No markdown
		{"plain text", "plain text"},
		// Mixed
		{"**bold** and *italic*", boldOn + "bold" + boldOff + " and " + italicOn + "italic" + italicOff},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := renderInline(tt.input)
			if got != tt.want {
				t.Errorf("renderInline(%q)\n got %q\nwant %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMarkdownWriter_Headers(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"# Title\n", "\033[1mTitle\033[0m\n"},
		{"## Subtitle\n", "\033[1mSubtitle\033[0m\n"},
		{"### Deep\n", "\033[1mDeep\033[0m\n"},
		// Header with inline bold
		{"# **Bold** heading\n", "\033[1m" + boldOn + "Bold" + boldOff + " heading\033[0m\n"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var buf bytes.Buffer
			w := NewMarkdownWriter(&buf)
			w.Write([]byte(tt.input))
			w.Flush()
			if buf.String() != tt.want {
				t.Errorf("header %q\n got %q\nwant %q", tt.input, buf.String(), tt.want)
			}
		})
	}
}

func TestMarkdownWriter_CodeBlock(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```\n"
	var buf bytes.Buffer
	w := NewMarkdownWriter(&buf)
	w.Write([]byte(input))
	w.Flush()

	got := buf.String()
	// Each line should be dimmed, no inline processing
	want := "\033[2m```go\033[0m\n" +
		"\033[2mfmt.Println(\"hello\")\033[0m\n" +
		"\033[2m```\033[0m\n"
	if got != want {
		t.Errorf("code block\n got %q\nwant %q", got, want)
	}
}

func TestMarkdownWriter_StreamedChunks(t *testing.T) {
	// Simulate streaming: bold text split across two Write calls
	var buf bytes.Buffer
	w := NewMarkdownWriter(&buf)
	w.Write([]byte("say **hel"))
	w.Write([]byte("lo** world\n"))
	w.Flush()

	want := "say " + boldOn + "hello" + boldOff + " world\n"
	if buf.String() != want {
		t.Errorf("streamed chunks\n got %q\nwant %q", buf.String(), want)
	}
}

func TestMarkdownWriter_Flush(t *testing.T) {
	// Partial line without trailing newline
	var buf bytes.Buffer
	w := NewMarkdownWriter(&buf)
	w.Write([]byte("**bold**"))
	w.Flush()

	want := boldOn + "bold" + boldOff
	if buf.String() != want {
		t.Errorf("flush partial\n got %q\nwant %q", buf.String(), want)
	}
}

func TestMarkdownWriter_PlainText(t *testing.T) {
	var buf bytes.Buffer
	w := NewMarkdownWriter(&buf)
	w.Write([]byte("no markdown here\n"))
	w.Flush()

	if buf.String() != "no markdown here\n" {
		t.Errorf("plain text altered: %q", buf.String())
	}
}
