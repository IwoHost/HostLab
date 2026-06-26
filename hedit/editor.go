package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
)

type Mode int

const (
	ModeNormal Mode = iota
	ModeFind
	ModeSettings
	ModeConfirmQuit  // [S]Save  [Q]Quit  [N]Cancel
	ModeReplaceFind
	ModeReplaceWith
	ModeSaveAs       // typing filename
	ModeSaveConflict // file exists: [O]verwrite  [C]opy  [N]Cancel
)

type snapshot struct {
	lines    []string
	cursor   Pos
	modified bool
}

// Editor is the full editor state.
type Editor struct {
	buf    *Buffer
	screen tcell.Screen

	cursor    Pos
	selActive bool
	selAnchor Pos

	scrollLine int
	scrollCol  int

	internalClip string

	mode Mode

	findStr string
	findPos []Pos
	findIdx int

	settingsIdx int
	themes      []*Theme
	themeIdx    int

	message string
	msgErr  bool

	width  int
	height int

	undoStack     []snapshot
	redoStack     []snapshot
	lastWasInsert bool

	replaceFind  string
	replaceWith  string
	replaceScope int // 0=selection, 1=line

	saveInput   string
	pendingQuit bool

	dragging bool
}

func NewEditor(filename string) (*Editor, error) {
	buf, err := NewBuffer(filename)
	if err != nil {
		return nil, err
	}
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}
	screen.EnableMouse(tcell.MouseButtonEvents | tcell.MouseMotionEvents)
	screen.SetStyle(tcell.StyleDefault)

	themes := DefaultThemes()
	e := &Editor{
		buf:    buf,
		screen: screen,
		themes: themes,
	}
	e.loadConfig()
	return e, nil
}

func (e *Editor) Run() error {
	defer e.screen.Fini()

	e.width, e.height = e.screen.Size()
	e.render()

	for {
		ev := e.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			e.width, e.height = ev.Size()
			e.screen.Sync()
			e.render()

		case *tcell.EventKey:
			quit := e.handleKey(ev)
			if quit {
				return nil
			}
			e.ensureVisible()
			e.render()

		case *tcell.EventMouse:
			quit := e.handleMouse(ev)
			if quit {
				return nil
			}
			e.ensureVisible()
			e.render()
		}
	}
}

// ── KEY HANDLING ────────────────────────────────────────────────────────────

