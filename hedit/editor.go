package main

import (
	"fmt"
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
	ModeConfirmQuit
)

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
			e.handleMouse(ev)
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
	}

	// ── Normal mode ──
	switch {
	// Alt+C → copy entire line
	case isAlt && key == tcell.KeyRune && (ch == 'c' || ch == 'C'):
		e.copyLine()

	// Alt+X → cut entire line
	case isAlt && key == tcell.KeyRune && (ch == 'x' || ch == 'X'):
		e.cutLine()

	// Ctrl+S → save
	case key == tcell.KeyCtrlS:
		e.save()

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
		e.paste()
		e.clearMessage()

	// Ctrl+F → find
	case key == tcell.KeyCtrlF:
		e.mode = ModeFind
		e.findStr = ""
		e.findPos = nil
		e.clearMessage()

	// Ctrl+U → settings
	case key == tcell.KeyCtrlU:
		e.mode = ModeSettings
		e.settingsIdx = e.themeIdx
		e.clearMessage()

	// Arrow keys (with optional Shift for selection)
	case key == tcell.KeyUp:
		e.moveUp(mod&tcell.ModShift != 0)
	case key == tcell.KeyDown:
		e.moveDown(mod&tcell.ModShift != 0)
	case key == tcell.KeyLeft:
		e.moveLeft(mod&tcell.ModShift != 0)
	case key == tcell.KeyRight:
		e.moveRight(mod&tcell.ModShift != 0)

	case key == tcell.KeyHome:
		e.moveToLineStart(mod&tcell.ModShift != 0)
	case key == tcell.KeyEnd:
		e.moveToLineEnd(mod&tcell.ModShift != 0)

	case key == tcell.KeyHome && mod&tcell.ModCtrl != 0:
		e.moveCursor(0, 0, mod&tcell.ModShift != 0)
	case key == tcell.KeyEnd && mod&tcell.ModCtrl != 0:
		last := e.buf.LineCount() - 1
		e.moveCursor(last, len(runesOf(e.buf.Line(last))), mod&tcell.ModShift != 0)

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
		if e.selActive {
			e.deleteSelection()
		} else {
			nl, nc := e.buf.Backspace(e.cursor.Line, e.cursor.Col)
			e.cursor = Pos{nl, nc}
			e.clearMessage()
		}

	// Delete
	case key == tcell.KeyDelete:
		if e.selActive {
			e.deleteSelection()
		} else {
			e.buf.Delete(e.cursor.Line, e.cursor.Col)
			e.clearMessage()
		}

	// Enter
	case key == tcell.KeyEnter || key == tcell.KeyCR:
		if e.selActive {
			e.deleteSelection()
		}
		nl, nc := e.buf.NewLine(e.cursor.Line, e.cursor.Col)
		e.cursor = Pos{nl, nc}
		e.clearMessage()

	// Tab → 4 spaces
	case key == tcell.KeyTab:
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

	// Printable character
	case key == tcell.KeyRune && !isAlt && unicode.IsPrint(ch):
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
		e.showMessage("Theme applied: " + e.themes[e.themeIdx].Name)
	case tcell.KeyEscape, tcell.KeyCtrlU:
		e.mode = ModeNormal
	}
	return false
}

func (e *Editor) handleConfirmKey(key tcell.Key, ch rune) bool {
	switch {
	case key == tcell.KeyRune && (ch == 'y' || ch == 'Y'), key == tcell.KeyEnter:
		return true
	default:
		e.mode = ModeNormal
		e.clearMessage()
	}
	return false
}

// ── MOUSE HANDLING ───────────────────────────────────────────────────────────

