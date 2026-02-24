package shell

import (
	"testing"
)

func newTestEditor() *viEditor {
	h := NewHistory(20)
	return newViEditor(0, h)
}

func TestViInsertBasicTyping(t *testing.T) {
	vi := newTestEditor()

	// Type "hello" in insert mode
	for _, b := range []byte("hello") {
		done, _, err := vi.handleByte(b)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if done {
			t.Fatal("unexpected done")
		}
	}

	if string(vi.line) != "hello" {
		t.Errorf("expected 'hello', got %q", string(vi.line))
	}
	if vi.pos != 5 {
		t.Errorf("expected pos=5, got %d", vi.pos)
	}
}

func TestViInsertBackspace(t *testing.T) {
	vi := newTestEditor()

	for _, b := range []byte("hello") {
		vi.handleByte(b)
	}
	// Backspace
	vi.handleByte(0x7f)

	if string(vi.line) != "hell" {
		t.Errorf("expected 'hell', got %q", string(vi.line))
	}
	if vi.pos != 4 {
		t.Errorf("expected pos=4, got %d", vi.pos)
	}
}

func TestViEscapeToNormal(t *testing.T) {
	vi := newTestEditor()

	for _, b := range []byte("hello") {
		vi.handleByte(b)
	}
	// Escape to normal mode
	vi.handleByte(0x1b)

	if vi.mode != viNormal {
		t.Error("expected normal mode after Esc")
	}
	// Cursor should move back one (onto last char typed)
	if vi.pos != 4 {
		t.Errorf("expected pos=4, got %d", vi.pos)
	}
}

func TestViNormalMovement(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello world")
	vi.pos = 5
	vi.mode = viNormal

	// h: move left
	vi.handleByte('h')
	if vi.pos != 4 {
		t.Errorf("h: expected pos=4, got %d", vi.pos)
	}

	// l: move right
	vi.handleByte('l')
	if vi.pos != 5 {
		t.Errorf("l: expected pos=5, got %d", vi.pos)
	}

	// 0: beginning of line
	vi.handleByte('0')
	if vi.pos != 0 {
		t.Errorf("0: expected pos=0, got %d", vi.pos)
	}

	// $: end of line
	vi.handleByte('$')
	if vi.pos != 10 {
		t.Errorf("$: expected pos=10, got %d", vi.pos)
	}
}

func TestViNormalWordMotion(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello world test")
	vi.pos = 0
	vi.mode = viNormal

	// w: next word
	vi.handleByte('w')
	if vi.pos != 6 {
		t.Errorf("w: expected pos=6, got %d", vi.pos)
	}

	// b: previous word
	vi.handleByte('b')
	if vi.pos != 0 {
		t.Errorf("b: expected pos=0, got %d", vi.pos)
	}
}

func TestViNormalDeleteX(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello")
	vi.pos = 2
	vi.mode = viNormal

	// x: delete char at cursor
	vi.handleByte('x')
	if string(vi.line) != "helo" {
		t.Errorf("x: expected 'helo', got %q", string(vi.line))
	}
	if vi.pos != 2 {
		t.Errorf("x: expected pos=2, got %d", vi.pos)
	}
}

func TestViNormalDeleteDD(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello")
	vi.pos = 2
	vi.mode = viNormal

	// dd: delete entire line
	vi.handleByte('d')
	vi.handleByte('d')
	if string(vi.line) != "" {
		t.Errorf("dd: expected empty line, got %q", string(vi.line))
	}
	if vi.pos != 0 {
		t.Errorf("dd: expected pos=0, got %d", vi.pos)
	}
}

func TestViNormalInsertModes(t *testing.T) {
	tests := []struct {
		name    string
		key     byte
		initPos int
		lineLen int
		wantPos int
	}{
		{"i: insert at cursor", 'i', 3, 5, 3},
		{"I: insert at beginning", 'I', 3, 5, 0},
		{"a: append after cursor", 'a', 3, 5, 4},
		{"A: append at end", 'A', 3, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vi := newTestEditor()
			vi.line = []rune("hello")
			vi.pos = tt.initPos
			vi.mode = viNormal

			vi.handleByte(tt.key)
			if vi.mode != viInsert {
				t.Errorf("expected insert mode")
			}
			if vi.pos != tt.wantPos {
				t.Errorf("expected pos=%d, got %d", tt.wantPos, vi.pos)
			}
		})
	}
}

