package main

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// newTestEditor builds an editor over an in-memory buffer. No screen is
// created: these tests drive the key handling and never render.
func newTestEditor(lines ...string) *Editor {
	e := &Editor{
		buf:      &Buffer{Lines: append([]string{}, lines...)},
		themes:   DefaultThemes(),
		softWrap: true,
	}
	if len(e.buf.Lines) == 0 {
		e.buf.Lines = []string{""}
	}
	return e
}

func (e *Editor) text() string { return strings.Join(e.buf.Lines, "\n") }

func key(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, 0, tcell.ModNone) }
func ch(r rune) *tcell.EventKey       { return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone) }

// typeString feeds text in as if a person were typing it, newline by newline,
// which is exactly what an unbracketed paste looks like to the editor.
func (e *Editor) typeString(s string) {
	for _, r := range s {
		if r == '\n' {
			e.handleKey(key(tcell.KeyEnter))
		} else {
			e.handleKey(ch(r))
		}
	}
}

// pasteString plays a bracketed paste: start, the content as key events, end.
func (e *Editor) pasteString(s string) {
	e.beginPaste()
	for _, r := range s {
		switch r {
		case '\n':
			e.collectPaste(key(tcell.KeyEnter))
		case '\t':
			e.collectPaste(key(tcell.KeyTab))
		default:
			e.collectPaste(ch(r))
		}
	}
	e.endPaste()
}

// ── paste ────────────────────────────────────────────────────────────────────

// The bug this whole change exists for: without bracketed paste every newline
// in the pasted text runs through auto-indent, so each line inherits the indent
// of the one above it and the block walks off to the right.
func TestPasteKeepsItsOwnIndentation(t *testing.T) {
	const block = "func main() {\n    if ok {\n        run()\n    }\n}"

	typed := newTestEditor("")
	typed.typeString(block)
	if typed.text() == block {
		t.Fatal("expected typing the block to staircase — the test no longer proves anything")
	}

	pasted := newTestEditor("")
	pasted.pasteString(block)
	if got := pasted.text(); got != block {
		t.Errorf("paste changed the text:\n got:\n%s\nwant:\n%s", got, block)
	}
}