func (e *Editor) handleKey(ev *tcell.EventKey) (quit bool) {
	mod := ev.Modifiers()
	key := ev.Key()
	ch := ev.Rune()
	isAlt := mod&tcell.ModAlt != 0

	switch e.mode {
	case ModeFind:
		return e.handleFindKey(key, ch)
	case ModeSettings:
		return e.handleSettingsKey(key)
	case ModeConfirmQuit:
		return e.handleConfirmKey(key, ch)
	case ModeReplaceFind:
		return e.handleReplaceFindKey(key, ch)
	case ModeReplaceWith:
		return e.handleReplaceWithKey(key, ch)
	case ModeSaveAs:
		return e.handleSaveAsKey(key, ch)
	case ModeSaveConflict:
		return e.handleSaveConflictKey(key, ch)
	}

	// ── Normal mode ──
	switch {
	// Alt+A → select current line
	case isAlt && key == tcell.KeyRune && (ch == 'a' || ch == 'A'):
		e.selectLine()

	// Alt+C → copy entire line
	case isAlt && key == tcell.KeyRune && (ch == 'c' || ch == 'C'):
		e.copyLine()

	// Alt+X → cut entire line
	case isAlt && key == tcell.KeyRune && (ch == 'x' || ch == 'X'):
		e.cutLine()

	// Alt+R → replace on current line
	case isAlt && key == tcell.KeyRune && (ch == 'r' || ch == 'R'):
		e.mode = ModeReplaceFind
		e.replaceFind = ""
		e.replaceWith = ""
		e.replaceScope = 1
		e.clearMessage()

	// Ctrl+Z → undo
	case key == tcell.KeyCtrlZ:
		e.undo()

	// Ctrl+Y → redo
	case key == tcell.KeyCtrlY:
		e.redo()

	// Ctrl+S → save (with filename prompt)
	case key == tcell.KeyCtrlS:
		e.startSave(false)

	// Ctrl+Q → quit
	case key == tcell.KeyCtrlQ:
		if e.buf.Modified {
			e.mode = ModeConfirmQuit
		} else {
			return true
		}

	// Ctrl+A → select all
	case key == tcell.KeyCtrlA:
		e.selectAll()

	// Ctrl+C → copy selection
	case key == tcell.KeyCtrlC:
		e.copySelection()

	// Ctrl+X → cut selection
	case key == tcell.KeyCtrlX:
		e.cutSelection()

	// Ctrl+V → paste
	case key == tcell.KeyCtrlV:
		e.pushUndo()
		e.paste()
		e.clearMessage()

	// Ctrl+F → find
	case key == tcell.KeyCtrlF:
		e.mode = ModeFind
		e.findStr = ""
		e.findPos = nil
		e.clearMessage()

	// Ctrl+R → replace in selection
	case key == tcell.KeyCtrlR:
		e.mode = ModeReplaceFind
		e.replaceFind = ""
		e.replaceWith = ""
		e.replaceScope = 0
		e.clearMessage()

	// Ctrl+U → settings
	case key == tcell.KeyCtrlU:
		e.mode = ModeSettings
		e.settingsIdx = e.themeIdx
		e.clearMessage()

	// Ctrl+Left → word left
	case key == tcell.KeyLeft && mod&tcell.ModCtrl != 0:
		e.wordLeft(mod&tcell.ModShift != 0)

	// Ctrl+Right → word right
	case key == tcell.KeyRight && mod&tcell.ModCtrl != 0:
		e.wordRight(mod&tcell.ModShift != 0)

	// Arrow keys (with optional Shift for selection) — Ctrl variants MUST come first
	case key == tcell.KeyUp:
		e.moveUp(mod&tcell.ModShift != 0)
	case key == tcell.KeyDown:
		e.moveDown(mod&tcell.ModShift != 0)
	case key == tcell.KeyLeft:
		e.moveLeft(mod&tcell.ModShift != 0)
	case key == tcell.KeyRight:
		e.moveRight(mod&tcell.ModShift != 0)

	case key == tcell.KeyHome:
		if mod&tcell.ModCtrl != 0 {
			e.moveCursor(0, 0, mod&tcell.ModShift != 0)
		} else {
			e.moveToLineStart(mod&tcell.ModShift != 0)
		}
	case key == tcell.KeyEnd:
		if mod&tcell.ModCtrl != 0 {
			last := e.buf.LineCount() - 1
			e.moveCursor(last, len(runesOf(e.buf.Line(last))), mod&tcell.ModShift != 0)
		} else {
			e.moveToLineEnd(mod&tcell.ModShift != 0)
		}

	case key == tcell.KeyPgUp:
		contentRows := e.contentRows()
		newLine := e.cursor.Line - contentRows
		if newLine < 0 {
			newLine = 0
		}
		e.moveCursor(newLine, e.cursor.Col, false)

	case key == tcell.KeyPgDn:
		contentRows := e.contentRows()
		newLine := e.cursor.Line + contentRows
		if newLine >= e.buf.LineCount() {
			newLine = e.buf.LineCount() - 1
		}
		e.moveCursor(newLine, e.cursor.Col, false)

	// Backspace
	case key == tcell.KeyBackspace || key == tcell.KeyBackspace2:
		e.pushUndo()
		if e.selActive {
			e.deleteSelection()
		} else {
			nl, nc := e.buf.Backspace(e.cursor.Line, e.cursor.Col)
			e.cursor = Pos{nl, nc}
			e.clearMessage()
		}

	// Delete
	case key == tcell.KeyDelete:
		e.pushUndo()
		if e.selActive {
			e.deleteSelection()
		} else {
			e.buf.Delete(e.cursor.Line, e.cursor.Col)
			e.clearMessage()
		}

	// Enter — with auto-indent
	case key == tcell.KeyEnter:
		e.pushUndo()
		indent := leadingWhitespace(e.buf.Line(e.cursor.Line))
		if e.selActive {
			e.deleteSelection()
		}
		nl, nc := e.buf.NewLine(e.cursor.Line, e.cursor.Col)
		e.cursor = Pos{nl, nc}
		if indent != "" {
			_, nc2 := e.buf.InsertText(e.cursor.Line, 0, indent)
			e.cursor = Pos{e.cursor.Line, nc2}
		}
		e.clearMessage()

	// Tab → 4 spaces
	case key == tcell.KeyTab:
		e.pushUndo()
		if e.selActive {
			e.deleteSelection()
		}
		for i := 0; i < 4; i++ {
			nc := e.buf.InsertRune(e.cursor.Line, e.cursor.Col, ' ')
			e.cursor.Col = nc
		}
		e.clearMessage()

	// Escape → clear selection / message
	case key == tcell.KeyEscape:
		e.selActive = false
		e.clearMessage()

	// Printable character — consecutive inserts are batched into one undo step
	case key == tcell.KeyRune && !isAlt && unicode.IsPrint(ch):
		if !e.lastWasInsert || e.selActive {
			e.pushUndo()
		} else {
			// Still clear redo so you can't redo after new typing
			e.redoStack = e.redoStack[:0]
		}
		e.lastWasInsert = true
		if e.selActive {
			e.deleteSelection()
		}
		nc := e.buf.InsertRune(e.cursor.Line, e.cursor.Col, ch)
		e.cursor.Col = nc
		e.clearMessage()
	}

	return false
}

