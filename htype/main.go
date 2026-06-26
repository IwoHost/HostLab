package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

// ── VERSION ───────────────────────────────────────────────────────────────────

const version = "0.1"

// ── TYPES ────────────────────────────────────────────────────────────────────

type runMode int

const (
	modeWords runMode = iota
	modeCode
	modeQuote
)

type gameState int

const (
	stateWaiting gameState = iota
	stateTyping
	stateResults
)

// ── STYLES ───────────────────────────────────────────────────────────────────

var (
	stBg      = tcell.StyleDefault.Background(tcell.NewHexColor(0x0c0c0c)).Foreground(tcell.NewHexColor(0xf0f0ee))
	stHeader  = tcell.StyleDefault.Background(tcell.NewHexColor(0xf0f0ee)).Foreground(tcell.NewHexColor(0x0c0c0c))
	stFooter  = tcell.StyleDefault.Background(tcell.NewHexColor(0x111110)).Foreground(tcell.NewHexColor(0x888884))
	stPast    = tcell.StyleDefault.Foreground(tcell.NewHexColor(0x2a2a28))
	stPending = tcell.StyleDefault.Foreground(tcell.NewHexColor(0x555550))
	stOK      = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xe8e8e4))
	stErr     = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xe05555)).Background(tcell.NewHexColor(0x1e0606))
	stCursor  = tcell.StyleDefault.Foreground(tcell.NewHexColor(0x0c0c0c)).Background(tcell.NewHexColor(0xf0f0ee)).Bold(true)
	stGreen   = tcell.StyleDefault.Foreground(tcell.NewHexColor(0x4ec94e))
	stAmber   = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xc49a14))
	stDim     = tcell.StyleDefault.Foreground(tcell.NewHexColor(0x3a3a38))
	stSub     = tcell.StyleDefault.Foreground(tcell.NewHexColor(0x888884))
	stBright  = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xf0f0ee)).Bold(true)
)

// ── PERSONAL BEST ─────────────────────────────────────────────────────────────

type bestScores struct {
	Words int `json:"words"`
	Code  int `json:"code"`
	Quote int `json:"quote"`
}

func (b *bestScores) get(md runMode) int {
	switch md {
	case modeCode:
		return b.Code
	case modeQuote:
		return b.Quote
	default:
		return b.Words
	}
}

func (b *bestScores) set(md runMode, wpm int) {
	switch md {
	case modeCode:
		b.Code = wpm
	case modeQuote:
		b.Quote = wpm
	default:
		b.Words = wpm
	}
}

func bestPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".htype")
}

func loadBest() bestScores {
	data, err := os.ReadFile(bestPath())
	if err != nil {
		return bestScores{}
	}
	var b bestScores
	json.Unmarshal(data, &b) //nolint
	return b
}

func saveBest(b bestScores) {
	p := bestPath()
	if p == "" {
		return
	}
	data, _ := json.Marshal(b)
	os.WriteFile(p, data, 0644) //nolint
}

// ── GAME ──────────────────────────────────────────────────────────────────────

type game struct {
	sc            tcell.Screen
	text          []rune
	typed         []rune
	correct       []bool
	pos           int
	state         gameState
	mode          runMode
	wordCount     int // effective display count (may reflect quote length)
	baseWordCount int // user's configured word count, preserved across mode switches
	startTime     time.Time
	endTime       time.Time
	errors        int
	newBest       bool
	best          bestScores
	// layout
	lineWidth  int
	lines      []string
	lineStarts []int
}

func newGame(sc tcell.Screen, md runMode, wc int) *game {
	g := &game{sc: sc, mode: md, wordCount: wc, baseWordCount: wc, best: loadBest()}
	g.reset()
	return g
}

func (g *game) reset() {
	g.text = generateText(g.mode, g.baseWordCount)
	if g.mode == modeQuote {
		g.wordCount = len(strings.Fields(string(g.text)))
	} else {
		g.wordCount = g.baseWordCount
	}
	n := len(g.text)
	g.typed = make([]rune, n)
	g.correct = make([]bool, n)
	g.pos = 0
	g.state = stateWaiting
	g.errors = 0
	g.newBest = false
	g.reflow()
}

func (g *game) reflow() {
	sw, _ := g.sc.Size()
	lw := sw - 10
	if lw < 38 {
		lw = 38
	}
	if lw > 68 {
		lw = 68
	}
	g.lineWidth = lw
	g.lines, g.lineStarts = wrapText(g.text, lw)
}