func TestViNormalDeleteDW(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello world")
	vi.pos = 0
	vi.mode = viNormal

	// dw: delete word
	vi.handleByte('d')
	vi.handleByte('w')
	if string(vi.line) != "world" {
		t.Errorf("dw: expected 'world', got %q", string(vi.line))
	}
}

func TestViNormalDeleteDDollar(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello world")
	vi.pos = 5
	vi.mode = viNormal

	// d$: delete to end
	vi.handleByte('d')
	vi.handleByte('$')
	if string(vi.line) != "hello" {
		t.Errorf("d$: expected 'hello', got %q", string(vi.line))
	}
}

func TestViNormalChangeCW(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello world")
	vi.pos = 0
	vi.mode = viNormal

	// cw: change word (deletes word, enters insert mode)
	vi.handleByte('c')
	vi.handleByte('w')
	if string(vi.line) != "world" {
		t.Errorf("cw: expected 'world', got %q", string(vi.line))
	}
	if vi.mode != viInsert {
		t.Error("cw: expected insert mode")
	}
}

func TestViNormalReplaceR(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello")
	vi.pos = 0
	vi.mode = viNormal

	// rx: replace first char with 'x'
	vi.handleByte('r')
	vi.handleByte('x')
	if string(vi.line) != "xello" {
		t.Errorf("r: expected 'xello', got %q", string(vi.line))
	}
	if vi.mode != viNormal {
		t.Error("r: should stay in normal mode")
	}
}

func TestViEnterReturnsLine(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello")
	vi.pos = 5
	vi.mode = viInsert

	done, line, err := vi.handleByte('\r')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !done {
		t.Fatal("expected done on Enter")
	}
	if line != "hello" {
		t.Errorf("expected 'hello', got %q", line)
	}
}

func TestViModeIndicator(t *testing.T) {
	vi := newTestEditor()

	vi.mode = viInsert
	ind := vi.modeIndicator()
	if ind == "" {
		t.Error("insert mode indicator should not be empty")
	}

	vi.mode = viNormal
	ind = vi.modeIndicator()
	if ind == "" {
		t.Error("normal mode indicator should not be empty")
	}
}

func TestViHistory(t *testing.T) {
	vi := newTestEditor()
	vi.history.Add("first")
	vi.history.Add("second")
	vi.history.Add("third")

	vi.line = []rune("current")
	vi.pos = 7

	// Navigate to previous entry
	vi.historyPrev()
	if string(vi.line) != "third" {
		t.Errorf("expected 'third', got %q", string(vi.line))
	}

	vi.historyPrev()
	if string(vi.line) != "second" {
		t.Errorf("expected 'second', got %q", string(vi.line))
	}

	// Navigate forward
	vi.historyNext()
	if string(vi.line) != "third" {
		t.Errorf("expected 'third', got %q", string(vi.line))
	}

	// Navigate back to the live line
	vi.historyNext()
	if string(vi.line) != "current" {
		t.Errorf("expected 'current', got %q", string(vi.line))
	}
}

func TestViCtrlC(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello")
	vi.pos = 3

	done, _, _ := vi.handleByte(0x03)
	if done {
		t.Error("Ctrl-C should not signal done")
	}
	if len(vi.line) != 0 {
		t.Errorf("Ctrl-C should clear line, got %q", string(vi.line))
	}
	if vi.pos != 0 {
		t.Errorf("Ctrl-C should reset pos, got %d", vi.pos)
	}
}

func TestViCtrlW(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello world")
	vi.pos = 11

	vi.handleByte(0x17) // Ctrl-W
	if string(vi.line) != "hello " {
		t.Errorf("Ctrl-W: expected 'hello ', got %q", string(vi.line))
	}
}

func TestViNormalSubstituteS(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello")
	vi.pos = 2
	vi.mode = viNormal

	vi.handleByte('s')
	if string(vi.line) != "helo" {
		t.Errorf("s: expected 'helo', got %q", string(vi.line))
	}
	if vi.mode != viInsert {
		t.Error("s: expected insert mode")
	}
}

