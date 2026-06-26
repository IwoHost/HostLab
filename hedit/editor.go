package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
)

type Mode int

const (
	ModeNormal Mode = iota
	ModeFind         // unified find / find+replace panel
	ModeSettings
	ModeConfirmQuit  // [S]Save  [Q]Quit  [N]Cancel
	ModeSaveAs       // typing filename
	ModeSaveConflict // file exists: [O]verwrite  [C]opy  [N]Cancel
	ModeGotoLine     // Ctrl+G: type a line number and jump
)

type snapshot struct {
	lines    []string
	cursor   Pos
	modified bool
}

// tabEntry holds the full per-file state for multi-file editing.
type tabEntry struct {
	buf           *Buffer
	cursor        Pos
	scrollLine    int
	scrollSubRow  int // first visible visual sub-row within scrollLine (soft-wrap)
	scrollCol     int
	selActive     bool
	selAnchor     Pos
	undoStack     []snapshot
	redoStack     []snapshot
	lastWasInsert bool
	lang          *langDef
}

// Editor is the full editor state.
type Editor struct {
	buf    *Buffer
	screen tcell.Screen

	cursor    Pos
	selActive bool
	selAnchor Pos

	scrollLine   int
	scrollSubRow int // first visible visual sub-row within scrollLine (soft-wrap only)
	scrollCol    int

	internalClip string

	mode Mode

	// Find / replace panel state
	findStr     string
	findPos     []Pos
	findIdx     int
	findCase    bool   // true = case-sensitive (default: false)
	findExpand  bool   // replace section visible
	findReplStr string // replacement text
	findFocus   int    // 0 = search field, 1 = replace field

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

	saveInput   string
	saveDir     string
	saveDirMode bool
	pendingQuit bool

	dragging bool
	lang     *langDef // syntax highlighting language (nil = plain text)
	softWrap bool     // visual soft-wrap (wraps display without changing buffer)

	tabs      []tabEntry
	tabIdx    int
	gotoInput string
}

