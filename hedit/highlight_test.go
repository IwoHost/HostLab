package main

import (
	"strings"
	"testing"
)

// kindRunes renders a highlighted line as one letter per character, so a test
// can state what it expects as a picture of the line:
//
//	.  normal    k  keyword   t  type
//	b  builtin   s  string    c  comment   n  number
func kindRunes(kinds []tokenKind) string {
	var b strings.Builder
	for _, k := range kinds {
		switch k {
		case tkKeyword:
			b.WriteByte('k')
		case tkType:
			b.WriteByte('t')
		case tkBuiltin:
			b.WriteByte('b')
		case tkString:
			b.WriteByte('s')
		case tkComment:
			b.WriteByte('c')
		case tkNumber:
			b.WriteByte('n')
		default:
			b.WriteByte('.')
		}
	}
	return b.String()
}

// hl highlights one line from a clean state.
func hl(t *testing.T, filename, line string) string {
	t.Helper()
	lang := detectLang(filename)
	if lang == nil {
		t.Fatalf("no language detected for %q", filename)
	}
	kinds, _ := highlight(line, lang, hlNormal)
	return kindRunes(kinds)
}

// spanOf reports the marker letters covering the first occurrence of sub.
func spanOf(line, marks, sub string) string {
	i := strings.Index(line, sub)
	if i < 0 {
		return ""
	}
	return marks[i : i+len(sub)]
}

func allSame(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != c {
			return false
		}
	}
	return len(s) > 0
}

// ── detection ────────────────────────────────────────────────────────────────

func TestDetectLangCoversTheFilesThisRepoIsMadeOf(t *testing.T) {
	cases := map[string]*langDef{
		"index.html":     hlLangHTML,
		"page.htm":       hlLangHTML,
		"icon.svg":       hlLangHTML,
		"feed.xml":       hlLangHTML,
		"style.css":      hlLangCSS,
		"theme.scss":     hlLangCSS,
		"old.less":       hlLangCSS,
		"README.md":      hlLangMD,
		"notes.markdown": hlLangMD,
		"deploy.yml":     hlLangYAML,
		"config.yaml":    hlLangYAML,
		"wrangler.toml":  hlLangTOML,
		"settings.ini":   hlLangTOML,
	}
	for name, want := range cases {
		if got := detectLang(name); got != want {
			t.Errorf("%s: wrong language", name)
		}
	}
	if detectLang("notes.txt") != nil {
		t.Error("a plain .txt should stay unhighlighted")
	}
}

// ── HTML ─────────────────────────────────────────────────────────────────────

func TestHTMLTagAttributeAndValue(t *testing.T) {
	line := `<a href="index.html" class="card">text</a>`
	marks := hl(t, "x.html", line)

	if got := spanOf(line, marks, "<a"); got != "kk" {
		t.Errorf("tag opener marked %q, want keyword", got)
	}
	if got := spanOf(line, marks, "href"); !allSame(got, 't') {
		t.Errorf("attribute marked %q, want type", got)
	}
	if got := spanOf(line, marks, `"index.html"`); !allSame(got, 's') {
		t.Errorf("attribute value marked %q, want string", got)
	}
	if got := spanOf(line, marks, "text"); !allSame(got, '.') {
		t.Errorf("body text marked %q, want normal", got)
	}
	if got := spanOf(line, marks, "</a>"); !allSame(got, 'k') {
		t.Errorf("closing tag marked %q, want keyword", got)
	}
}

func TestHTMLEntity(t *testing.T) {
	line := "a &amp; b"
	marks := hl(t, "x.html", line)
	if got := spanOf(line, marks, "&amp;"); !allSame(got, 'b') {
		t.Errorf("entity marked %q, want builtin", got)
	}
}

func TestHTMLCommentCarriesAcrossLines(t *testing.T) {
	lang := detectLang("x.html")
	k1, st := highlight("<!-- start", lang, hlNormal)
	if !allSame(kindRunes(k1), 'c') {
		t.Errorf("first line %q, want all comment", kindRunes(k1))
	}
	if st != hlBlockCmt {
		t.Fatal("comment should carry to the next line")
	}
	k2, st := highlight("still comment -->", lang, st)
	if !allSame(kindRunes(k2), 'c') {
		t.Errorf("second line %q, want all comment", kindRunes(k2))
	}
	if st != hlNormal {
		t.Error("comment should close on -->")
	}
}