func TestViNormalSubstituteLineSS(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello")
	vi.pos = 2
	vi.mode = viNormal

	vi.handleByte('S')
	if string(vi.line) != "" {
		t.Errorf("S: expected empty, got %q", string(vi.line))
	}
	if vi.mode != viInsert {
		t.Error("S: expected insert mode")
	}
}

func TestViNormalDeleteBigD(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello world")
	vi.pos = 5
	vi.mode = viNormal

	vi.handleByte('D')
	if string(vi.line) != "hello" {
		t.Errorf("D: expected 'hello', got %q", string(vi.line))
	}
}

func TestViNormalChangeBigC(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello world")
	vi.pos = 5
	vi.mode = viNormal

	vi.handleByte('C')
	if string(vi.line) != "hello" {
		t.Errorf("C: expected 'hello', got %q", string(vi.line))
	}
	if vi.mode != viInsert {
		t.Error("C: expected insert mode")
	}
}

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"\033[1;34mhello\033[0m", "hello"},
		{"\033[1;34magent-sh\033[0m ~/dir\033[1;34m>\033[0m ", "agent-sh ~/dir> "},
	}
	for _, tt := range tests {
		got := stripAnsi(tt.input)
		if got != tt.want {
			t.Errorf("stripAnsi(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestViEscapeSequences(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello")
	vi.pos = 2

	// Right arrow
	vi.handleEscapeSeq('C')
	if vi.pos != 3 {
		t.Errorf("Right arrow: expected pos=3, got %d", vi.pos)
	}

	// Left arrow
	vi.handleEscapeSeq('D')
	if vi.pos != 2 {
		t.Errorf("Left arrow: expected pos=2, got %d", vi.pos)
	}

	// Home
	vi.handleEscapeSeq('H')
	if vi.pos != 0 {
		t.Errorf("Home: expected pos=0, got %d", vi.pos)
	}

	// End
	vi.handleEscapeSeq('F')
	if vi.pos != 5 {
		t.Errorf("End: expected pos=5, got %d", vi.pos)
	}
}

func TestViDeleteXAtEnd(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("ab")
	vi.pos = 1
	vi.mode = viNormal

	// Delete last char, cursor should move back
	vi.handleByte('x')
	if string(vi.line) != "a" {
		t.Errorf("expected 'a', got %q", string(vi.line))
	}
	if vi.pos != 0 {
		t.Errorf("expected pos=0, got %d", vi.pos)
	}
}

func TestViNormalDeleteDB(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello world")
	vi.pos = 6
	vi.mode = viNormal

	// db: delete backward word
	vi.handleByte('d')
	vi.handleByte('b')
	if string(vi.line) != "world" {
		t.Errorf("db: expected 'world', got %q", string(vi.line))
	}
}

func TestViNormalDeleteD0(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello world")
	vi.pos = 6
	vi.mode = viNormal

	// d0: delete to beginning
	vi.handleByte('d')
	vi.handleByte('0')
	if string(vi.line) != "world" {
		t.Errorf("d0: expected 'world', got %q", string(vi.line))
	}
	if vi.pos != 0 {
		t.Errorf("d0: expected pos=0, got %d", vi.pos)
	}
}

func TestViNormalCC(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello")
	vi.pos = 3
	vi.mode = viNormal

	vi.handleByte('c')
	vi.handleByte('c')
	if string(vi.line) != "" {
		t.Errorf("cc: expected empty, got %q", string(vi.line))
	}
	if vi.mode != viInsert {
		t.Error("cc: expected insert mode")
	}
}

func TestViNormalCaret(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("  hello")
	vi.pos = 6
	vi.mode = viNormal

	vi.handleByte('^')
	if vi.pos != 2 {
		t.Errorf("^: expected pos=2, got %d", vi.pos)
	}
}

func TestViNormalBigX(t *testing.T) {
	vi := newTestEditor()
	vi.line = []rune("hello")
	vi.pos = 3
	vi.mode = viNormal

	vi.handleByte('X')
	if string(vi.line) != "helo" {
		t.Errorf("X: expected 'helo', got %q", string(vi.line))
	}
	if vi.pos != 2 {
		t.Errorf("X: expected pos=2, got %d", vi.pos)
	}
}