func (e *Editor) handleFindKey(key tcell.Key, ch rune) bool {
	switch key {
	case tcell.KeyEscape, tcell.KeyCtrlF:
		e.mode = ModeNormal
		e.findPos = nil
	case tcell.KeyEnter, tcell.KeyDown:
		e.findNext()
	case tcell.KeyUp:
		e.findPrev()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(e.findStr) > 0 {
			r := runesOf(e.findStr)
			e.findStr = string(r[:len(r)-1])
			e.updateFind()
		}
	case tcell.KeyRune:
		if unicode.IsPrint(ch) {
			e.findStr += string(ch)
			e.updateFind()
		}
	}
	return false
}

func (e *Editor) handleSettingsKey(key tcell.Key) bool {
	switch key {
	case tcell.KeyUp:
		if e.settingsIdx > 0 {
			e.settingsIdx--
		}
	case tcell.KeyDown:
		if e.settingsIdx < len(e.themes)-1 {
			e.settingsIdx++
		}
	case tcell.KeyEnter:
		e.themeIdx = e.settingsIdx
		e.mode = ModeNormal
		e.saveConfig()
		e.showMessage("Theme applied: " + e.themes[e.themeIdx].Name)
	case tcell.KeyEscape, tcell.KeyCtrlU:
		e.mode = ModeNormal
	}
	return false
}

func (e *Editor) handleConfirmKey(key tcell.Key, ch rune) bool {
	switch {
	case key == tcell.KeyRune && (ch == 's' || ch == 'S'):
		e.startSave(true) // save then quit
	case key == tcell.KeyRune && (ch == 'q' || ch == 'Q'):
		return true // quit without saving
	default:
		e.mode = ModeNormal
		e.clearMessage()
	}
	return false
}