func generateText(md runMode, wc int) []rune {
	switch md {
	case modeCode:
		return []rune(strings.Join(pickN(codeWords, wc), " "))
	case modeQuote:
		return []rune(quotes[rand.Intn(len(quotes))])
	default:
		return []rune(strings.Join(pickN(commonWords, wc), " "))
	}
}

// ── WRAP ──────────────────────────────────────────────────────────────────────

// wrapText splits text into lines of at most `width` runes (at word boundaries).
// Each line may have a trailing space for the inter-word gap that follows it,
// so lineStarts[i] is the exact rune offset of lines[i][0] in text.
func wrapText(text []rune, width int) (lines []string, starts []int) {
	if len(text) == 0 {
		return
	}
	words := strings.Fields(string(text))

	var lineRunes []rune
	lineStart := 0
	pos := 0

	flush := func() {
		lines = append(lines, string(lineRunes))
		starts = append(starts, lineStart)
		lineRunes = nil
	}

	for i, w := range words {
		wr := []rune(w)
		if i > 0 {
			if len(lineRunes)+1+len(wr) > width {
				// include the space as last char of the current line so
				// pos→lineStart mapping is contiguous in g.text
				lineRunes = append(lineRunes, ' ')
				flush()
				pos++        // account for the space in g.text
				lineStart = pos
			} else {
				lineRunes = append(lineRunes, ' ')
				pos++
			}
		}
		lineRunes = append(lineRunes, wr...)
		pos += len(wr)
	}
	if len(lineRunes) > 0 {
		flush()
	}
	return
}

// ── GAME QUERIES ──────────────────────────────────────────────────────────────

func (g *game) currentLineIdx() int {
	for i := range g.lines {
		if i+1 < len(g.lines) && g.pos < g.lineStarts[i+1] {
			return i
		}
	}
	return len(g.lines) - 1
}

func (g *game) correctChars() int {
	n := 0
	for _, ok := range g.correct {
		if ok {
			n++
		}
	}
	return n
}

func (g *game) wpm() int {
	var elapsed float64
	switch g.state {
	case stateWaiting:
		return 0
	case stateResults:
		elapsed = g.endTime.Sub(g.startTime).Minutes()
	default:
		elapsed = time.Since(g.startTime).Minutes()
	}
	if elapsed <= 0 {
		return 0
	}
	limit := g.pos
	if g.state == stateResults {
		limit = len(g.correct)
	}
	correct := 0
	for i := 0; i < limit; i++ {
		if g.correct[i] {
			correct++
		}
	}
	return int(float64(correct) / 5.0 / elapsed)
}

func (g *game) rawWPM() int {
	if g.state != stateResults {
		return 0
	}
	elapsed := g.endTime.Sub(g.startTime).Minutes()
	if elapsed <= 0 {
		return 0
	}
	return int(float64(len(g.text)) / 5.0 / elapsed)
}

func (g *game) accuracy() float64 {
	total := g.pos
	if g.state == stateResults {
		total = len(g.correct)
	}
	if total == 0 {
		return 100.0
	}
	correct := 0
	for i := 0; i < total; i++ {
		if g.correct[i] {
			correct++
		}
	}
	return float64(correct) / float64(total) * 100.0
}

func (g *game) elapsedStr() string {
	var d time.Duration
	switch g.state {
	case stateTyping:
		d = time.Since(g.startTime)
	case stateResults:
		d = g.endTime.Sub(g.startTime)
	}
	s := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

// ── INPUT ─────────────────────────────────────────────────────────────────────

// handleKey returns true when the app should quit.
func (g *game) handleKey(ev *tcell.EventKey) bool {
	switch g.state {
	case stateWaiting:
		switch ev.Key() {
		case tcell.KeyEsc, tcell.KeyCtrlC:
			return true
		case tcell.KeyTab:
			g.mode = (g.mode + 1) % 3
			g.reset()
		case tcell.KeyRune:
			r := ev.Rune()
			if r == '+' || r == '=' {
				g.baseWordCount = clampWC(g.baseWordCount + 10)
				g.reset()
			} else if r == '-' {
				g.baseWordCount = clampWC(g.baseWordCount - 10)
				g.reset()
			} else {
				g.state = stateTyping
				g.startTime = time.Now()
				g.typeRune(r)
			}
		}
	case stateTyping:
		switch ev.Key() {
		case tcell.KeyEsc, tcell.KeyCtrlC:
			return true
		case tcell.KeyCtrlR:
			g.reset()
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			g.backspace()
		case tcell.KeyRune:
			g.typeRune(ev.Rune())
		}
	case stateResults:
		switch ev.Key() {
		case tcell.KeyEsc, tcell.KeyCtrlC:
			return true
		case tcell.KeyTab, tcell.KeyCtrlR, tcell.KeyEnter:
			g.reset()
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'r', 'R':
				g.reset()
			case 'q', 'Q':
				return true
			}
		}
	}
	return false
}

