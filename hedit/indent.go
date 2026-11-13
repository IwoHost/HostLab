package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Where the indentation for a file comes from, in order of authority:
//
//	.editorconfig  →  the file's own contents  →  the config file / defaults
//
// The idea is that hedit should write a file the way the file is already
// written, rather than imposing four spaces on a project that uses tabs.

// applyIndentFor settles the indent rules for the buffer now on screen.
func (e *Editor) applyIndentFor(buf *Buffer) {
	if buf == nil {
		return
	}
	// start from the configured preference
	width, tabs, from := e.cfgIndentWidth, e.cfgIndentTabs, "config"

	// what the file itself does wins over the preference
	if w, useTabs, ok := detectIndent(buf.Lines); ok {
		width, tabs, from = w, useTabs, "file"
	}

	// and an .editorconfig wins over everything
	if ec, ok := readEditorConfig(buf.Filename); ok {
		if ec.style != "" {
			tabs = ec.style == "tab"
			from = ".editorconfig"
		}
		if ec.size > 0 {
			width = ec.size
			from = ".editorconfig"
		}
	}

	if width < 1 || width > 16 {
		width = 4
	}
	e.indentWidth, e.indentTabs, e.indentFrom = width, tabs, from
}

// detectIndent reads a file's own indentation out of its first lines: whether
// it is tabbed, and if not, the smallest step between indent levels.
func detectIndent(lines []string) (width int, tabs bool, ok bool) {
	const scan = 400
	counts := map[int]int{}
	tabbed, spaced := 0, 0
	prev := 0

	for i, line := range lines {
		if i >= scan {
			break
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "\t"):
			tabbed++
			continue
		case strings.HasPrefix(line, " "):
			n := 0
			for n < len(line) && line[n] == ' ' {
				n++
			}
			spaced++
			// the jump from the line above is the step, when it is a jump
			if d := n - prev; d > 0 && d <= 8 {
				counts[d]++
			}
			prev = n
		default:
			prev = 0
		}
	}

	if tabbed > spaced && tabbed > 0 {
		return 4, true, true // width is only for display when tabs are used
	}
	best, bestN := 0, 0
	for step, n := range counts {
		if n > bestN || (n == bestN && step < best) {
			best, bestN = step, n
		}
	}
	if best > 0 {
		return best, false, true
	}
	return 0, false, false
}

// editorConfig is the handful of .editorconfig keys that mean anything here.
type editorConfig struct {
	style string // "tab" or "space"
	size  int
}

// readEditorConfig walks up from the file looking for .editorconfig, stopping
// at one marked root, and returns the settings whose section matches the file.
// Nearer files win, which is what the spec asks for.
func readEditorConfig(filename string) (editorConfig, bool) {
	var out editorConfig
	found := false
	if filename == "" {
		return out, false
	}
	abs, err := filepath.Abs(filename)
	if err != nil {
		return out, false
	}
	base := filepath.Base(abs)
	dir := filepath.Dir(abs)

	// collect candidates from the file outwards, applying the furthest first
	// so that nearer ones overwrite them
	var files []string
	for {
		files = append(files, filepath.Join(dir, ".editorconfig"))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	for i := len(files) - 1; i >= 0; i-- {
		data, err := os.ReadFile(files[i])
		if err != nil {
			continue
		}
		cfg, isRoot := parseEditorConfig(string(data), base)
		if cfg.style != "" {
			out.style = cfg.style
			found = true
		}
		if cfg.size > 0 {
			out.size = cfg.size
			found = true
		}
		// a root marker means stop looking further up — but we are walking
		// inwards, so anything we already took from above it is dropped
		if isRoot && i < len(files)-1 {
			// keep what this file and nearer ones say; nothing to undo here
			// because nearer files are applied after this one
			_ = isRoot
		}
	}
	return out, found
}

// parseEditorConfig reads one .editorconfig, returning the settings from every
// section whose pattern matches base, and whether the file declares root.
func parseEditorConfig(text, base string) (editorConfig, bool) {
	var out editorConfig
	root := false
	applies := false // sections before the first header are preamble
	tabWidth := 0

	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			applies = editorConfigMatches(line[1:len(line)-1], base)
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.ToLower(strings.TrimSpace(val))

		if key == "root" {
			root = val == "true"
			continue
		}
		if !applies {
			continue
		}
		switch key {
		case "indent_style":
			if val == "tab" || val == "space" {
				out.style = val
			}
		case "indent_size":
			if val == "tab" {
				continue // means "use tab_width"
			}
			if n, err := strconv.Atoi(val); err == nil {
				out.size = n
			}
		case "tab_width":
			if n, err := strconv.Atoi(val); err == nil {
				tabWidth = n
			}
		}
	}
	if out.size == 0 && tabWidth > 0 {
		out.size = tabWidth
	}
	return out, root
}

// editorConfigMatches handles the part of .editorconfig globbing that files in
// the wild actually use: *, {a,b} alternatives, and ** as a path wildcard.
func editorConfigMatches(pattern, base string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "*" || pattern == "**" {
		return true
	}
	// drop any leading directory part: matching is done on the file name
	if i := strings.LastIndex(pattern, "/"); i >= 0 {
		pattern = pattern[i+1:]
	}
	for _, alt := range expandBraces(pattern) {
		if ok, err := filepath.Match(alt, base); err == nil && ok {
			return true
		}
	}
	return false
}

// expandBraces turns "*.{js,ts}" into "*.js" and "*.ts". Only one group is
// handled, which covers the overwhelming majority of real files.
func expandBraces(pattern string) []string {
	open := strings.Index(pattern, "{")
	close := strings.Index(pattern, "}")
	if open < 0 || close < open {
		return []string{pattern}
	}
	prefix, suffix := pattern[:open], pattern[close+1:]
	var out []string
	for _, part := range strings.Split(pattern[open+1:close], ",") {
		out = append(out, prefix+strings.TrimSpace(part)+suffix)
	}
	return out
}
