package main

import (
	"os"
	"strings"
)

// Pos is a line+column cursor position (both zero-based, col is rune-indexed).
type Pos struct {
	Line int
	Col  int
}

func (p Pos) Before(other Pos) bool {
	if p.Line != other.Line {
		return p.Line < other.Line
	}
	return p.Col < other.Col
}

// Buffer holds the in-memory text and file metadata.
type Buffer struct {
	Lines    []string
	Filename string
	Modified bool
	CRLF     bool // true if the file originally used \r\n line endings
}

func NewBuffer(filename string) (*Buffer, error) {
	b := &Buffer{
		Lines:    []string{""},
		Filename: filename,
	}
	if filename == "" {
		return b, nil
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return b, nil
		}
		return nil, err
	}
	text := string(data)
	if strings.Contains(text, "\r\n") {
		b.CRLF = true
		text = strings.ReplaceAll(text, "\r\n", "\n")
	}
	text = strings.TrimRight(text, "\n")
	b.Lines = strings.Split(text, "\n")
	if len(b.Lines) == 0 {
		b.Lines = []string{""}
	}
	return b, nil
}

func (b *Buffer) lineEnding() string {
	if b.CRLF {
		return "\r\n"
	}
	return "\n"
}

func (b *Buffer) Save() error {
	le := b.lineEnding()
	content := strings.Join(b.Lines, le) + le
	if err := os.WriteFile(b.Filename, []byte(content), 0644); err != nil {
		return err
	}
	b.Modified = false
	return nil
}

func (b *Buffer) LineCount() int { return len(b.Lines) }

func (b *Buffer) Line(i int) string {
	if i < 0 || i >= len(b.Lines) {
		return ""
	}
	return b.Lines[i]
}

func runesOf(s string) []rune { return []rune(s) }

// InsertRune inserts ch at (line, col) and returns the new col.
func (b *Buffer) InsertRune(line, col int, ch rune) int {
	if line >= len(b.Lines) {
		return col
	}
	r := runesOf(b.Lines[line])
	if col > len(r) {
		col = len(r)
	}
	nr := make([]rune, len(r)+1)
	copy(nr, r[:col])
	nr[col] = ch
	copy(nr[col+1:], r[col:])
	b.Lines[line] = string(nr)
	b.Modified = true
	return col + 1
}

// Backspace removes the character before (line, col). Returns new position.
func (b *Buffer) Backspace(line, col int) (int, int) {
	if col > 0 {
		r := runesOf(b.Lines[line])
		if col > len(r) {
			col = len(r)
		}
		b.Lines[line] = string(r[:col-1]) + string(r[col:])
		b.Modified = true
		return line, col - 1
	}
	if line > 0 {
		prevLen := len(runesOf(b.Lines[line-1]))
		b.Lines[line-1] += b.Lines[line]
		b.Lines = append(b.Lines[:line], b.Lines[line+1:]...)
		b.Modified = true
		return line - 1, prevLen
	}
	return line, col
}

// Delete removes the character at (line, col).
func (b *Buffer) Delete(line, col int) {
	if line >= len(b.Lines) {
		return
	}
	r := runesOf(b.Lines[line])
	if col < len(r) {
		b.Lines[line] = string(r[:col]) + string(r[col+1:])
		b.Modified = true
	} else if line < len(b.Lines)-1 {
		b.Lines[line] += b.Lines[line+1]
		b.Lines = append(b.Lines[:line+1], b.Lines[line+2:]...)
		b.Modified = true
	}
}

// NewLine splits the line at col, returning the new cursor position.
func (b *Buffer) NewLine(line, col int) (int, int) {
	if line >= len(b.Lines) {
		return line, col
	}
	r := runesOf(b.Lines[line])
	if col > len(r) {
		col = len(r)
	}
	before, after := string(r[:col]), string(r[col:])
	b.Lines[line] = before
	newLines := make([]string, len(b.Lines)+1)
	copy(newLines, b.Lines[:line+1])
	newLines[line+1] = after
	copy(newLines[line+2:], b.Lines[line+1:])
	b.Lines = newLines
	b.Modified = true
	return line + 1, 0
}

// GetRange returns the text between from and to (exclusive).
func (b *Buffer) GetRange(from, to Pos) string {
	if from == to {
		return ""
	}
	if from.Line == to.Line {
		r := runesOf(b.Lines[from.Line])
		fc := clampCol(from.Col, len(r))
		tc := clampCol(to.Col, len(r))
		return string(r[fc:tc])
	}
	var sb strings.Builder
	fr := runesOf(b.Lines[from.Line])
	sb.WriteString(string(fr[clampCol(from.Col, len(fr)):]))
	for i := from.Line + 1; i < to.Line; i++ {
		sb.WriteByte('\n')
		sb.WriteString(b.Lines[i])
	}
	sb.WriteByte('\n')
	lr := runesOf(b.Lines[to.Line])
	sb.WriteString(string(lr[:clampCol(to.Col, len(lr))]))
	return sb.String()
}

// DeleteRange removes text from→to and returns the resulting cursor position.
func (b *Buffer) DeleteRange(from, to Pos) Pos {
	if from == to {
		return from
	}
	if from.Line == to.Line {
		r := runesOf(b.Lines[from.Line])
		fc := clampCol(from.Col, len(r))
		tc := clampCol(to.Col, len(r))
		b.Lines[from.Line] = string(r[:fc]) + string(r[tc:])
		b.Modified = true
		return from
	}
	fr := runesOf(b.Lines[from.Line])
	lr := runesOf(b.Lines[to.Line])
	fc := clampCol(from.Col, len(fr))
	tc := clampCol(to.Col, len(lr))
	combined := string(fr[:fc]) + string(lr[tc:])
	newLines := make([]string, 0, len(b.Lines)-(to.Line-from.Line))
	newLines = append(newLines, b.Lines[:from.Line]...)
	newLines = append(newLines, combined)
	if to.Line+1 < len(b.Lines) {
		newLines = append(newLines, b.Lines[to.Line+1:]...)
	}
	b.Lines = newLines
	b.Modified = true
	return from
}

// InsertText inserts a (possibly multi-line) string and returns the new cursor position.
func (b *Buffer) InsertText(line, col int, text string) (int, int) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	parts := strings.Split(text, "\n")

	if len(parts) == 1 {
		r := runesOf(b.Lines[line])
		if col > len(r) {
			col = len(r)
		}
		b.Lines[line] = string(r[:col]) + parts[0] + string(r[col:])
		b.Modified = true
		return line, col + len(runesOf(parts[0]))
	}

	r := runesOf(b.Lines[line])
	if col > len(r) {
		col = len(r)
	}
	before := string(r[:col])
	after := string(r[col:])

	newLines := make([]string, 0, len(b.Lines)+len(parts)-1)
	newLines = append(newLines, b.Lines[:line]...)
	newLines = append(newLines, before+parts[0])
	for _, p := range parts[1 : len(parts)-1] {
		newLines = append(newLines, p)
	}
	last := parts[len(parts)-1]
	newLines = append(newLines, last+after)
	if line+1 < len(b.Lines) {
		newLines = append(newLines, b.Lines[line+1:]...)
	}
	b.Lines = newLines
	b.Modified = true
	return line + len(parts) - 1, len(runesOf(last))
}

func clampCol(col, length int) int {
	if col < 0 {
		return 0
	}
	if col > length {
		return length
	}
	return col
}