// A tag split over several lines is normal in hand-written HTML.
func TestHTMLTagCarriesAcrossLines(t *testing.T) {
	lang := detectLang("x.html")
	line1 := `<button class="btn"`
	k1, st := highlight(line1, lang, hlNormal)
	if st != hlTag {
		t.Fatal("an unclosed tag should carry")
	}
	if got := spanOf(line1, kindRunes(k1), "class"); !allSame(got, 't') {
		t.Errorf("attribute marked %q", got)
	}
	line2 := `  id="go">label`
	k2, st := highlight(line2, lang, st)
	marks2 := kindRunes(k2)
	if got := spanOf(line2, marks2, "id"); !allSame(got, 't') {
		t.Errorf("attribute on the next line marked %q, want type", got)
	}
	if got := spanOf(line2, marks2, "label"); !allSame(got, '.') {
		t.Errorf("text after the tag marked %q, want normal", got)
	}
	if st != hlNormal {
		t.Error("the tag should close on >")
	}
}

// ── CSS ──────────────────────────────────────────────────────────────────────

func TestCSSPropertyValueAndColour(t *testing.T) {
	line := "  background-color: #1a2b3c;"
	marks := hl(t, "x.css", line)
	if got := spanOf(line, marks, "background-color"); !allSame(got, 't') {
		t.Errorf("property marked %q, want type", got)
	}
	if got := spanOf(line, marks, "#1a2b3c"); !allSame(got, 'n') {
		t.Errorf("hex colour marked %q, want number", got)
	}
}

func TestCSSAtRuleAndComment(t *testing.T) {
	line := "@media (hover: hover) { /* note */ }"
	marks := hl(t, "x.css", line)
	if got := spanOf(line, marks, "@media"); !allSame(got, 'k') {
		t.Errorf("at-rule marked %q, want keyword", got)
	}
	if got := spanOf(line, marks, "/* note */"); !allSame(got, 'c') {
		t.Errorf("comment marked %q", got)
	}
}

func TestCSSKeywordValues(t *testing.T) {
	line := "display: flex;"
	marks := hl(t, "x.css", line)
	if got := spanOf(line, marks, "flex"); !allSame(got, 'b') {
		t.Errorf("value marked %q, want builtin", got)
	}
}

// ── Markdown ─────────────────────────────────────────────────────────────────

func TestMarkdownHeading(t *testing.T) {
	marks := hl(t, "x.md", "## A heading")
	if !allSame(marks, 'k') {
		t.Errorf("heading marked %q, want all keyword", marks)
	}
}

func TestMarkdownInlineCodeAndLink(t *testing.T) {
	line := "see `go test` and [docs](https://x.dev)"
	marks := hl(t, "x.md", line)
	if got := spanOf(line, marks, "`go test`"); !allSame(got, 's') {
		t.Errorf("inline code marked %q, want string", got)
	}
	if got := spanOf(line, marks, "(https://x.dev)"); !allSame(got, 's') {
		t.Errorf("link target marked %q, want string", got)
	}
}

func TestMarkdownFenceCarriesAcrossLines(t *testing.T) {
	lang := detectLang("x.md")
	_, st := highlight("```go", lang, hlNormal)
	if st != hlFence {
		t.Fatal("a fence should open")
	}
	k, st := highlight("# not a heading in here", lang, st)
	if !allSame(kindRunes(k), 's') {
		t.Errorf("fenced line marked %q, want all string", kindRunes(k))
	}
	if st != hlFence {
		t.Error("the fence should still be open")
	}
	_, st = highlight("```", lang, st)
	if st != hlNormal {
		t.Error("the fence should close")
	}
}

func TestMarkdownBulletAndQuote(t *testing.T) {
	line := "- an item"
	marks := hl(t, "x.md", line)
	if marks[0] != 'b' {
		t.Errorf("bullet marked %q, want builtin", marks[0:1])
	}
	if !allSame(hl(t, "x.md", "> quoted"), 'c') {
		t.Error("a quote line should read as a comment")
	}
}