func (e *Editor) handleReplaceFindKey(key tcell.Key, ch rune) bool {
	switch key {
	case tcell.KeyEscape:
		e.mode = ModeNormal
		e.clearMessage()
	case tcell.KeyEnter:
		if e.replaceFind != "" {
			e.mode = ModeReplaceWith
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(e.replaceFind) > 0 {
			r := runesOf(e.replaceFind)
			e.replaceFind = string(r[:len(r)-1])
		}
	case tcell.KeyRune:
		if unicode.IsPrint(ch) {
			e.replaceFind += string(ch)
		}
	}
	return false
}

func (e *Editor) handleReplaceWithKey(key tcell.Key, ch rune) bool {
	switch key {
	case tcell.KeyEscape:
		e.mode = ModeNormal
		e.clearMessage()
	case tcell.KeyEnter:
		e.performReplace()
		e.mode = ModeNormal
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(e.replaceWith) > 0 {
			r := runesOf(e.replaceWith)
			e.replaceWith = string(r[:len(r)-1])
		}
	case tcell.KeyRune:
		if unicode.IsPrint(ch) {
			e.replaceWith += string(ch)
		}
	}
	return false
}

func (e *Editor) handleSaveAsKey(key tcell.Key, ch rune) bool {
	switch key {
	case tcell.KeyEscape:
		e.pendingQuit = false
		e.mode = ModeNormal
		e.clearMessage()
	case tcell.KeyEnter:
		name := strings.TrimSpace(e.saveInput)
		if name == "" {
			e.showError("Filename required")
			return false
		}
		if name == e.buf.Filename {
			return e.commitSave(name, true)
		}
		if _, err := os.Stat(name); err == nil {
			e.mode = ModeSaveConflict
		} else {
			return e.commitSave(name, true)
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(e.saveInput) > 0 {
			r := runesOf(e.saveInput)
			e.saveInput = string(r[:len(r)-1])
		}
	case tcell.KeyRune:
		if unicode.IsPrint(ch) {
			e.saveInput += string(ch)
		}
	}
	return false
}

func (e *Editor) handleSaveConflictKey(key tcell.Key, ch rune) bool {
	switch {
	case key == tcell.KeyRune && (ch == 'o' || ch == 'O'):
		return e.commitSave(e.saveInput, true)
	case key == tcell.KeyRune && (ch == 'c' || ch == 'C'):
		return e.commitSave(e.saveInput, false)
	case key == tcell.KeyRune && (ch == 'n' || ch == 'N'), key == tcell.KeyEscape:
		e.mode = ModeSaveAs
	}
	return false
}

func (e *Editor) startSave(quitAfter bool) {
	e.pendingQuit = quitAfter
	e.saveInput = e.buf.Filename
	e.mode = ModeSaveAs
	e.clearMessage()
}

func (e *Editor) commitSave(filename string, updateFilename bool) bool {
	content := strings.Join(e.buf.Lines, "\n") + "\n"
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		e.showError(fmt.Sprintf("Save failed: %v", err))
		e.mode = ModeNormal
		return false
	}
	if updateFilename {
		e.buf.Filename = filename
		e.buf.Modified = false
		e.showMessage(fmt.Sprintf("Saved → %s", filename))
	} else {
		e.showMessage(fmt.Sprintf("Copy saved → %s", filename))
	}
	e.mode = ModeNormal
	return e.pendingQuit
}

// ── MOUSE HANDLING ───────────────────────────────────────────────────────────

func (e *Editor) handleMouse(ev *tcell.EventMouse) (quit bool) {
	x, y := ev.Position()
	btn := ev.Buttons()
	gutterW := e.gutterWidth()

	if y < 1 || y > e.height-4 || x < gutterW {
		if btn == 0 {
			e.dragging = false
		}
		return false
	}

	line := y - 1 + e.scrollLine
	col := x - gutterW + e.scrollCol
	if line >= e.buf.LineCount() {
		line = e.buf.LineCount() - 1
	}
	lineLen := len(runesOf(e.buf.Line(line)))
	if col < 0 {
		col = 0
	}
	if col > lineLen {
		col = lineLen
	}

	switch btn {
	case tcell.Button1:
		if ev.Modifiers()&tcell.ModShift != 0 || e.dragging {
			e.moveCursor(line, col, true)
		} else {
			e.selActive = false
			e.cursor = Pos{line, col}
			e.dragging = true
		}
	case 0:
		e.dragging = false
	case tcell.WheelUp:
		e.scrollLine -= 3
		if e.scrollLine < 0 {
			e.scrollLine = 0
		}
	case tcell.WheelDown:
		maxScroll := e.buf.LineCount() - 1
		e.scrollLine += 3
		if e.scrollLine > maxScroll {
			e.scrollLine = maxScroll
		}
	}
	return false
}

// ── UNDO / REDO ──────────────────────────────────────────────────────────────

func (e *Editor) pushUndo() {
	lines := make([]string, len(e.buf.Lines))
	copy(lines, e.buf.Lines)
	e.undoStack = append(e.undoStack, snapshot{lines, e.cursor, e.buf.Modified})
	if len(e.undoStack) > 200 {
		e.undoStack = e.undoStack[1:]
	}
	e.redoStack = e.redoStack[:0]
	e.lastWasInsert = false
}

func (e *Editor) undo() {
	if len(e.undoStack) == 0 {
		e.showMessage("Nothing to undo")
		return
	}
	e.lastWasInsert = false
	lines := make([]string, len(e.buf.Lines))
	copy(lines, e.buf.Lines)
	e.redoStack = append(e.redoStack, snapshot{lines, e.cursor, e.buf.Modified})
	s := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	e.buf.Lines = s.lines
	e.buf.Modified = s.modified
	e.cursor = s.cursor
	e.selActive = false
	e.showMessage("Undo")
}

func (e *Editor) redo() {
	if len(e.redoStack) == 0 {
		e.showMessage("Nothing to redo")
		return
	}
	e.lastWasInsert = false
	lines := make([]string, len(e.buf.Lines))
	copy(lines, e.buf.Lines)
	e.undoStack = append(e.undoStack, snapshot{lines, e.cursor, e.buf.Modified})
	if len(e.undoStack) > 200 {
		e.undoStack = e.undoStack[1:]
	}
	s := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	e.buf.Lines = s.lines
	e.buf.Modified = s.modified
	e.cursor = s.cursor
	e.selActive = false
	e.showMessage("Redo")
}

// ── CURSOR MOVEMENT ──────────────────────────────────────────────────────────

func (e *Editor) moveCursor(line, col int, extending bool) {
	e.lastWasInsert = false
	if !extending {
		e.selActive = false
	} else if !e.selActive {
		e.selAnchor = e.cursor
		e.selActive = true
	}
	if line < 0 {
		line = 0
	}
	if line >= e.buf.LineCount() {
		line = e.buf.LineCount() - 1
	}
	lineLen := len(runesOf(e.buf.Line(line)))
	if col < 0 {
		col = 0
	}
	if col > lineLen {
		col = lineLen
	}
	e.cursor = Pos{line, col}
}

func (e *Editor) moveUp(ext bool) {
	if e.cursor.Line > 0 {
		nl := e.cursor.Line - 1
		nc := clampCol(e.cursor.Col, len(runesOf(e.buf.Line(nl))))
		e.moveCursor(nl, nc, ext)
	}
}

func (e *Editor) moveDown(ext bool) {
	if e.cursor.Line < e.buf.LineCount()-1 {
		nl := e.cursor.Line + 1
		nc := clampCol(e.cursor.Col, len(runesOf(e.buf.Line(nl))))
		e.moveCursor(nl, nc, ext)
	}
}

func (e *Editor) moveLeft(ext bool) {
	if e.cursor.Col > 0 {
		e.moveCursor(e.cursor.Line, e.cursor.Col-1, ext)
	} else if e.cursor.Line > 0 {
		pl := e.cursor.Line - 1
		e.moveCursor(pl, len(runesOf(e.buf.Line(pl))), ext)
	}
}

func (e *Editor) moveRight(ext bool) {
	lineLen := len(runesOf(e.buf.Line(e.cursor.Line)))
	if e.cursor.Col < lineLen {
		e.moveCursor(e.cursor.Line, e.cursor.Col+1, ext)
	} else if e.cursor.Line < e.buf.LineCount()-1 {
		e.moveCursor(e.cursor.Line+1, 0, ext)
	}
}

func (e *Editor) moveToLineStart(ext bool) {
	e.moveCursor(e.cursor.Line, 0, ext)
}

func (e *Editor) moveToLineEnd(ext bool) {
	e.moveCursor(e.cursor.Line, len(runesOf(e.buf.Line(e.cursor.Line))), ext)
}

func (e *Editor) wordLeft(ext bool) {
	line := e.cursor.Line
	col := e.cursor.Col
	r := runesOf(e.buf.Line(line))
	if col == 0 {
		if line > 0 {
			line--
			r = runesOf(e.buf.Line(line))
			col = len(r)
		}
	} else {
		col--
		for col > 0 && !isWordRune(r[col]) {
			col--
		}
		for col > 0 && isWordRune(r[col-1]) {
			col--
		}
	}
	e.moveCursor(line, col, ext)
}

func (e *Editor) wordRight(ext bool) {
	line := e.cursor.Line
	col := e.cursor.Col
	r := runesOf(e.buf.Line(line))
	if col >= len(r) {
		if line < e.buf.LineCount()-1 {
			line++
			col = 0
		}
	} else {
		for col < len(r) && isWordRune(r[col]) {
			col++
		}
		for col < len(r) && !isWordRune(r[col]) {
			col++
		}
	}
	e.moveCursor(line, col, ext)
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func leadingWhitespace(s string) string {
	for i, ch := range s {
		if ch != ' ' && ch != '\t' {
			return s[:i]
		}
	}
	return s
}

// ── SELECTION HELPERS ────────────────────────────────────────────────────────

func (e *Editor) normalizedSel() (from, to Pos) {
	if !e.selActive {
		return e.cursor, e.cursor
	}
	if e.selAnchor.Before(e.cursor) || e.selAnchor == e.cursor {
		return e.selAnchor, e.cursor
	}
	return e.cursor, e.selAnchor
}

func (e *Editor) selectAll() {
	e.selAnchor = Pos{0, 0}
	last := e.buf.LineCount() - 1
	e.cursor = Pos{last, len(runesOf(e.buf.Line(last)))}
	e.selActive = true
}

func (e *Editor) selectLine() {
	e.selAnchor = Pos{e.cursor.Line, 0}
	e.cursor = Pos{e.cursor.Line, len(runesOf(e.buf.Line(e.cursor.Line)))}
	e.selActive = true
}

func (e *Editor) deleteSelection() {
	from, to := e.normalizedSel()
	e.cursor = e.buf.DeleteRange(from, to)
	e.selActive = false
}

// ── EDIT OPERATIONS ──────────────────────────────────────────────────────────

func (e *Editor) copySelection() {
	if !e.selActive {
		return
	}
	from, to := e.normalizedSel()
	text := e.buf.GetRange(from, to)
	e.writeClipboard(text)
	e.showMessage("Copied")
}

func (e *Editor) copyLine() {
	text := e.buf.Line(e.cursor.Line) + "\n"
	e.writeClipboard(text)
	e.showMessage("Line copied")
}

func (e *Editor) cutSelection() {
	if !e.selActive {
		return
	}
	e.pushUndo()
	from, to := e.normalizedSel()
	text := e.buf.GetRange(from, to)
	e.writeClipboard(text)
	e.cursor = e.buf.DeleteRange(from, to)
	e.selActive = false
	e.showMessage("Cut")
}

func (e *Editor) cutLine() {
	e.pushUndo()
	line := e.cursor.Line
	text := e.buf.Line(line) + "\n"
	e.writeClipboard(text)
	if e.buf.LineCount() > 1 {
		e.buf.Lines = append(e.buf.Lines[:line], e.buf.Lines[line+1:]...)
	} else {
		e.buf.Lines[0] = ""
	}
	e.buf.Modified = true
	if e.cursor.Line >= e.buf.LineCount() {
		e.cursor.Line = e.buf.LineCount() - 1
	}
	e.cursor.Col = 0
	e.selActive = false
	e.showMessage("Line cut")
}

func (e *Editor) paste() {
	text := e.readClipboard()
	if text == "" {
		return
	}
	if e.selActive {
		e.deleteSelection()
	}
	nl, nc := e.buf.InsertText(e.cursor.Line, e.cursor.Col, text)
	e.cursor = Pos{nl, nc}
}

func (e *Editor) performReplace() {
	if e.replaceFind == "" {
		e.showError("Nothing to replace")
		return
	}
	switch e.replaceScope {
	case 0: // selection
		if !e.selActive {
			e.showError("No selection — select text first, then ^R")
			return
		}
		from, to := e.normalizedSel()
		text := e.buf.GetRange(from, to)
		newText := strings.ReplaceAll(text, e.replaceFind, e.replaceWith)
		if newText == text {
			e.showMessage("No matches in selection")
			return
		}
		e.pushUndo()
		e.cursor = e.buf.DeleteRange(from, to)
		nl, nc := e.buf.InsertText(e.cursor.Line, e.cursor.Col, newText)
		e.cursor = Pos{nl, nc}
		e.selActive = false
		e.showMessage("Replaced in selection")
	case 1: // current line
		line := e.cursor.Line
		text := e.buf.Lines[line]
		newText := strings.ReplaceAll(text, e.replaceFind, e.replaceWith)
		if newText == text {
			e.showMessage("No matches on this line")
			return
		}
		e.pushUndo()
		e.buf.Lines[line] = newText
		e.buf.Modified = true
		lineLen := len(runesOf(newText))
		if e.cursor.Col > lineLen {
			e.cursor.Col = lineLen
		}
		e.showMessage(fmt.Sprintf("Replaced on line %d", line+1))
	}
}

// ── FIND ─────────────────────────────────────────────────────────────────────

func (e *Editor) updateFind() {
	e.findPos = e.findPos[:0]
	if e.findStr == "" {
		return
	}
	needle := []rune(strings.ToLower(e.findStr))
	nlen := len(needle)
	for li, line := range e.buf.Lines {
		haystack := []rune(strings.ToLower(line))
		for ci := 0; ci <= len(haystack)-nlen; ci++ {
			match := true
			for i := 0; i < nlen; i++ {
				if haystack[ci+i] != needle[i] {
					match = false
					break
				}
			}
			if match {
				e.findPos = append(e.findPos, Pos{li, ci})
			}
		}
	}
	if len(e.findPos) == 0 {
		return
	}
	e.findIdx = 0
	for i, p := range e.findPos {
		if !p.Before(e.cursor) {
			e.findIdx = i
			break
		}
	}
	e.cursor = e.findPos[e.findIdx]
}

func (e *Editor) findNext() {
	if len(e.findPos) == 0 {
		return
	}
	e.findIdx = (e.findIdx + 1) % len(e.findPos)
	e.cursor = e.findPos[e.findIdx]
}

func (e *Editor) findPrev() {
	if len(e.findPos) == 0 {
		return
	}
	e.findIdx = (e.findIdx - 1 + len(e.findPos)) % len(e.findPos)
	e.cursor = e.findPos[e.findIdx]
}

// ── CLIPBOARD ────────────────────────────────────────────────────────────────

func (e *Editor) writeClipboard(text string) {
	e.internalClip = text
	_ = clipboard.WriteAll(text)
}

func (e *Editor) readClipboard() string {
	if text, err := clipboard.ReadAll(); err == nil && text != "" {
		return text
	}
	return e.internalClip
}

// ── CONFIG ───────────────────────────────────────────────────────────────────

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "hedit", "config")
}