func clampWC(n int) int {
	if n < 10 {
		return 10
	}
	if n > 200 {
		return 200
	}
	return n
}

func (g *game) typeRune(r rune) {
	if g.pos >= len(g.text) {
		return
	}
	g.typed[g.pos] = r
	g.correct[g.pos] = r == g.text[g.pos]
	if !g.correct[g.pos] {
		g.errors++
	}
	g.pos++
	if g.pos >= len(g.text) {
		g.state = stateResults
		g.endTime = time.Now()
		wpm := g.wpm()
		prev := g.best.get(g.mode)
		if wpm > prev {
			g.newBest = true
			g.best.set(g.mode, wpm)
			saveBest(g.best)
		}
	}
}

func (g *game) backspace() {
	if g.pos == 0 {
		return
	}
	g.pos--
	if !g.correct[g.pos] && g.errors > 0 {
		g.errors--
	}
	g.typed[g.pos] = 0
	g.correct[g.pos] = false
}

// ── RENDER ────────────────────────────────────────────────────────────────────

func (g *game) render() {
	sc := g.sc
	sc.Clear()
	sw, sh := sc.Size()
	if g.state == stateResults {
		g.renderResults(sw, sh)
	} else {
		g.renderTyping(sw, sh)
	}
	sc.Show()
}

// ── TYPING SCREEN ─────────────────────────────────────────────────────────────

func (g *game) renderTyping(sw, sh int) {
	sc := g.sc

	// ── header ──
	fillRow(sc, 0, sw, stHeader)
	var headerLeft string
	if g.mode == modeQuote {
		headerLeft = fmt.Sprintf(" htype · QUOTE · %d chars", len(g.text))
	} else {
		headerLeft = fmt.Sprintf(" htype · %s · %d words", modeName(g.mode), g.wordCount)
	}
	puts(sc, 0, 0, headerLeft, stHeader)

	bestWPM := g.best.get(g.mode)
	var headerRight string
	if g.state == stateTyping {
		wpm := g.wpm()
		headerRight = fmt.Sprintf("WPM: %d ", wpm)
		// color the number
		prefix := "WPM: "
		numStr := fmt.Sprintf("%d ", wpm)
		rx := sw - len(headerRight)
		if rx > 0 {
			puts(sc, rx, 0, prefix, stHeader)
			numSt := stHeader
			if wpm >= 80 {
				numSt = stHeader.Foreground(tcell.NewHexColor(0x1a6a1a)).Bold(true)
			} else if wpm >= 50 {
				numSt = stHeader.Foreground(tcell.NewHexColor(0x7a5200))
			}
			puts(sc, rx+len(prefix), 0, numStr, numSt)
		}
	} else {
		if bestWPM > 0 {
			headerRight = fmt.Sprintf("best: %d wpm ", bestWPM)
		} else {
			headerRight = " "
		}
		rx := sw - len(headerRight)
		if rx > 0 {
			puts(sc, rx, 0, headerRight, stHeader.Foreground(tcell.NewHexColor(0x444440)))
		}
	}

	if len(g.lines) == 0 {
		return
	}

	// ── text area ──
	curLine := g.currentLineIdx()
	viewStart := curLine - 1
	if viewStart < 0 {
		viewStart = 0
	}

	// center text
	maxW := maxLineLen(g.lines)
	textLeft := (sw - maxW) / 2
	if textLeft < 4 {
		textLeft = 4
	}

	const lineStep = 2
	const contentH = 8 // 3 lines (at +0,+2,+4) + progress (at +7)
	available := sh - 2
	textTop := (available-contentH)/2 + 1
	if textTop < 2 {
		textTop = 2
	}

	for i := 0; i < 3; i++ {
		li := viewStart + i
		if li >= len(g.lines) {
			break
		}
		row := textTop + i*lineStep
		if row >= sh-2 {
			break
		}
		g.renderTextLine(li, textLeft, row, curLine)
	}

	// ── progress ──
	progRow := textTop + 3*lineStep + 1
	if progRow < sh-2 {
		g.renderProgress(textLeft, progRow, maxW)
	}

	// ── footer ──
	fillRow(sc, sh-1, sw, stFooter)
	if g.state == stateWaiting {
		puts(sc, 2, sh-1, "any key to start  ·  Tab cycle mode  ·  +/- words", stFooter)
		hint := "ESC quit "
		puts(sc, sw-len(hint), sh-1, hint, stFooter)
	} else {
		left := fmt.Sprintf(" ACC: %.1f%%  ·  %s", g.accuracy(), g.elapsedStr())
		puts(sc, 0, sh-1, left, stFooter)
		right := "Ctrl+R restart  ·  ESC quit "
		if sw-len(right) > 0 {
			puts(sc, sw-len(right), sh-1, right, stFooter)
		}
	}
}

