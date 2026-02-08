package render

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// MarkdownWriter buffers streamed text line-by-line and writes
// ANSI-styled output for markdown formatting (bold, italic, headers,
// inline code, fenced code blocks).
type MarkdownWriter struct {
	w      io.Writer
	buf    strings.Builder
	inCode bool // inside a ``` fenced code block
}

// NewMarkdownWriter creates a MarkdownWriter that writes styled output to w.
func NewMarkdownWriter(w io.Writer) *MarkdownWriter {
	return &MarkdownWriter{w: w}
}

// Write accepts streamed text chunks, buffers until complete lines, and
// renders each line with ANSI formatting.
func (m *MarkdownWriter) Write(p []byte) (int, error) {
	n := len(p)
	m.buf.Write(p)

	for {
		text := m.buf.String()
		idx := strings.Index(text, "\n")
		if idx == -1 {
			break
		}
		line := text[:idx]
		m.buf.Reset()
		m.buf.WriteString(text[idx+1:])

		rendered := m.renderLine(line)
		if _, err := io.WriteString(m.w, rendered+"\n"); err != nil {
			return n, err
		}
	}

	return n, nil
}

// Flush writes any remaining buffered text (the last partial line).
func (m *MarkdownWriter) Flush() error {
	if m.buf.Len() > 0 {
		line := m.buf.String()
		m.buf.Reset()
		rendered := m.renderLine(line)
		_, err := io.WriteString(m.w, rendered)
		return err
	}
	return nil
}

func (m *MarkdownWriter) renderLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// Code fence toggle.
	if strings.HasPrefix(trimmed, "```") {
		m.inCode = !m.inCode
		return "\033[2m" + line + "\033[0m"
	}

	// Inside code block — dim, no markdown processing.
	if m.inCode {
		return "\033[2m" + line + "\033[0m"
	}

	// Headers: # … → bold.
	if strings.HasPrefix(trimmed, "#") {
		i := 0
		for i < len(trimmed) && trimmed[i] == '#' {
			i++
		}
		if i <= 6 && i < len(trimmed) && trimmed[i] == ' ' {
			heading := trimmed[i+1:]
			return "\033[1m" + renderInline(heading) + "\033[0m"
		}
	}

	return renderInline(line)
}

// ANSI escape sequences.
const (
	boldOn       = "\033[1m"
	boldOff      = "\033[22m"
	italicOn     = "\033[3m"
	italicOff    = "\033[23m"
	cyanOn       = "\033[36m"
	colorOff     = "\033[39m"
	boldItalicOn = "\033[1;3m"
	biOff        = "\033[22;23m"
)

var (
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reBoldItalic = regexp.MustCompile(`\*\*\*(.+?)\*\*\*`)
	reBold       = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reItalic     = regexp.MustCompile(`\*([^*]+?)\*`)
)

// renderInline applies inline markdown formatting (code, bold, italic).
// Code spans are extracted first and replaced with placeholders so their
// contents are never touched by the bold/italic regexps.
func renderInline(s string) string {
	// Extract inline code spans into placeholders.
	var codeSpans []string
	s = reInlineCode.ReplaceAllStringFunc(s, func(m string) string {
		inner := reInlineCode.FindStringSubmatch(m)[1]
		idx := len(codeSpans)
		codeSpans = append(codeSpans, cyanOn+inner+colorOff)
		return fmt.Sprintf("\x00CODE%d\x00", idx)
	})

	// Bold/italic on the placeholder-safe text.
	s = reBoldItalic.ReplaceAllString(s, boldItalicOn+"$1"+biOff)
	s = reBold.ReplaceAllString(s, boldOn+"$1"+boldOff)
	s = reItalic.ReplaceAllString(s, italicOn+"$1"+italicOff)

	// Restore code spans.
	for i, span := range codeSpans {
		s = strings.Replace(s, fmt.Sprintf("\x00CODE%d\x00", i), span, 1)
	}
	return s
}