func (e *Editor) loadConfig() {
	path := configPath()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var idx int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "theme=%d", &idx); err == nil {
		if idx >= 0 && idx < len(e.themes) {
			e.themeIdx = idx
		}
	}
}

func (e *Editor) saveConfig() {
	path := configPath()
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, []byte(fmt.Sprintf("theme=%d\n", e.themeIdx)), 0644)
}

// ── MESSAGES ─────────────────────────────────────────────────────────────────

func (e *Editor) showMessage(msg string) { e.message = msg; e.msgErr = false }
func (e *Editor) showError(msg string)   { e.message = msg; e.msgErr = true }
func (e *Editor) clearMessage()          { e.message = ""; e.msgErr = false }

// ── SCROLL ───────────────────────────────────────────────────────────────────

func (e *Editor) contentRows() int {
	n := e.height - 4 // header + msgbar + hints1 + hints2
	if n < 1 {
		return 1
	}
	return n
}

func (e *Editor) gutterWidth() int {
	w := len(fmt.Sprintf("%d", e.buf.LineCount())) + 2
	if w < 5 {
		w = 5
	}
	return w
}

func (e *Editor) ensureVisible() {
	cr := e.contentRows()
	if e.cursor.Line < e.scrollLine {
		e.scrollLine = e.cursor.Line
	}
	if e.cursor.Line >= e.scrollLine+cr {
		e.scrollLine = e.cursor.Line - cr + 1
	}
	textW := e.width - e.gutterWidth()
	if textW < 1 {
		textW = 1
	}
	if e.cursor.Col < e.scrollCol {
		e.scrollCol = e.cursor.Col
	}
	if e.cursor.Col >= e.scrollCol+textW {
		e.scrollCol = e.cursor.Col - textW + 1
	}
}