func (e *Editor) handleMouse(ev *tcell.EventMouse) {
	x, y := ev.Position()
	btn := ev.Buttons()
	gutterW := e.gutterWidth()

	// Only care about clicks in the content area
	if y < 1 || y > e.height-3 || x < gutterW {
		return
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
		if ev.Modifiers()&tcell.ModShift != 0 {
			e.moveCursor(line, col, true)
		} else {
			e.selActive = false
			e.cursor = Pos{line, col}
		}
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
}

// ── CURSOR MOVEMENT ──────────────────────────────────────────────────────────

func (e *Editor) moveCursor(line, col int, extending bool) {
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
	from, to := e.normalizedSel()
	text := e.buf.GetRange(from, to)
	e.writeClipboard(text)
	e.cursor = e.buf.DeleteRange(from, to)
	e.selActive = false
	e.showMessage("Cut")
}

func (e *Editor) cutLine() {
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

func (e *Editor) save() {
	if e.buf.Filename == "" {
		e.showError("No filename — run:  hedit <filename>")
		return
	}
	if err := e.buf.Save(); err != nil {
		e.showError(fmt.Sprintf("Save failed: %v", err))
		return
	}
	e.showMessage(fmt.Sprintf("Saved  %s", e.buf.Filename))
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
	// Jump to closest match at or after cursor
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

// ── MESSAGES ─────────────────────────────────────────────────────────────────

func (e *Editor) showMessage(msg string) {
	e.message = msg
	e.msgErr = false
}

func (e *Editor) showError(msg string) {
	e.message = msg
	e.msgErr = true
}

func (e *Editor) clearMessage() {
	e.message = ""
	e.msgErr = false
}

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
	// Vertical
	if e.cursor.Line < e.scrollLine {
		e.scrollLine = e.cursor.Line
	}
	if e.cursor.Line >= e.scrollLine+cr {
		e.scrollLine = e.cursor.Line - cr + 1
	}
	// Horizontal
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

func (e *Editor) theme() *Theme {
	return e.themes[e.themeIdx]
}

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
	// Center filename
	fnStart := (e.width - len([]rune(filename))) / 2
	if fnStart < len([]rune(appName))+1 {
		fnStart = len([]rune(appName)) + 1
	}
	e.drawStr(0, fnStart, filename, t.Header)
	if e.width-len([]rune(posStr)) > 0 {
		e.drawStr(0, e.width-len([]rune(posStr)), posStr, t.Header)
	}

	// ── CONTENT (rows 1..cr) ──
	for row := 0; row < cr; row++ {
		lineIdx := e.scrollLine + row
		screenRow := row + 1

		// Gutter
		if lineIdx < e.buf.LineCount() {
			numStr := fmt.Sprintf("%*d  ", gutterW-2, lineIdx+1)
			e.drawStr(screenRow, 0, numStr, t.LineNum)
		} else {
			e.fillRow(screenRow, t.Normal)
			e.screen.SetContent(0, screenRow, '~', nil, t.LineNum)
			continue
		}

		// Fill text area background
		for x := gutterW; x < e.width; x++ {
			e.screen.SetContent(x, screenRow, ' ', nil, t.Normal)
		}

		if lineIdx >= e.buf.LineCount() {
			continue
		}

		lineRunes := runesOf(e.buf.Line(lineIdx))

		for col := 0; col < textW; col++ {
			runeIdx := col + e.scrollCol
			screenCol := col + gutterW
			var ch rune = ' '
			if runeIdx < len(lineRunes) {
				ch = lineRunes[runeIdx]
			} else if runeIdx >= len(lineRunes) {
				// Past end of line — only draw background
				e.screen.SetContent(screenCol, screenRow, ' ', nil, t.Normal)
				continue
			}

			style := t.Normal

			// Selection
			if e.selActive {
				p := Pos{lineIdx, runeIdx}
				if !p.Before(selFrom) && p.Before(selTo) {
					style = t.Selection
				}
			}

			// Search highlight
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
	case ModeConfirmQuit:
		e.drawStr(msgRow, 0, " Unsaved changes. Quit? (y/n) ", t.MsgBarErr)
	default:
		if e.message != "" {
			style := t.MsgBar
			if e.msgErr {
				style = t.MsgBarErr
			}
			e.drawStr(msgRow, 0, " "+e.message, style)
		}
	}

	// ── HINTS (rows height-2 and height-1) ──
	hints1Row := e.height - 2
	hints2Row := e.height - 1
	e.fillRow(hints1Row, t.Hints)
	e.fillRow(hints2Row, t.Hints)

	switch e.mode {
	case ModeFind:
		e.drawHints(hints1Row, []hintItem{
			{"↵", "Next"}, {"↑", "Prev"}, {"↓", "Next"}, {"Esc", "Close Find"},
		}, t)
	case ModeSettings:
		e.drawHints(hints1Row, []hintItem{
			{"↑↓", "Select Theme"}, {"↵", "Apply"}, {"Esc", "Close"},
		}, t)
	case ModeConfirmQuit:
		e.drawHints(hints1Row, []hintItem{
			{"Y", "Confirm Quit"}, {"N/Esc", "Cancel"},
		}, t)
	default:
		e.drawHints(hints1Row, []hintItem{
			{"^S", "Save"}, {"^Q", "Quit"}, {"^F", "Find"},
			{"^A", "Sel All"}, {"^U", "Settings"},
		}, t)
		e.drawHints(hints2Row, []hintItem{
			{"^C", "Copy"}, {"^X", "Cut"}, {"^V", "Paste"},
			{"Alt+C", "Copy Line"}, {"Alt+X", "Cut Line"},
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

	// Background fill
	for row := 0; row < panelH; row++ {
		for col := 0; col < panelW; col++ {
			e.screen.SetContent(px+col, py+row, ' ', nil, t.Header)
		}
	}

	// Border
	top := "┌" + strings.Repeat("─", panelW-2) + "┐"
	bot := "└" + strings.Repeat("─", panelW-2) + "┘"
	e.drawStr(py, px, top, t.HeaderAcct)
	e.drawStr(py+panelH-1, px, bot, t.HeaderAcct)
	for row := 1; row < panelH-1; row++ {
		e.screen.SetContent(px, py+row, '│', nil, t.HeaderAcct)
		e.screen.SetContent(px+panelW-1, py+row, '│', nil, t.HeaderAcct)
	}

	// Title
	e.drawStr(py+1, px+2, "  Settings — Color Theme", t.HeaderAcct)
	e.drawStr(py+2, px+2, strings.Repeat("─", panelW-4), t.Hints)

	// Theme list
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