func (g *game) renderTextLine(li, x, y, curLine int) {
	lineText := []rune(g.lines[li])
	lineStart := g.lineStarts[li]

	for i, ch := range lineText {
		pos := lineStart + i
		var st tcell.Style

		switch {
		case li < curLine:
			// completed line — very dim regardless of correctness
			st = stPast
		case pos < g.pos:
			if g.correct[pos] {
				st = stOK
			} else {
				st = stErr
			}
		case pos == g.pos:
			st = stCursor
		default:
			if li == curLine {
				st = stPending
			} else {
				st = stDim // future lines even dimmer
			}
		}

		g.sc.SetContent(x+i, y, ch, nil, st)
	}
}

func (g *game) renderProgress(x, y, barW int) {
	frac := 0.0
	if len(g.text) > 0 {
		frac = float64(g.pos) / float64(len(g.text))
	}
	filled := int(frac * float64(barW))
	for i := 0; i < barW; i++ {
		if i < filled {
			put(g.sc, x+i, y, '█', stGreen)
		} else {
			put(g.sc, x+i, y, '░', stDim)
		}
	}

	// word/char counter
	var label string
	if g.mode == modeQuote {
		label = fmt.Sprintf("  %d / %d chars", g.pos, len(g.text))
	} else {
		wordsTyped := 0
		for i := 0; i < g.pos; i++ {
			if g.text[i] == ' ' {
				wordsTyped++
			}
		}
		label = fmt.Sprintf("  %d / %d words", wordsTyped, g.wordCount)
	}
	puts(g.sc, x+barW, y, label, stDim)
}

// ── RESULTS SCREEN ────────────────────────────────────────────────────────────

func (g *game) renderResults(sw, sh int) {
	sc := g.sc
	wpm := g.wpm()
	raw := g.rawWPM()
	acc := g.accuracy()
	elapsed := g.endTime.Sub(g.startTime)
	secs := int(elapsed.Seconds())
	ms := int(elapsed.Milliseconds() % 1000 / 100)
	timeStr := fmt.Sprintf("%02d:%02d.%d", secs/60, secs%60, ms)
	prev := g.best.get(g.mode)

	const bw = 48
	const bh = 16
	bx := (sw - bw) / 2
	by := (sh - bh) / 2
	if bx < 2 {
		bx = 2
	}
	if by < 1 {
		by = 1
	}

	// box outline
	put(sc, bx, by, '┌', stDim)
	put(sc, bx+bw-1, by, '┐', stDim)
	put(sc, bx, by+bh-1, '└', stDim)
	put(sc, bx+bw-1, by+bh-1, '┘', stDim)
	for x := bx + 1; x < bx+bw-1; x++ {
		put(sc, x, by, '─', stDim)
		put(sc, x, by+bh-1, '─', stDim)
	}
	for y := by + 1; y < by+bh-1; y++ {
		put(sc, bx, y, '│', stDim)
		put(sc, bx+bw-1, y, '│', stDim)
	}

	// title
	title := "COMPLETE"
	titleSt := stSub
	if g.newBest {
		title = "★  NEW BEST  ★"
		titleSt = stGreen.Bold(true)
	}
	puts(sc, bx+(bw-len(title))/2, by+1, title, titleSt)

	// divider
	for x := bx + 1; x < bx+bw-1; x++ {
		put(sc, x, by+2, '─', stDim)
	}

	// WPM — the hero stat
	wpmStr := strconv.Itoa(wpm) + " wpm"
	wSt := wpmStyle(wpm)
	puts(sc, bx+(bw-len(wpmStr))/2, by+4, wpmStr, wSt)

	// prev best comparison
	if g.newBest {
		prevStr := fmt.Sprintf("was: %d  →  %d", prev, wpm)
		puts(sc, bx+(bw-len(prevStr))/2, by+5, prevStr, stSub)
	} else if prev > 0 {
		prevStr := fmt.Sprintf("best: %d wpm", prev)
		puts(sc, bx+(bw-len(prevStr))/2, by+5, prevStr, stDim)
	}

	// divider
	for x := bx + 1; x < bx+bw-1; x++ {
		put(sc, x, by+6, '─', stDim)
	}

	// stats grid
	const statX = 5
	const barX = 18
	const barW = 14

	type row struct {
		label string
		val   string
		frac  float64
	}
	rows := []row{
		{"accuracy", fmt.Sprintf("%.1f%%", acc), acc / 100.0},
		{"raw wpm", strconv.Itoa(raw), float64(raw) / 150.0},
		{"time", timeStr, 0},
		{"errors", strconv.Itoa(g.errors), 0},
		{"chars", fmt.Sprintf("%d / %d", g.correctChars(), len(g.text)), 0},
	}

	for i, r := range rows {
		ry := by + 7 + i
		puts(sc, bx+statX, ry, r.label, stDim)
		if r.frac > 0 {
			miniBar(sc, bx+barX, ry, barW, r.frac)
			puts(sc, bx+barX+barW+2, ry, r.val, stSub)
		} else {
			puts(sc, bx+barX+2, ry, r.val, stSub)
		}
	}

	// footer hint
	foot := "[R] again  ·  [Tab] next  ·  [Q] quit"
	puts(sc, bx+(bw-len(foot))/2, by+bh-2, foot, stDim)
}