// ── RENDERING ────────────────────────────────────────────────────────────────

func (e *Editor) theme() *Theme { return e.themes[e.themeIdx] }

func (e *Editor) fillRow(row int, style tcell.Style) {
	for x := 0; x < e.width; x++ {
		e.screen.SetContent(x, row, ' ', nil, style)
	}
}

func (e *Editor) drawStr(row, col int, s string, style tcell.Style) {
	for _, ch := range s {
		if col >= e.width {
			break
		}
		e.screen.SetContent(col, row, ch, nil, style)
		col++
	}
}

func (e *Editor) render() {
	t := e.theme()
	e.screen.Clear()

	gutterW := e.gutterWidth()
	textW := e.width - gutterW
	if textW < 0 {
		textW = 0
	}
	cr := e.contentRows()

	selFrom, selTo := e.normalizedSel()
	searchLen := len(runesOf(e.findStr))

	// ── HEADER (row 0) ──
	e.fillRow(0, t.Header)
	appName := " HEdit "
	e.drawStr(0, 0, appName, t.HeaderAcct)

	filename := e.buf.Filename
	if filename == "" {
		filename = "[ New File ]"
	}
	if e.buf.Modified {
		filename += " ●"
	}

	posStr := fmt.Sprintf("  Ln %d, Col %d ", e.cursor.Line+1, e.cursor.Col+1)
	fnStart := (e.width - len([]rune(filename))) / 2
	if fnStart < len([]rune(appName))+1 {
		fnStart = len([]rune(appName)) + 1
	}
	e.drawStr(0, fnStart, filename, t.Header)
	if e.width-len([]rune(posStr)) > 0 {
		e.drawStr(0, e.width-len([]rune(posStr)), posStr, t.Header)
	}

	// ── CONTENT ──
	for row := 0; row < cr; row++ {
		lineIdx := e.scrollLine + row
		screenRow := row + 1

		if lineIdx < e.buf.LineCount() {
			numStr := fmt.Sprintf("%*d  ", gutterW-2, lineIdx+1)
			e.drawStr(screenRow, 0, numStr, t.LineNum)
		} else {
			e.fillRow(screenRow, t.Normal)
			e.screen.SetContent(0, screenRow, '~', nil, t.LineNum)
			continue
		}

		for x := gutterW; x < e.width; x++ {
			e.screen.SetContent(x, screenRow, ' ', nil, t.Normal)
		}

		lineRunes := runesOf(e.buf.Line(lineIdx))

		for col := 0; col < textW; col++ {
			runeIdx := col + e.scrollCol
			screenCol := col + gutterW
			if runeIdx >= len(lineRunes) {
				e.screen.SetContent(screenCol, screenRow, ' ', nil, t.Normal)
				continue
			}
			ch := lineRunes[runeIdx]
			style := t.Normal

			if e.selActive {
				p := Pos{lineIdx, runeIdx}
				if !p.Before(selFrom) && p.Before(selTo) {
					style = t.Selection
				}
			}

			if e.mode == ModeFind && searchLen > 0 {
				for _, mp := range e.findPos {
					if mp.Line == lineIdx && runeIdx >= mp.Col && runeIdx < mp.Col+searchLen {
						if e.findPos[e.findIdx] == mp {
							style = t.FindCur
						} else {
							style = t.FindHL
						}
						break
					}
				}
			}

			e.screen.SetContent(screenCol, screenRow, ch, nil, style)
		}
	}

	// ── MESSAGE BAR (row height-3) ──
	msgRow := e.height - 3
	e.fillRow(msgRow, t.MsgBar)
	switch e.mode {
	case ModeFind:
		indicator := ""
		if e.findStr != "" {
			if len(e.findPos) > 0 {
				indicator = fmt.Sprintf("  [%d/%d]", e.findIdx+1, len(e.findPos))
			} else {
				indicator = "  [ not found ]"
			}
		}
		e.drawStr(msgRow, 0, " Find: "+e.findStr+indicator, t.MsgBar)
	case ModeReplaceFind:
		scopeStr := "Selection"
		if e.replaceScope == 1 {
			scopeStr = "Line"
		}
		e.drawStr(msgRow, 0, fmt.Sprintf(" Replace (%s) — Find: %s", scopeStr, e.replaceFind), t.MsgBar)
	case ModeReplaceWith:
		e.drawStr(msgRow, 0, fmt.Sprintf(" Replace: %s  →  With: %s", e.replaceFind, e.replaceWith), t.MsgBar)
	case ModeConfirmQuit:
		e.drawStr(msgRow, 0, " Unsaved changes.  [S] Save & Quit  [Q] Quit  [N] Cancel", t.MsgBarErr)
	case ModeSaveAs:
		e.drawStr(msgRow, 0, " Save as: "+e.saveInput+"_", t.MsgBar)
	case ModeSaveConflict:
		e.drawStr(msgRow, 0, fmt.Sprintf(" '%s' exists.  [O] Overwrite  [C] Copy  [N] Cancel", e.saveInput), t.MsgBarErr)
	default:
		if e.message != "" {
			style := t.MsgBar
			if e.msgErr {
				style = t.MsgBarErr
			}
			e.drawStr(msgRow, 0, " "+e.message, style)
		}
	}

	// ── HINTS ──
	hints1Row := e.height - 2
	hints2Row := e.height - 1
	e.fillRow(hints1Row, t.Hints)
	e.fillRow(hints2Row, t.Hints)

	switch e.mode {
	case ModeFind:
		e.drawHints(hints1Row, []hintItem{
			{"↵", "Next"}, {"↑", "Prev"}, {"↓", "Next"}, {"Esc", "Close"},
		}, t)
	case ModeReplaceFind:
		e.drawHints(hints1Row, []hintItem{
			{"↵", "Next Step"}, {"Esc", "Cancel"},
		}, t)
	case ModeReplaceWith:
		e.drawHints(hints1Row, []hintItem{
			{"↵", "Replace"}, {"Esc", "Cancel"},
		}, t)
	case ModeSettings:
		e.drawHints(hints1Row, []hintItem{
			{"↑↓", "Select Theme"}, {"↵", "Apply"}, {"Esc", "Close"},
		}, t)
	case ModeConfirmQuit:
		e.drawHints(hints1Row, []hintItem{
			{"S", "Save & Quit"}, {"Q", "Quit without saving"}, {"N/Esc", "Cancel"},
		}, t)
	case ModeSaveAs:
		e.drawHints(hints1Row, []hintItem{
			{"↵", "Confirm"}, {"Esc", "Cancel"},
		}, t)
	case ModeSaveConflict:
		e.drawHints(hints1Row, []hintItem{
			{"O", "Overwrite"}, {"C", "Save copy"}, {"N/Esc", "Back"},
		}, t)
	default:
		e.drawHints(hints1Row, []hintItem{
			{"^S", "Save"}, {"^Z", "Undo"}, {"^Y", "Redo"},
			{"^F", "Find"}, {"^R", "Replace"}, {"^U", "Settings"},
		}, t)
		e.drawHints(hints2Row, []hintItem{
			{"^C", "Copy"}, {"^X", "Cut"}, {"^V", "Paste"},
			{"^←→", "Word"}, {"Alt+A", "Sel Line"}, {"^Q", "Quit"},
		}, t)
	}

	// ── SETTINGS OVERLAY ──
	if e.mode == ModeSettings {
		e.renderSettings(t)
	}

	// ── CURSOR ──
	cursorScreenRow := e.cursor.Line - e.scrollLine + 1
	cursorScreenCol := e.cursor.Col - e.scrollCol + gutterW
	if cursorScreenRow >= 1 && cursorScreenRow <= cr && cursorScreenCol >= gutterW {
		e.screen.ShowCursor(cursorScreenCol, cursorScreenRow)
	} else {
		e.screen.HideCursor()
	}

	e.screen.Show()
}

