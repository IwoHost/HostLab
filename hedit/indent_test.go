package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// ── reading the indent out of a file ─────────────────────────────────────────

func TestDetectIndentFindsTwoSpaces(t *testing.T) {
	lines := []string{
		"root:",
		"  level one",
		"    level two",
		"  back again",
	}
	w, tabs, ok := detectIndent(lines)
	if !ok || tabs || w != 2 {
		t.Errorf("got width %d tabs %v ok %v, want 2 spaces", w, tabs, ok)
	}
}

func TestDetectIndentFindsFourSpaces(t *testing.T) {
	lines := []string{"func f() {", "    a()", "    if x {", "        b()", "    }", "}"}
	w, tabs, ok := detectIndent(lines)
	if !ok || tabs || w != 4 {
		t.Errorf("got width %d tabs %v ok %v, want 4 spaces", w, tabs, ok)
	}
}

func TestDetectIndentFindsTabs(t *testing.T) {
	lines := []string{"func f() {", "\ta()", "\tif x {", "\t\tb()", "\t}", "}"}
	_, tabs, ok := detectIndent(lines)
	if !ok || !tabs {
		t.Errorf("got tabs %v ok %v, want tabs", tabs, ok)
	}
}

func TestDetectIndentGivesUpOnFlatFiles(t *testing.T) {
	lines := []string{"one", "two", "three"}
	if _, _, ok := detectIndent(lines); ok {
		t.Error("a file with no indentation should not claim a style")
	}
}

func TestDetectIndentIgnoresBlankLines(t *testing.T) {
	lines := []string{"a:", "", "  b:", "", "    c"}
	w, _, ok := detectIndent(lines)
	if !ok || w != 2 {
		t.Errorf("got width %d ok %v, want 2", w, ok)
	}
}

// ── .editorconfig ────────────────────────────────────────────────────────────

func TestParseEditorConfigReadsMatchingSection(t *testing.T) {
	const text = `
root = true

[*]
indent_style = space
indent_size = 2

[*.go]
indent_style = tab
`
	css, _ := parseEditorConfig(text, "style.css")
	if css.style != "space" || css.size != 2 {
		t.Errorf("css got %+v, want space/2", css)
	}
	goCfg, root := parseEditorConfig(text, "main.go")
	if goCfg.style != "tab" {
		t.Errorf("go got %+v, want tab", goCfg)
	}
	if !root {
		t.Error("root = true was not picked up")
	}
}

func TestParseEditorConfigBraceAlternatives(t *testing.T) {
	const text = "[*.{js,ts,tsx}]\nindent_size = 2\n"
	for _, name := range []string{"a.js", "a.ts", "a.tsx"} {
		if cfg, _ := parseEditorConfig(text, name); cfg.size != 2 {
			t.Errorf("%s: got %+v, want size 2", name, cfg)
		}
	}
	if cfg, _ := parseEditorConfig(text, "a.go"); cfg.size != 0 {
		t.Errorf("a.go should not match, got %+v", cfg)
	}
}

func TestParseEditorConfigTabWidthStandsInForSize(t *testing.T) {
	const text = "[*]\nindent_style = tab\nindent_size = tab\ntab_width = 8\n"
	cfg, _ := parseEditorConfig(text, "x.go")
	if cfg.style != "tab" || cfg.size != 8 {
		t.Errorf("got %+v, want tab/8", cfg)
	}
}

func TestParseEditorConfigIgnoresCommentsAndPreamble(t *testing.T) {
	const text = "# a comment\nindent_size = 99\n\n[*]\nindent_size = 3\n"
	cfg, _ := parseEditorConfig(text, "x.txt")
	if cfg.size != 3 {
		t.Errorf("got size %d, want 3 — settings before a section header are not rules", cfg.size)
	}
}