// ── DRAW HELPERS ──────────────────────────────────────────────────────────────

func put(sc tcell.Screen, x, y int, ch rune, st tcell.Style) {
	sc.SetContent(x, y, ch, nil, st)
}

func puts(sc tcell.Screen, x, y int, s string, st tcell.Style) {
	for i, ch := range s {
		sc.SetContent(x+i, y, ch, nil, st)
	}
}

func fillRow(sc tcell.Screen, y, w int, st tcell.Style) {
	for x := 0; x < w; x++ {
		sc.SetContent(x, y, ' ', nil, st)
	}
}

func miniBar(sc tcell.Screen, x, y, width int, frac float64) {
	filled := int(frac * float64(width))
	if filled > width {
		filled = width
	}
	for i := 0; i < width; i++ {
		if i < filled {
			put(sc, x+i, y, '█', stGreen)
		} else {
			put(sc, x+i, y, '░', stDim)
		}
	}
}

func maxLineLen(lines []string) int {
	m := 0
	for _, l := range lines {
		if n := len([]rune(l)); n > m {
			m = n
		}
	}
	return m
}

func wpmStyle(wpm int) tcell.Style {
	switch {
	case wpm >= 80:
		return stGreen.Bold(true)
	case wpm >= 50:
		return stAmber
	case wpm > 0:
		return stSub
	default:
		return stDim
	}
}

func modeName(md runMode) string {
	return [...]string{"WORDS", "CODE", "QUOTE"}[md]
}

// ── MAIN ──────────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("htype v%s — iwohost.github.io/HostLab\n", version)
		return
	}

	var modeStr string
	var wordCount int
	flag.StringVar(&modeStr, "m", "words", "mode: words, code, quote")
	flag.IntVar(&wordCount, "n", 25, "word count (ignored in quote mode)")
	flag.Parse()

	// positional arg overrides -n
	if args := flag.Args(); len(args) > 0 {
		if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
			wordCount = n
		}
	}
	wordCount = clampWC(wordCount)

	md := modeWords
	switch strings.ToLower(modeStr) {
	case "code":
		md = modeCode
	case "quote":
		md = modeQuote
	}

	sc, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "htype:", err)
		os.Exit(1)
	}
	if err := sc.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "htype:", err)
		os.Exit(1)
	}
	defer sc.Fini()

	sc.SetStyle(stBg)
	sc.Clear()

	g := newGame(sc, md, wordCount)
	g.render()

	evCh := make(chan tcell.Event, 8)
	go func() {
		for {
			ev := sc.PollEvent()
			if ev == nil {
				return
			}
			evCh <- ev
		}
	}()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if g.state == stateTyping {
				g.render()
			}
		case ev := <-evCh:
			switch ev := ev.(type) {
			case *tcell.EventResize:
				sc.Sync()
				g.reflow()
				g.render()
			case *tcell.EventKey:
				if g.handleKey(ev) {
					return
				}
				g.render()
			}
		}
	}
}