func TestPasteIntoIndentedPositionIsVerbatim(t *testing.T) {
	e := newTestEditor("        ")
	e.cursor = Pos{0, 8}
	e.pasteString("a = 1\nb = 2")

	want := "        a = 1\nb = 2"
	if got := e.text(); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestPasteTabsSurvive(t *testing.T) {
	e := newTestEditor("")
	e.pasteString("one\n\ttwo")
	if got := e.text(); got != "one\n\ttwo" {
		t.Errorf("tab was not preserved: %q", got)
	}
}

func TestPasteIsOneUndoStep(t *testing.T) {
	e := newTestEditor("start")
	e.cursor = Pos{0, 5}
	e.pasteString("\nline one\nline two\nline three")
	if e.buf.LineCount() != 4 {
		t.Fatalf("expected 4 lines after paste, got %d", e.buf.LineCount())
	}
	e.undo()
	if got := e.text(); got != "start" {
		t.Errorf("one undo should remove the whole paste, got %q", got)
	}
}

func TestPasteReplacesSelection(t *testing.T) {
	e := newTestEditor("keep DROP keep")
	e.selActive = true
	e.selAnchor = Pos{0, 5}
	e.cursor = Pos{0, 9}
	e.pasteString("NEW")
	if got := e.text(); got != "keep NEW keep" {
		t.Errorf("got %q", got)
	}
}

func TestPasteNormalisesCRLF(t *testing.T) {
	e := newTestEditor("")
	e.insertPasted("a\r\nb\rc")
	if got := e.text(); got != "a\nb\nc" {
		t.Errorf("got %q", got)
	}
}

// Keys arriving inside a paste are content, never commands: a pasted "q" must
// not be read as a shortcut.
func TestCollectPasteIgnoresNonContentKeys(t *testing.T) {
	e := newTestEditor("")
	e.beginPaste()
	e.collectPaste(ch('q'))
	e.collectPaste(key(tcell.KeyF1))
	e.collectPaste(key(tcell.KeyEnter))
	e.collectPaste(ch('x'))
	e.endPaste()
	if got := e.text(); got != "q\nx" {
		t.Errorf("got %q, want %q", got, "q\nx")
	}
}

// ── auto-indent ──────────────────────────────────────────────────────────────

func TestEnterCarriesIndent(t *testing.T) {
	e := newTestEditor("    value := 1")
	e.cursor = Pos{0, 14}
	e.handleKey(key(tcell.KeyEnter))
	if got := e.buf.Line(1); got != "    " {
		t.Errorf("got %q, want four spaces", got)
	}
}

func TestEnterOpensALevelAfterBrace(t *testing.T) {
	e := newTestEditor("    if ok {")
	e.cursor = Pos{0, 11}
	e.handleKey(key(tcell.KeyEnter))
	if got := e.buf.Line(1); got != "        " {
		t.Errorf("got %q, want eight spaces", got)
	}
	if e.cursor.Col != 8 {
		t.Errorf("cursor at col %d, want 8", e.cursor.Col)
	}
}

func TestEnterAfterColonOpensALevel(t *testing.T) {
	e := newTestEditor("def f():")
	e.cursor = Pos{0, 8}
	e.handleKey(key(tcell.KeyEnter))
	if got := e.buf.Line(1); got != "    " {
		t.Errorf("got %q, want four spaces", got)
	}
}

func TestEnterBetweenBracesParksTheCloser(t *testing.T) {
	e := newTestEditor("    if ok {}")
	e.cursor = Pos{0, 11} // between { and }
	e.handleKey(key(tcell.KeyEnter))

	want := []string{"    if ok {", "        ", "    }"}
	for i, w := range want {
		if got := e.buf.Line(i); got != w {
			t.Errorf("line %d = %q, want %q", i, got, w)
		}
	}
	if e.cursor.Line != 1 || e.cursor.Col != 8 {
		t.Errorf("cursor at %v, want line 1 col 8", e.cursor)
	}
}

func TestEnterFollowsTabbedFiles(t *testing.T) {
	e := newTestEditor("\tif ok {")
	e.cursor = Pos{0, 8}
	e.handleKey(key(tcell.KeyEnter))
	if got := e.buf.Line(1); got != "\t\t" {
		t.Errorf("got %q, want two tabs", got)
	}
}

// ── tab / shift+tab ──────────────────────────────────────────────────────────

func TestTabGoesToTheNextStop(t *testing.T) {
	e := newTestEditor("ab")
	e.cursor = Pos{0, 2}
	e.handleKey(key(tcell.KeyTab))
	if got := e.buf.Line(0); got != "ab  " {
		t.Errorf("got %q, want two spaces of padding to reach column 4", got)
	}
}

func TestTabIndentsSelectedBlock(t *testing.T) {
	e := newTestEditor("one", "two", "three")
	e.selActive = true
	e.selAnchor = Pos{0, 0}
	e.cursor = Pos{2, 3}
	e.handleKey(key(tcell.KeyTab))

	want := "    one\n    two\n    three"
	if got := e.text(); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
	if !e.selActive {
		t.Error("selection should survive indenting")
	}
}

func TestTabLeavesBlankLinesAlone(t *testing.T) {
	e := newTestEditor("one", "", "two")
	e.selActive = true
	e.selAnchor = Pos{0, 0}
	e.cursor = Pos{2, 3}
	e.handleKey(key(tcell.KeyTab))
	if got := e.buf.Line(1); got != "" {
		t.Errorf("blank line became %q", got)
	}
}

func TestShiftTabDedentsSelectedBlock(t *testing.T) {
	e := newTestEditor("    one", "    two")
	e.selActive = true
	e.selAnchor = Pos{0, 0}
	e.cursor = Pos{1, 7}
	e.handleKey(key(tcell.KeyBacktab))
	if got := e.text(); got != "one\ntwo" {
		t.Errorf("got %q", got)
	}
}

func TestShiftTabWithoutSelectionDedentsTheLine(t *testing.T) {
	e := newTestEditor("        deep")
	e.cursor = Pos{0, 12}
	e.handleKey(key(tcell.KeyBacktab))
	if got := e.buf.Line(0); got != "    deep" {
		t.Errorf("got %q", got)
	}
}

func TestShiftTabOnAnUnindentedLineDoesNothing(t *testing.T) {
	e := newTestEditor("flush")
	e.cursor = Pos{0, 5}
	e.handleKey(key(tcell.KeyBacktab))
	if got := e.buf.Line(0); got != "flush" {
		t.Errorf("got %q", got)
	}
}

// ── backspace ────────────────────────────────────────────────────────────────

func TestBackspaceInIndentRemovesALevel(t *testing.T) {
	e := newTestEditor("        x")
	e.cursor = Pos{0, 8}
	e.handleKey(key(tcell.KeyBackspace2))
	if got := e.buf.Line(0); got != "    x" {
		t.Errorf("got %q, want one level removed", got)
	}
	if e.cursor.Col != 4 {
		t.Errorf("cursor at %d, want 4", e.cursor.Col)
	}
}

func TestBackspaceInTextRemovesOneCharacter(t *testing.T) {
	e := newTestEditor("    word")
	e.cursor = Pos{0, 8}
	e.handleKey(key(tcell.KeyBackspace2))
	if got := e.buf.Line(0); got != "    wor" {
		t.Errorf("got %q", got)
	}
}

func TestBackspaceOffATabStopGoesToTheStop(t *testing.T) {
	e := newTestEditor("      x") // six spaces: cursor at 6 should land on 4
	e.cursor = Pos{0, 6}
	e.handleKey(key(tcell.KeyBackspace2))
	if got := e.buf.Line(0); got != "    x" {
		t.Errorf("got %q", got)
	}
}