type hintItem struct{ key, label string }

func (e *Editor) drawHints(row int, items []hintItem, t *Theme) {
	col := 1
	for _, item := range items {
		if col >= e.width-2 {
			break
		}
		e.drawStr(row, col, item.key, t.HintsKey)
		col += len([]rune(item.key))
		e.drawStr(row, col, " "+item.label, t.Hints)
		col += 1 + len([]rune(item.label)) + 3
	}
}

func (e *Editor) renderSettings(t *Theme) {
	panelW := 44
	panelH := len(e.themes) + 7
	px := (e.width - panelW) / 2
	py := (e.height - panelH) / 2

	for row := 0; row < panelH; row++ {
		for col := 0; col < panelW; col++ {
			e.screen.SetContent(px+col, py+row, ' ', nil, t.Header)
		}
	}

	top := "┌" + strings.Repeat("─", panelW-2) + "┐"
	bot := "└" + strings.Repeat("─", panelW-2) + "┘"
	e.drawStr(py, px, top, t.HeaderAcct)
	e.drawStr(py+panelH-1, px, bot, t.HeaderAcct)
	for row := 1; row < panelH-1; row++ {
		e.screen.SetContent(px, py+row, '│', nil, t.HeaderAcct)
		e.screen.SetContent(px+panelW-1, py+row, '│', nil, t.HeaderAcct)
	}

	e.drawStr(py+1, px+2, "  Settings — Color Theme", t.HeaderAcct)
	e.drawStr(py+2, px+2, strings.Repeat("─", panelW-4), t.Hints)

	for i, th := range e.themes {
		row := py + 3 + i
		marker := "  ○  "
		style := t.Hints
		if i == e.settingsIdx {
			marker = "  ●  "
			style = t.HintsKey
		}
		e.drawStr(row, px+1, marker+th.Name, style)
	}

	e.drawStr(py+panelH-2, px+2, "↑↓ select   ↵ apply   Esc close", t.Hints)
}