func TestEditorConfigMatchesPatterns(t *testing.T) {
	cases := []struct {
		pattern, name string
		want          bool
	}{
		{"*", "a.go", true},
		{"**", "a.go", true},
		{"*.go", "a.go", true},
		{"*.go", "a.js", false},
		{"lib/**.js", "a.js", true},
		{"Makefile", "Makefile", true},
		{"Makefile", "a.go", false},
	}
	for _, c := range cases {
		if got := editorConfigMatches(c.pattern, c.name); got != c.want {
			t.Errorf("%q vs %q: got %v, want %v", c.pattern, c.name, got, c.want)
		}
	}
}

// ── the whole chain, on disk ─────────────────────────────────────────────────

func TestEditorConfigOverridesWhatTheFileLooksLike(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, ".editorconfig"), "root = true\n[*.go]\nindent_style = tab\n")

	// a file that looks like four spaces, in a project that says tabs
	e := newTestEditor("func f() {", "    a()", "}")
	e.cfgIndentWidth, e.cfgIndentTabs = 4, false
	e.buf.Filename = filepath.Join(dir, "main.go")

	e.applyIndentFor(e.buf)
	if !e.indentTabs {
		t.Error("the .editorconfig should win over the file's own look")
	}
	if e.indentFrom != ".editorconfig" {
		t.Errorf("source reported as %q", e.indentFrom)
	}
}

func TestFileContentsOverrideTheConfiguredDefault(t *testing.T) {
	dir := t.TempDir()
	e := newTestEditor("a:", "  b:", "    c")
	e.cfgIndentWidth, e.cfgIndentTabs = 4, false
	e.buf.Filename = filepath.Join(dir, "conf.yml")

	e.applyIndentFor(e.buf)
	if e.indentWidth != 2 {
		t.Errorf("got width %d, want the file's own 2", e.indentWidth)
	}
	if e.indentFrom != "file" {
		t.Errorf("source reported as %q", e.indentFrom)
	}
}

func TestConfiguredDefaultIsUsedWhenNothingElseSpeaks(t *testing.T) {
	dir := t.TempDir()
	e := newTestEditor("flat", "file", "no indent")
	e.cfgIndentWidth, e.cfgIndentTabs = 8, false
	e.buf.Filename = filepath.Join(dir, "notes.txt")

	e.applyIndentFor(e.buf)
	if e.indentWidth != 8 || e.indentFrom != "config" {
		t.Errorf("got %d from %q, want 8 from config", e.indentWidth, e.indentFrom)
	}
}

func TestNearerEditorConfigWins(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".editorconfig"), "root = true\n[*]\nindent_size = 8\n")
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(sub, ".editorconfig"), "[*]\nindent_size = 2\n")

	cfg, ok := readEditorConfig(filepath.Join(sub, "x.txt"))
	if !ok || cfg.size != 2 {
		t.Errorf("got %+v, want the nearer 2", cfg)
	}
}

// ── indentation is actually applied while editing ────────────────────────────

func TestTabInsertsTheDetectedWidth(t *testing.T) {
	e := newTestEditor("a:", "  b")
	e.indentWidth, e.indentTabs = 2, false
	e.cursor = Pos{0, 2}
	e.handleKey(key(tcell.KeyTab))
	if got := e.buf.Line(0); got != "a:  " {
		t.Errorf("got %q, want two spaces of indent", got)
	}
}

func TestTabInsertsATabWhenTheFileIsTabbed(t *testing.T) {
	e := newTestEditor("func f() {")
	e.indentTabs = true
	e.cursor = Pos{0, 10}
	e.handleKey(key(tcell.KeyTab))
	if got := e.buf.Line(0); got != "func f() {\t" {
		t.Errorf("got %q, want a tab", got)
	}
}

func TestEnterIndentsByTheDetectedWidth(t *testing.T) {
	e := newTestEditor("items:")
	e.indentWidth = 2
	e.cursor = Pos{0, 6}
	e.handleKey(key(tcell.KeyEnter))
	if got := e.buf.Line(1); got != "  " {
		t.Errorf("got %q, want two spaces", got)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