func NewEditor(filenames []string) (*Editor, error) {
	if len(filenames) == 0 {
		filenames = []string{""}
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
	e := &Editor{screen: screen, themes: themes}
	e.loadConfig()
	e.softWrap = true

	for _, fn := range filenames {
		buf, err := NewBuffer(fn)
		if err != nil {
			screen.Fini()
			return nil, err
		}
		e.tabs = append(e.tabs, tabEntry{
			buf:  buf,
			lang: detectLang(buf.Filename),
		})
	}
	e.loadTab(0)
	return e, nil
}

func (e *Editor) saveCurrentTab() {
	e.tabs[e.tabIdx] = tabEntry{
		buf:           e.buf,
		cursor:        e.cursor,
		scrollLine:    e.scrollLine,
		scrollSubRow:  e.scrollSubRow,
		scrollCol:     e.scrollCol,
		selActive:     e.selActive,
		selAnchor:     e.selAnchor,
		undoStack:     e.undoStack,
		redoStack:     e.redoStack,
		lastWasInsert: e.lastWasInsert,
		lang:          e.lang,
	}
}

func (e *Editor) loadTab(idx int) {
	te := e.tabs[idx]
	e.tabIdx = idx
	e.buf = te.buf
	e.cursor = te.cursor
	e.scrollLine = te.scrollLine
	e.scrollSubRow = te.scrollSubRow
	e.scrollCol = te.scrollCol
	e.selActive = te.selActive
	e.selAnchor = te.selAnchor
	e.undoStack = te.undoStack
	e.redoStack = te.redoStack
	e.lastWasInsert = te.lastWasInsert
	e.lang = te.lang
	e.mode = ModeNormal
	e.clearMessage()
}

func (e *Editor) switchTab() {
	if len(e.tabs) <= 1 {
		e.showMessage("One file open — pass multiple files: hedit a.txt b.txt")
		return
	}
	e.saveCurrentTab()
	e.loadTab((e.tabIdx + 1) % len(e.tabs))
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
			e.ensureVisible()
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
	switch e.mode {
	case ModeFind:
		return e.handleFindKey(ev)
	case ModeSettings:
		return e.handleSettingsKey(ev)
	case ModeConfirmQuit:
		return e.handleConfirmKey(ev)
	case ModeSaveAs:
		return e.handleSaveAsKey(ev)
	case ModeSaveConflict:
		return e.handleSaveConflictKey(ev)
	case ModeGotoLine:
		return e.handleGotoKey(ev)
	}

	mod := ev.Modifiers()
	key := ev.Key()
	ch := ev.Rune()
	isAlt := mod&tcell.ModAlt != 0

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

	// Alt+D → duplicate current line below
	case isAlt && key == tcell.KeyRune && (ch == 'd' || ch == 'D'):
		e.pushUndo()
		e.duplicateLine()
		e.clearMessage()

	// Alt+G → go to end of logical line (works correctly even when soft-wrapped)
	case isAlt && key == tcell.KeyRune && (ch == 'g' || ch == 'G'):
		e.moveToLineEnd(false)

	// Ctrl+Z → undo
	case key == tcell.KeyCtrlZ:
		e.undo()

	// Ctrl+Y → redo
	case key == tcell.KeyCtrlY:
		e.redo()

	// Ctrl+S → save
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

	// Ctrl+F → find panel
	case key == tcell.KeyCtrlF:
		e.openFind(false)

	// Ctrl+R → find+replace panel
	case key == tcell.KeyCtrlR:
		e.openFind(true)

	// Ctrl+U → settings
	case key == tcell.KeyCtrlU:
		e.mode = ModeSettings
		e.settingsIdx = e.themeIdx
		e.clearMessage()

	// Ctrl+G → go-to-line prompt
	case key == tcell.KeyCtrlG:
		e.mode = ModeGotoLine
		e.gotoInput = ""
		e.clearMessage()

	// Ctrl+Left → word left
	case key == tcell.KeyLeft && mod&tcell.ModCtrl != 0:
		e.wordLeft(mod&tcell.ModShift != 0)

	// Ctrl+Right → word right
	case key == tcell.KeyRight && mod&tcell.ModCtrl != 0:
		e.wordRight(mod&tcell.ModShift != 0)

	// Alt+Up/Down → move entire logical line up or down (MUST come before plain Up/Down)
	case isAlt && key == tcell.KeyUp:
		e.pushUndo()
		e.moveLineUp()
		e.clearMessage()
	case isAlt && key == tcell.KeyDown:
		e.pushUndo()
		e.moveLineDown()
		e.clearMessage()

	// Arrow keys — Ctrl variants MUST come before plain variants
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
		newLine := e.cursor.Line - e.contentRows()
		if newLine < 0 {
			newLine = 0
		}
		e.moveCursor(newLine, e.cursor.Col, false)

	case key == tcell.KeyPgDn:
		newLine := e.cursor.Line + e.contentRows()
		if newLine >= e.buf.LineCount() {
			newLine = e.buf.LineCount() - 1
		}
		e.moveCursor(newLine, e.cursor.Col, false)

	// Alt+Backspace → delete entire current line
	case isAlt && (key == tcell.KeyBackspace || key == tcell.KeyBackspace2):
		e.pushUndo()
		e.deleteLine()
		e.clearMessage()

	// Alt+Delete → delete from cursor to end of line
	case isAlt && key == tcell.KeyDelete:
		e.pushUndo()
		e.deleteToEOL()
		e.clearMessage()

	case key == tcell.KeyBackspace || key == tcell.KeyBackspace2:
		e.pushUndo()
		if e.selActive {
			e.deleteSelection()
		} else {
			nl, nc := e.buf.Backspace(e.cursor.Line, e.cursor.Col)
			e.cursor = Pos{nl, nc}
			e.clearMessage()
		}

	case key == tcell.KeyDelete:
		e.pushUndo()
		if e.selActive {
			e.deleteSelection()
		} else {
			e.buf.Delete(e.cursor.Line, e.cursor.Col)
			e.clearMessage()
		}

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

	// Ctrl+Tab → cycle to next file tab
	case key == tcell.KeyTab && mod&tcell.ModCtrl != 0:
		e.switchTab()

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

	case key == tcell.KeyEscape:
		e.selActive = false
		e.clearMessage()

	case key == tcell.KeyRune && !isAlt && unicode.IsPrint(ch):
		if !e.lastWasInsert || e.selActive {
			e.pushUndo()
		} else {
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

// ── FIND / REPLACE PANEL ─────────────────────────────────────────────────────

func (e *Editor) openFind(withReplace bool) {
	e.findExpand = withReplace
	e.findFocus = 0
	if !withReplace {
		e.findReplStr = ""
	}
	e.mode = ModeFind
	e.clearMessage()
	e.updateFind()
}

func (e *Editor) handleFindKey(ev *tcell.EventKey) bool {
	mod := ev.Modifiers()
	key := ev.Key()
	ch := ev.Rune()
	isAlt := mod&tcell.ModAlt != 0

	switch {
	// Close panel
	case key == tcell.KeyEscape, key == tcell.KeyCtrlF:
		e.mode = ModeNormal
		e.findPos = nil

	// Ctrl+R → toggle replace section
	case key == tcell.KeyCtrlR:
		e.findExpand = !e.findExpand
		e.findFocus = 0

	// Alt+S → toggle case sensitivity
	case isAlt && key == tcell.KeyRune && (ch == 's' || ch == 'S'):
		e.findCase = !e.findCase
		e.updateFind()

	// Tab → swap focus between search / replace fields
	case key == tcell.KeyTab && e.findExpand:
		e.findFocus = 1 - e.findFocus

	// Enter → next match (search focus) or replace current (replace focus)
	case key == tcell.KeyEnter:
		if e.findExpand && e.findFocus == 1 {
			e.performFindReplace()
		} else {
			e.findNext()
		}

	// Down / Alt+Down → next match
	case key == tcell.KeyDown, isAlt && key == tcell.KeyDown:
		e.findNext()

	// Up / Alt+Up → prev match
	case key == tcell.KeyUp, isAlt && key == tcell.KeyUp:
		e.findPrev()

	// Alt+A → replace all (only when replace section is visible)
	case isAlt && key == tcell.KeyRune && (ch == 'a' || ch == 'A') && e.findExpand:
		e.performReplaceAll()

	// Backspace → delete from active field
	case key == tcell.KeyBackspace || key == tcell.KeyBackspace2:
		if e.findExpand && e.findFocus == 1 {
			if len(e.findReplStr) > 0 {
				r := runesOf(e.findReplStr)
				e.findReplStr = string(r[:len(r)-1])
			}
		} else {
			if len(e.findStr) > 0 {
				r := runesOf(e.findStr)
				e.findStr = string(r[:len(r)-1])
				e.updateFind()
			}
		}

	// Printable → append to active field
	case key == tcell.KeyRune && !isAlt && unicode.IsPrint(ch):
		if e.findExpand && e.findFocus == 1 {
			e.findReplStr += string(ch)
		} else {
			e.findStr += string(ch)
			e.updateFind()
		}
	}
	return false
}

// settingsItemCount returns total number of selectable items in the settings panel.
// Items 0..len(themes)-1 are themes; item len(themes) is the CRLF toggle.
func (e *Editor) settingsItemCount() int { return len(e.themes) + 1 }

func (e *Editor) handleSettingsKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		if e.settingsIdx > 0 {
			e.settingsIdx--
		}
	case tcell.KeyDown:
		if e.settingsIdx < e.settingsItemCount()-1 {
			e.settingsIdx++
		}
	case tcell.KeyEnter:
		if e.settingsIdx < len(e.themes) {
			e.themeIdx = e.settingsIdx
			e.saveConfig()
			e.showMessage("Theme applied: " + e.themes[e.themeIdx].Name)
		} else {
			e.buf.CRLF = !e.buf.CRLF
			lbl := "LF"
			if e.buf.CRLF {
				lbl = "CRLF"
			}
			e.showMessage("Line endings: " + lbl)
		}
		e.mode = ModeNormal
	case tcell.KeyEscape, tcell.KeyCtrlU:
		e.mode = ModeNormal
	}
	return false
}

func (e *Editor) handleConfirmKey(ev *tcell.EventKey) bool {
	key := ev.Key()
	ch := ev.Rune()
	switch {
	case key == tcell.KeyRune && (ch == 's' || ch == 'S'):
		e.startSave(true)
	case key == tcell.KeyRune && (ch == 'q' || ch == 'Q'):
		return true
	default:
		e.mode = ModeNormal
		e.clearMessage()
	}
	return false
}

func (e *Editor) handleSaveAsKey(ev *tcell.EventKey) bool {
	key := ev.Key()
	ch := ev.Rune()
	switch key {
	case tcell.KeyEscape:
		e.pendingQuit = false
		e.mode = ModeNormal
		e.clearMessage()
	case tcell.KeyEnter:
		if strings.TrimSpace(e.saveInput) == "" {
			e.showError("Filename required")
			return false
		}
		name := e.resolvedSavePath()
		if name == e.buf.Filename {
			return e.commitSave(name, true)
		}
		if _, err := os.Stat(name); err == nil {
			e.saveInput = name
			e.saveDirMode = true
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
		switch {
		case !e.saveDirMode && (ch == 'd' || ch == 'D'):
			e.saveInput = e.resolvedSavePath()
			e.saveDirMode = true
		case unicode.IsPrint(ch):
			e.saveInput += string(ch)
		}
	}
	return false
}

func (e *Editor) handleGotoKey(ev *tcell.EventKey) bool {
	key := ev.Key()
	ch := ev.Rune()
	switch {
	case key == tcell.KeyEscape:
		e.mode = ModeNormal
		e.clearMessage()
	case key == tcell.KeyBackspace || key == tcell.KeyBackspace2:
		if len(e.gotoInput) > 0 {
			e.gotoInput = e.gotoInput[:len(e.gotoInput)-1]
		}
	case key == tcell.KeyEnter:
		n, err := strconv.Atoi(strings.TrimSpace(e.gotoInput))
		if err == nil && n >= 1 {
			line := n - 1
			if line >= e.buf.LineCount() {
				line = e.buf.LineCount() - 1
			}
			e.cursor = Pos{line, 0}
			e.showMessage(fmt.Sprintf("Jumped to line %d", line+1))
		}
		e.mode = ModeNormal
		e.gotoInput = ""
	case key == tcell.KeyRune && ch >= '0' && ch <= '9':
		e.gotoInput += string(ch)
	}
	return false
}

func (e *Editor) handleSaveConflictKey(ev *tcell.EventKey) bool {
	key := ev.Key()
	ch := ev.Rune()
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
	e.saveDirMode = false
	if e.buf.Filename != "" {
		e.saveInput = filepath.Base(e.buf.Filename)
		e.saveDir = filepath.Dir(e.buf.Filename)
	} else {
		e.saveInput = ""
		if dir, err := os.Getwd(); err == nil {
			e.saveDir = dir
		} else {
			e.saveDir = "."
		}
	}
	e.mode = ModeSaveAs
	e.clearMessage()
}

func (e *Editor) resolvedSavePath() string {
	name := strings.TrimSpace(e.saveInput)
	if e.saveDirMode || filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(e.saveDir, name)
}

func (e *Editor) commitSave(filename string, updateFilename bool) bool {
	_ = os.MkdirAll(filepath.Dir(filename), 0755)
	le := e.buf.lineEnding()
	content := strings.Join(e.buf.Lines, le) + le
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		e.showError(fmt.Sprintf("Save failed: %v", err))
		e.mode = ModeNormal
		return false
	}
	if updateFilename {
		e.buf.Filename = filename
		e.buf.Modified = false
		e.lang = detectLang(filename)
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
	if e.softWrap {
		cw := e.width - e.gutterWidth()
		if cw < 1 {
			cw = 1
		}
		curSub := e.cursor.Col / cw
		visCol := e.cursor.Col % cw
		if curSub > 0 {
			// move to the visual row above within the same logical line
			newCol := (curSub-1)*cw + visCol
			lineLen := len(runesOf(e.buf.Line(e.cursor.Line)))
			if newCol > lineLen {
				newCol = lineLen
			}
			e.moveCursor(e.cursor.Line, newCol, ext)
		} else if e.cursor.Line > 0 {
			// jump to the last visual row of the previous logical line
			prevLine := e.cursor.Line - 1
			prevLen := len(runesOf(e.buf.Line(prevLine)))
			lastSub := prevLen/cw // index of last sub-row (prevLen/cw + 1 - 1)
			newCol := lastSub*cw + visCol
			if newCol > prevLen {
				newCol = prevLen
			}
			e.moveCursor(prevLine, newCol, ext)
		}
		return
	}
	if e.cursor.Line > 0 {
		nl := e.cursor.Line - 1
		nc := clampCol(e.cursor.Col, len(runesOf(e.buf.Line(nl))))
		e.moveCursor(nl, nc, ext)
	}
}

func (e *Editor) moveDown(ext bool) {
	if e.softWrap {
		cw := e.width - e.gutterWidth()
		if cw < 1 {
			cw = 1
		}
		lineLen := len(runesOf(e.buf.Line(e.cursor.Line)))
		totalSubs := lineLen/cw + 1
		curSub := e.cursor.Col / cw
		visCol := e.cursor.Col % cw
		if curSub+1 < totalSubs {
			// move to the visual row below within the same logical line
			newCol := (curSub+1)*cw + visCol
			if newCol > lineLen {
				newCol = lineLen
			}
			e.moveCursor(e.cursor.Line, newCol, ext)
		} else if e.cursor.Line < e.buf.LineCount()-1 {
			// jump to the first visual row of the next logical line
			nextLine := e.cursor.Line + 1
			nextLen := len(runesOf(e.buf.Line(nextLine)))
			newCol := visCol
			if newCol > nextLen {
				newCol = nextLen
			}
			e.moveCursor(nextLine, newCol, ext)
		}
		return
	}
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

func (e *Editor) moveToLineStart(ext bool) { e.moveCursor(e.cursor.Line, 0, ext) }

func (e *Editor) moveToLineEnd(ext bool) {
	e.moveCursor(e.cursor.Line, len(runesOf(e.buf.Line(e.cursor.Line))), ext)
}

func (e *Editor) wordLeft(ext bool) {
	line, col := e.cursor.Line, e.cursor.Col
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
	line, col := e.cursor.Line, e.cursor.Col
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

// ── SELECTION ────────────────────────────────────────────────────────────────

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

func (e *Editor) deleteLine() {
	e.selActive = false
	ln := e.cursor.Line
	if e.buf.LineCount() == 1 {
		e.buf.Lines[0] = ""
		e.cursor = Pos{0, 0}
		e.buf.Modified = true
		return
	}
	e.buf.Lines = append(e.buf.Lines[:ln], e.buf.Lines[ln+1:]...)
	e.buf.Modified = true
	if e.cursor.Line >= e.buf.LineCount() {
		e.cursor.Line = e.buf.LineCount() - 1
	}
	lineLen := len(runesOf(e.buf.Line(e.cursor.Line)))
	if e.cursor.Col > lineLen {
		e.cursor.Col = lineLen
	}
}

func (e *Editor) deleteToEOL() {
	e.selActive = false
	ln := e.cursor.Line
	r := runesOf(e.buf.Line(ln))
	if e.cursor.Col >= len(r) {
		return
	}
	e.buf.Lines[ln] = string(r[:e.cursor.Col])
	e.buf.Modified = true
}

func (e *Editor) moveLineUp() {
	ln := e.cursor.Line
	if ln == 0 {
		return
	}
	e.buf.Lines[ln-1], e.buf.Lines[ln] = e.buf.Lines[ln], e.buf.Lines[ln-1]
	e.buf.Modified = true
	e.cursor.Line = ln - 1
	e.selActive = false
}

func (e *Editor) moveLineDown() {
	ln := e.cursor.Line
	if ln >= e.buf.LineCount()-1 {
		return
	}
	e.buf.Lines[ln], e.buf.Lines[ln+1] = e.buf.Lines[ln+1], e.buf.Lines[ln]
	e.buf.Modified = true
	e.cursor.Line = ln + 1
	e.selActive = false
}

func (e *Editor) duplicateLine() {
	e.selActive = false
	ln := e.cursor.Line
	text := e.buf.Lines[ln]
	newLines := make([]string, len(e.buf.Lines)+1)
	copy(newLines[:ln+1], e.buf.Lines[:ln+1])
	newLines[ln+1] = text
	copy(newLines[ln+2:], e.buf.Lines[ln+1:])
	e.buf.Lines = newLines
	e.buf.Modified = true
	e.cursor.Line = ln + 1
	lineLen := len(runesOf(text))
	if e.cursor.Col > lineLen {
		e.cursor.Col = lineLen
	}
}

// ── EDIT OPERATIONS ──────────────────────────────────────────────────────────

func (e *Editor) copySelection() {
	if !e.selActive {
		return
	}
	from, to := e.normalizedSel()
	e.writeClipboard(e.buf.GetRange(from, to))
	e.showMessage("Copied")
}

func (e *Editor) copyLine() {
	e.writeClipboard(e.buf.Line(e.cursor.Line) + "\n")
	e.showMessage("Line copied")
}

func (e *Editor) cutSelection() {
	if !e.selActive {
		return
	}
	e.pushUndo()
	from, to := e.normalizedSel()
	e.writeClipboard(e.buf.GetRange(from, to))
	e.cursor = e.buf.DeleteRange(from, to)
	e.selActive = false
	e.showMessage("Cut")
}

func (e *Editor) cutLine() {
	e.pushUndo()
	line := e.cursor.Line
	e.writeClipboard(e.buf.Line(line) + "\n")
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

// ── FIND & REPLACE LOGIC ─────────────────────────────────────────────────────

// replaceAllFold replaces all occurrences of old with newStr in s.
// When caseSensitive is false the match is case-insensitive; original
// surrounding text casing is preserved.
func replaceAllFold(s, old, newStr string, caseSensitive bool) (string, int) {
	if old == "" {
		return s, 0
	}
	if caseSensitive {
		count := strings.Count(s, old)
		return strings.ReplaceAll(s, old, newStr), count
	}
	lowerS := strings.ToLower(s)
	lowerOld := strings.ToLower(old)
	var result strings.Builder
	count := 0
	start := 0
	for {
		idx := strings.Index(lowerS[start:], lowerOld)
		if idx < 0 {
			result.WriteString(s[start:])
			break
		}
		result.WriteString(s[start : start+idx])
		result.WriteString(newStr)
		start += idx + len(lowerOld)
		count++
	}
	return result.String(), count
}

func (e *Editor) updateFind() {
	e.findPos = e.findPos[:0]
	if e.findStr == "" {
		return
	}
	var needle []rune
	if e.findCase {
		needle = []rune(e.findStr)
	} else {
		needle = []rune(strings.ToLower(e.findStr))
	}
	nlen := len(needle)
	for li, line := range e.buf.Lines {
		var haystack []rune
		if e.findCase {
			haystack = []rune(line)
		} else {
			haystack = []rune(strings.ToLower(line))
		}
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

// performFindReplace replaces the current match and advances to the next.
func (e *Editor) performFindReplace() {
	if len(e.findPos) == 0 || e.findStr == "" {
		return
	}
	pos := e.findPos[e.findIdx]
	nlen := len(runesOf(e.findStr))
	lineRunes := runesOf(e.buf.Line(pos.Line))
	endCol := pos.Col + nlen
	if endCol > len(lineRunes) {
		endCol = len(lineRunes)
	}
	e.pushUndo()
	e.cursor = e.buf.DeleteRange(pos, Pos{pos.Line, endCol})
	nl, nc := e.buf.InsertText(e.cursor.Line, e.cursor.Col, e.findReplStr)
	e.cursor = Pos{nl, nc}
	e.updateFind()
	if len(e.findPos) > 0 {
		e.findNext()
	}
}

// performReplaceAll replaces every match in the document.
func (e *Editor) performReplaceAll() {
	if e.findStr == "" || len(e.findPos) == 0 {
		e.showMessage("No matches")
		return
	}
	e.pushUndo()
	total := 0
	for i, line := range e.buf.Lines {
		newLine, count := replaceAllFold(line, e.findStr, e.findReplStr, e.findCase)
		if count > 0 {
			e.buf.Lines[i] = newLine
			total += count
			e.buf.Modified = true
		}
	}
	e.updateFind()
	e.showMessage(fmt.Sprintf("Replaced %d occurrence(s)", total))
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
	n := e.height - 4
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

func (e *Editor) lineVisualRows(idx int) int {
	if !e.softWrap {
		return 1
	}
	cw := e.width - e.gutterWidth()
	if cw < 1 {
		return 1
	}
	// +1 because the cursor can sit one past the last character,
	// landing on an extra blank visual row (matches the render's nsub formula).
	return len(runesOf(e.buf.Line(idx)))/cw + 1
}

func (e *Editor) ensureVisible() {
	if e.softWrap {
		e.scrollCol = 0
		cr := e.contentRows()
		cw := e.width - e.gutterWidth()
		if cw < 1 {
			cw = 1
		}

		// Clamp scroll state to valid range after resizes or deletions.
		if e.scrollLine >= e.buf.LineCount() {
			e.scrollLine = e.buf.LineCount() - 1
			e.scrollSubRow = 0
		}
		scrollLineSubs := len(runesOf(e.buf.Line(e.scrollLine)))/cw + 1
		if e.scrollSubRow >= scrollLineSubs {
			e.scrollSubRow = scrollLineSubs - 1
		}
		if e.scrollSubRow < 0 {
			e.scrollSubRow = 0
		}

		cursorSub := e.cursor.Col / cw

		// Cursor above the current scroll position?
		if e.cursor.Line < e.scrollLine ||
			(e.cursor.Line == e.scrollLine && cursorSub < e.scrollSubRow) {
			e.scrollLine = e.cursor.Line
			e.scrollSubRow = cursorSub
			return
		}

		// Visual rows from (scrollLine, scrollSubRow) to cursor.
		visFromScroll := -e.scrollSubRow
		for li := e.scrollLine; li < e.cursor.Line; li++ {
			visFromScroll += e.lineVisualRows(li)
		}
		visFromScroll += cursorSub

		if visFromScroll < cr {
			return // cursor already in view
		}

		// Cursor below viewport: advance scroll so cursor lands on last visible row.
		advance := visFromScroll - (cr - 1)
		e.scrollSubRow += advance
		// Normalise: carry-over sub-rows into scrollLine increments.
		for e.scrollLine < e.buf.LineCount() {
			subs := len(runesOf(e.buf.Line(e.scrollLine)))/cw + 1
			if e.scrollSubRow < subs {
				break
			}
			e.scrollSubRow -= subs
			e.scrollLine++
		}
		if e.scrollLine >= e.buf.LineCount() {
			e.scrollLine = e.buf.LineCount() - 1
			e.scrollSubRow = 0
		}
		return
	}
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

func tokenStyle(k tokenKind, t *Theme) tcell.Style {
	switch k {
	case tkKeyword:
		return t.HLKeyword
	case tkType:
		return t.HLType
	case tkBuiltin:
		return t.HLBuiltin
	case tkString:
		return t.HLString
	case tkComment:
		return t.HLComment
	case tkNumber:
		return t.HLNumber
	default:
		return t.Normal
	}
}

func (e *Editor) theme() *Theme { return e.themes[e.themeIdx] }

func (e *Editor) fillRow(row int, style tcell.Style) {
	for x := 0; x < e.width; x++ {
		e.screen.SetContent(x, row, ' ', nil, style)
	}
}

func (e *Editor) drawStr(row, col int, s string, style tcell.Style) int {
	for _, ch := range s {
		if col >= e.width {
			break
		}
		e.screen.SetContent(col, row, ch, nil, style)
		col++
	}
	return col
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

	// ── HEADER ──
	e.fillRow(0, t.Header)
	e.drawStr(0, 0, " HEdit ", t.HeaderAcct)
	filename := e.buf.Filename
	if filename == "" {
		filename = "[ New File ]"
	}
	if len(e.tabs) > 1 {
		filename = fmt.Sprintf("[%d/%d] %s", e.tabIdx+1, len(e.tabs), filename)
	}
	if e.buf.Modified {
		filename += " ●"
	}
	eol := "LF"
	if e.buf.CRLF {
		eol = "CRLF"
	}
	posStr := fmt.Sprintf("  %s  Ln %d, Col %d ", eol, e.cursor.Line+1, e.cursor.Col+1)
	fnStart := (e.width - len([]rune(filename))) / 2
	if fnStart < 8 {
		fnStart = 8
	}
	e.drawStr(0, fnStart, filename, t.Header)
	if e.width-len([]rune(posStr)) > 0 {
		e.drawStr(0, e.width-len([]rune(posStr)), posStr, t.Header)
	}

	// ── CONTENT ──

	// Pre-scan: compute highlight carry-state at the top of the viewport.
	hlSt := hlNormal
	if e.lang != nil {
		for li := 0; li < e.scrollLine && li < e.buf.LineCount(); li++ {
			_, hlSt = highlight(e.buf.Line(li), e.lang, hlSt)
		}
	}

	if e.softWrap {
		cw := e.width - gutterW
		if cw < 1 {
			cw = 1
		}
		// Build list of (logicalLine, subRow) for visible visual rows.
		type visRow struct{ line, sub int }
		vis := make([]visRow, 0, cr+4)
		for li := e.scrollLine; li < e.buf.LineCount() && len(vis) < cr; li++ {
			r := runesOf(e.buf.Line(li))
			nsub := len(r)/cw + 1
			if len(r) == 0 {
				nsub = 1
			}
			startSr := 0
			if li == e.scrollLine {
				startSr = e.scrollSubRow
			}
			for sr := startSr; sr < nsub && len(vis) < cr; sr++ {
				vis = append(vis, visRow{li, sr})
			}
		}
		// Precompute syntax tokens for each visible logical line.
		lineTokens := make(map[int][]tokenKind, cr)
		if e.lang != nil {
			scanSt := hlSt
			for li := e.scrollLine; li < e.scrollLine+cr+1 && li < e.buf.LineCount(); li++ {
				var toks []tokenKind
				toks, scanSt = highlight(e.buf.Line(li), e.lang, scanSt)
				lineTokens[li] = toks
			}
		}
		for i, vr := range vis {
			screenRow := i + 1
			lineRunes := runesOf(e.buf.Line(vr.line))
			tokens := lineTokens[vr.line]
			// Clear row.
			for x := 0; x < e.width; x++ {
				e.screen.SetContent(x, screenRow, ' ', nil, t.Normal)
			}
			// Gutter: line number on first sub-row, blank continuation on rest.
			if vr.sub == 0 {
				e.drawStr(screenRow, 0, fmt.Sprintf("%*d  ", gutterW-2, vr.line+1), t.LineNum)
			} else {
				for x := 0; x < gutterW; x++ {
					e.screen.SetContent(x, screenRow, ' ', nil, t.LineNum)
				}
			}
			// Draw runes for this visual sub-row.
			startRune := vr.sub * cw
			for col := 0; col < cw; col++ {
				runeIdx := startRune + col
				if runeIdx >= len(lineRunes) {
					break
				}
				ch := lineRunes[runeIdx]
				style := t.Normal
				if tokens != nil && runeIdx < len(tokens) {
					style = tokenStyle(tokens[runeIdx], t)
				}
				if e.selActive {
					p := Pos{vr.line, runeIdx}
					if !p.Before(selFrom) && p.Before(selTo) {
						style = t.Selection
					}
				}
				if e.mode == ModeFind && searchLen > 0 {
					for _, mp := range e.findPos {
						if mp.Line == vr.line && runeIdx >= mp.Col && runeIdx < mp.Col+searchLen {
							if e.findPos[e.findIdx] == mp {
								style = t.FindCur
							} else {
								style = t.FindHL
							}
							break
						}
					}
				}
				e.screen.SetContent(col+gutterW, screenRow, ch, nil, style)
			}
		}
		// Fill rows past end of file with ~.
		for row := len(vis); row < cr; row++ {
			screenRow := row + 1
			e.fillRow(screenRow, t.Normal)
			e.screen.SetContent(0, screenRow, '~', nil, t.LineNum)
		}
	} else {
		for row := 0; row < cr; row++ {
			lineIdx := e.scrollLine + row
			screenRow := row + 1
			if lineIdx < e.buf.LineCount() {
				e.drawStr(screenRow, 0, fmt.Sprintf("%*d  ", gutterW-2, lineIdx+1), t.LineNum)
			} else {
				e.fillRow(screenRow, t.Normal)
				e.screen.SetContent(0, screenRow, '~', nil, t.LineNum)
				continue
			}
			for x := gutterW; x < e.width; x++ {
				e.screen.SetContent(x, screenRow, ' ', nil, t.Normal)
			}
			lineRunes := runesOf(e.buf.Line(lineIdx))
			var tokens []tokenKind
			if e.lang != nil {
				tokens, hlSt = highlight(e.buf.Line(lineIdx), e.lang, hlSt)
			}
			for col := 0; col < textW; col++ {
				runeIdx := col + e.scrollCol
				if runeIdx >= len(lineRunes) {
					break
				}
				ch := lineRunes[runeIdx]
				style := t.Normal
				if tokens != nil && runeIdx < len(tokens) {
					style = tokenStyle(tokens[runeIdx], t)
				}
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
				e.screen.SetContent(col+gutterW, screenRow, ch, nil, style)
			}
		}
	}

	// ── BOTTOM ROWS ──
	msgRow := e.height - 3
	h1Row := e.height - 2
	h2Row := e.height - 1

	switch e.mode {
	case ModeFind:
		e.renderFindPanel(msgRow, h1Row, h2Row, t)

	case ModeConfirmQuit:
		e.fillRow(msgRow, t.MsgBarErr)
		e.fillRow(h1Row, t.Hints)
		e.fillRow(h2Row, t.Hints)
		e.drawStr(msgRow, 0, " Unsaved changes.  [S] Save & Quit  [Q] Quit  [N] Cancel", t.MsgBarErr)
		e.drawHints(h1Row, []hintItem{
			{"S", "Save & Quit"}, {"Q", "Quit without saving"}, {"N/Esc", "Cancel"},
		}, t)

	case ModeSaveAs:
		e.fillRow(msgRow, t.MsgBar)
		e.fillRow(h1Row, t.Hints)
		e.fillRow(h2Row, t.Hints)
		if e.saveDirMode {
			e.drawStr(msgRow, 0, " Path: "+e.saveInput+"_", t.MsgBar)
		} else {
			e.drawStr(msgRow, 0, " Save as: "+e.saveInput+"_", t.MsgBar)
			e.drawStr(h2Row, 1, "in: "+e.saveDir, t.LineNum)
		}
		e.drawHints(h1Row, []hintItem{
			{"↵", "Confirm"}, {"Esc", "Cancel"}, {"d", "Edit path"},
		}, t)

	case ModeSaveConflict:
		e.fillRow(msgRow, t.MsgBarErr)
		e.fillRow(h1Row, t.Hints)
		e.fillRow(h2Row, t.Hints)
		e.drawStr(msgRow, 0, fmt.Sprintf(" '%s' exists.  [O] Overwrite  [C] Copy  [N] Cancel", e.saveInput), t.MsgBarErr)
		e.drawHints(h1Row, []hintItem{
			{"O", "Overwrite"}, {"C", "Save copy"}, {"N/Esc", "Back"},
		}, t)

	case ModeGotoLine:
		e.fillRow(msgRow, t.MsgBar)
		e.fillRow(h1Row, t.Hints)
		e.fillRow(h2Row, t.Hints)
		e.drawStr(msgRow, 0, " Go to line: "+e.gotoInput+"_", t.MsgBar)
		e.drawHints(h1Row, []hintItem{{"↵", "Jump"}, {"Esc", "Cancel"}}, t)

	default:
		e.fillRow(msgRow, t.MsgBar)
		e.fillRow(h1Row, t.Hints)
		e.fillRow(h2Row, t.Hints)
		if e.message != "" {
			style := t.MsgBar
			if e.msgErr {
				style = t.MsgBarErr
			}
			e.drawStr(msgRow, 0, " "+e.message, style)
		}
		e.drawHints(h1Row, []hintItem{
			{"^S", "Save"}, {"^Z", "Undo"}, {"^F", "Find"}, {"^R", "Replace"}, {"^G", "GoLine"}, {"^U", "Settings"},
		}, t)
		e.drawHints(h2Row, []hintItem{
			{"Alt+↑↓", "Move Line"}, {"Alt+D", "Dup"}, {"Alt+BS", "Del Line"}, {"^Tab", "Next Tab"}, {"^Q", "Quit"},
		}, t)
	}

	// ── SETTINGS OVERLAY ──
	if e.mode == ModeSettings {
		e.fillRow(msgRow, t.MsgBar)
		e.fillRow(h1Row, t.Hints)
		e.fillRow(h2Row, t.Hints)
		e.renderSettings(t)
	}

	// ── CURSOR ──
	var cursorScreenRow, cursorScreenCol int
	if e.softWrap {
		cw := e.width - gutterW
		if cw < 1 {
			cw = 1
		}
		// Offset from the scroll position (scrollSubRow rows of scrollLine are hidden).
		visAbove := -e.scrollSubRow
		for li := e.scrollLine; li < e.cursor.Line; li++ {
			visAbove += e.lineVisualRows(li)
		}
		cursorScreenRow = visAbove + e.cursor.Col/cw + 1
		cursorScreenCol = e.cursor.Col%cw + gutterW
	} else {
		cursorScreenRow = e.cursor.Line - e.scrollLine + 1
		cursorScreenCol = e.cursor.Col - e.scrollCol + gutterW
	}
	if cursorScreenRow >= 1 && cursorScreenRow <= cr && cursorScreenCol >= gutterW {
		e.screen.ShowCursor(cursorScreenCol, cursorScreenRow)
	} else {
		e.screen.HideCursor()
	}
	e.screen.Show()
}

// renderFindPanel draws the find (and optionally replace) panel across the
// three bottom rows.
func (e *Editor) renderFindPanel(msgRow, h1Row, h2Row int, t *Theme) {
	e.fillRow(msgRow, t.MsgBar)
	e.fillRow(h1Row, t.MsgBar)
	e.fillRow(h2Row, t.Hints)

	// Case indicator: bright when case-sensitive is ON
	caseStyle := t.Hints
	caseLabel := " [Aa] "
	if e.findCase {
		caseStyle = t.HintsKey
	}

	// Match counter
	counter := ""
	if e.findStr != "" {
		if len(e.findPos) > 0 {
			counter = fmt.Sprintf("  %d/%d", e.findIdx+1, len(e.findPos))
		} else {
			counter = "  no match"
		}
	}

	if !e.findExpand {
		// ── Find only ──
		// Row: [ Find: text_   [Aa]   3/7 ]
		label := " Find: "
		col := e.drawStr(msgRow, 0, label, t.MsgBar)
		input := e.findStr
		if e.findFocus == 0 {
			input += "_"
		}
		col = e.drawStr(msgRow, col, input, t.FindCur)
		// right-align case + counter
		suffix := caseLabel + counter + " "
		if e.width-len([]rune(suffix)) > col {
			e.drawStr(msgRow, e.width-len([]rune(suffix)), caseLabel, caseStyle)
			e.drawStr(msgRow, e.width-len([]rune(counter+" ")), counter+" ", t.MsgBar)
		}
		// Hints row
		e.drawHints(h1Row, []hintItem{
			{"↵/↓", "Next"}, {"↑", "Prev"}, {"^R", "Replace"}, {"Alt+S", "Case"}, {"Esc", "Close"},
		}, t)
	} else {
		// ── Find + Replace ──
		// Find row
		findLabel := " Find:    "
		col := e.drawStr(msgRow, 0, findLabel, t.MsgBar)
		findInput := e.findStr
		if e.findFocus == 0 {
			findInput += "_"
		}
		col = e.drawStr(msgRow, col, findInput, t.FindCur)
		suffix := caseLabel + counter + " "
		if e.width-len([]rune(suffix)) > col {
			e.drawStr(msgRow, e.width-len([]rune(suffix)), caseLabel, caseStyle)
			e.drawStr(msgRow, e.width-len([]rune(counter+" ")), counter+" ", t.MsgBar)
		}

		// Replace row (using h1Row with MsgBar style)
		replLabel := " Replace: "
		col = e.drawStr(h1Row, 0, replLabel, t.MsgBar)
		replInput := e.findReplStr
		if e.findFocus == 1 {
			replInput += "_"
		}
		e.drawStr(h1Row, col, replInput, t.FindHL)

		// Hints
		e.drawHints(h2Row, []hintItem{
			{"Tab", "Swap"}, {"↵", "Replace"}, {"Alt+A", "All"}, {"Alt+S", "Case"}, {"Esc", "Close"},
		}, t)
	}
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
	panelH := len(e.themes) + 10 // themes + header + separator + crlf row + separator + footer
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
	// Theme section
	e.drawStr(py+1, px+2, "  Settings — Color Theme", t.HeaderAcct)
	e.drawStr(py+2, px+2, strings.Repeat("─", panelW-4), t.Hints)
	for i, th := range e.themes {
		row := py + 3 + i
		marker, style := "  ○  ", t.Hints
		if i == e.settingsIdx {
			marker, style = "  ●  ", t.HintsKey
		}
		e.drawStr(row, px+1, marker+th.Name, style)
	}
	// Line endings section
	sepRow := py + 3 + len(e.themes)
	e.drawStr(sepRow, px+2, strings.Repeat("─", panelW-4), t.Hints)
	e.drawStr(sepRow+1, px+2, "  Line Endings", t.HeaderAcct)
	crlfIdx := len(e.themes)
	eolMarker, eolStyle := "  ○  ", t.Hints
	if e.settingsIdx == crlfIdx {
		eolMarker, eolStyle = "  ●  ", t.HintsKey
	}
	eolLabel := "LF  (Unix)"
	if e.buf.CRLF {
		eolLabel = "CRLF  (Windows)"
	}
	e.drawStr(sepRow+2, px+1, eolMarker+"Toggle: "+eolLabel, eolStyle)
	e.drawStr(py+panelH-2, px+2, "↑↓ select   ↵ apply   Esc close", t.Hints)
}