// ── YAML ─────────────────────────────────────────────────────────────────────

func TestYAMLKeyAndValue(t *testing.T) {
	line := "  timeout: 30"
	marks := hl(t, "x.yml", line)
	if got := spanOf(line, marks, "timeout"); !allSame(got, 't') {
		t.Errorf("key marked %q, want type", got)
	}
	if got := spanOf(line, marks, "30"); !allSame(got, 'n') {
		t.Errorf("number marked %q", got)
	}
}

func TestYAMLComment(t *testing.T) {
	line := "key: value # why"
	marks := hl(t, "x.yml", line)
	if got := spanOf(line, marks, "# why"); !allSame(got, 'c') {
		t.Errorf("comment marked %q", got)
	}
}

func TestYAMLListItemKey(t *testing.T) {
	line := "  - name: build"
	marks := hl(t, "x.yml", line)
	if got := spanOf(line, marks, "name"); !allSame(got, 't') {
		t.Errorf("key under a list marker marked %q, want type", got)
	}
}

func TestYAMLBooleans(t *testing.T) {
	line := "enabled: true"
	marks := hl(t, "x.yml", line)
	if got := spanOf(line, marks, "true"); !allSame(got, 'b') {
		t.Errorf("boolean marked %q, want builtin", got)
	}
}

// ── TOML ─────────────────────────────────────────────────────────────────────

func TestTOMLTableHeader(t *testing.T) {
	line := "[build.upload]"
	marks := hl(t, "x.toml", line)
	if !allSame(marks, 'k') {
		t.Errorf("table header marked %q, want keyword", marks)
	}
}

func TestTOMLKeyAndString(t *testing.T) {
	line := `name = "hedit"`
	marks := hl(t, "x.toml", line)
	if got := spanOf(line, marks, "name"); !allSame(got, 't') {
		t.Errorf("key marked %q, want type", got)
	}
	if got := spanOf(line, marks, `"hedit"`); !allSame(got, 's') {
		t.Errorf("string marked %q", got)
	}
}

func TestTOMLComment(t *testing.T) {
	line := "# a note"
	if !allSame(hl(t, "x.toml", line), 'c') {
		t.Error("a whole-line comment should read as a comment")
	}
}

// ── the existing languages still work ────────────────────────────────────────

func TestGoStillHighlights(t *testing.T) {
	line := `func main() { s := "hi" }`
	marks := hl(t, "x.go", line)
	if got := spanOf(line, marks, "func"); !allSame(got, 'k') {
		t.Errorf("keyword marked %q", got)
	}
	if got := spanOf(line, marks, `"hi"`); !allSame(got, 's') {
		t.Errorf("string marked %q", got)
	}
}

// Nothing should wander off the end of a line, whatever the language.
func TestHighlightLengthMatchesTheLine(t *testing.T) {
	samples := map[string]string{
		"x.html": `<div class="a">b</div>`,
		"x.css":  "a { color: #fff; }",
		"x.md":   "# h `c` [l](u) **b**",
		"x.yml":  "k: v # c",
		"x.toml": `[t]` + "\n",
		"x.go":   `func f() {}`,
	}
	for name, line := range samples {
		lang := detectLang(name)
		kinds, _ := highlight(line, lang, hlNormal)
		if len(kinds) != len([]rune(line)) {
			t.Errorf("%s: %d kinds for %d runes", name, len(kinds), len([]rune(line)))
		}
	}
}

// Unterminated things must not panic or run away.
func TestHighlightHandlesRaggedInput(t *testing.T) {
	ragged := []string{"<", "<<", "&", "&amp", "#", "```", "[", "](", `"`, "'", "[t", "k:", "= v", "@", "/*"}
	for _, name := range []string{"x.html", "x.css", "x.md", "x.yml", "x.toml"} {
		lang := detectLang(name)
		for _, line := range ragged {
			kinds, _ := highlight(line, lang, hlNormal)
			if len(kinds) != len([]rune(line)) {
				t.Errorf("%s with %q: %d kinds for %d runes", name, line, len(kinds), len([]rune(line)))
			}
		}
	}
}
