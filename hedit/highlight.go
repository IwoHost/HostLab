package main

import (
	"path/filepath"
	"strings"
	"unicode"
)

// ── TOKEN KINDS ───────────────────────────────────────────────────────────────

type tokenKind uint8

const (
	tkNormal  tokenKind = iota
	tkKeyword           // language keyword
	tkType              // built-in type
	tkBuiltin           // built-in function / constant
	tkString            // string literal
	tkComment           // comment
	tkNumber            // numeric literal
)

// ── INTER-LINE CARRY STATE ────────────────────────────────────────────────────

type hlState uint8

const (
	hlNormal   hlState = iota
	hlBlockCmt         // inside /* … */
	hlRawStr           // inside Go/JS backtick `…`
	hlTripleDQ         // inside Python """…"""
	hlTripleSQ         // inside Python '''…'''
	hlTag              // inside an HTML/XML tag that runs past the line end
	hlFence            // inside a ``` fenced block in Markdown
)

// ── LANGUAGE DEFINITION ───────────────────────────────────────────────────────

type langDef struct {
	keywords    map[string]bool
	types       map[string]bool
	builtins    map[string]bool
	lineComment string // "#", "//", "--"
	blockOpen   string // "/*"
	blockClose  string // "*/"
	rawStrings  bool   // backtick strings that can span lines
	tripleStrs  bool   // Python """ / ''' strings that can span lines
	noSingleQ   bool   // JSON: no single-quoted strings
	ppDirective bool   // C/C++: treat #word as keyword
	caseInsKw   bool   // SQL: look up strings.ToUpper(word) in sets
	atDirective bool   // CSS: treat @word as a keyword
	hashColor   bool   // CSS: #rrggbb reads as a number
	markup      bool   // HTML/XML: tags, attributes, entities
	markdown    bool   // Markdown: a line-oriented pass of its own
	dashWords   bool   // CSS: a hyphen is part of a word (background-color)
	valueColon  bool   // CSS: what follows ":" is a value, so prefer builtins
	keyColon    bool   // YAML: "key:" at the head of a line is a key
	tomlKeys    bool   // TOML: [table] headers and "key =" are keys
}

func mkWords(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

// ── HELPERS ───────────────────────────────────────────────────────────────────

// hasPrefixAt reports whether runes[i:] starts with the rune sequence of s.
func hasPrefixAt(runes []rune, i int, s string) bool {
	sr := []rune(s)
	if i+len(sr) > len(runes) {
		return false
	}
	for j, r := range sr {
		if runes[i+j] != r {
			return false
		}
	}
	return true
}

func isHexRune(r rune) bool {
	return (r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') ||
		r == '_'
}

// ── TOKENIZER ─────────────────────────────────────────────────────────────────

// highlight tokenizes one source line given carry-in state.
// Returns one tokenKind per rune and the carry-out state for the next line.
func highlight(line string, lang *langDef, in hlState) ([]tokenKind, hlState) {
	if lang == nil {
		return nil, hlNormal
	}
	runes := []rune(line)
	n := len(runes)
	kinds := make([]tokenKind, n)
	state := in
	i := 0
	afterColon := false // CSS: between ":" and ";" we are reading a value

	// Markdown is a line-oriented format rather than a stream of tokens, so it
	// gets a pass of its own instead of being bent through the scanner below.
	if lang.markdown {
		return highlightMarkdown(runes, kinds, state)
	}

	// YAML keys and TOML headers are decided by where they sit on the line, so
	// they are claimed before the general scan starts.
	if lang.keyColon {
		i = markYAMLKey(runes, kinds)
	} else if lang.tomlKeys {
		i = markTOMLKey(runes, kinds)
	}

outer:
	for i < n {
		// ── multi-line continuation states ────────────────────────────────
		switch state {
		case hlBlockCmt:
			if lang.blockClose != "" && hasPrefixAt(runes, i, lang.blockClose) {
				cl := len([]rune(lang.blockClose))
				for j := 0; j < cl && i < n; j++ {
					kinds[i] = tkComment
					i++
				}
				state = hlNormal
			} else {
				kinds[i] = tkComment
				i++
			}
			continue outer

		case hlRawStr:
			kinds[i] = tkString
			if runes[i] == '`' {
				i++
				state = hlNormal
			} else {
				i++
			}
			continue outer

		case hlTripleDQ:
			if hasPrefixAt(runes, i, `"""`) {
				kinds[i], kinds[i+1], kinds[i+2] = tkString, tkString, tkString
				i += 3
				state = hlNormal
			} else {
				kinds[i] = tkString
				i++
			}
			continue outer

		case hlTag:
			i = scanTagBody(runes, kinds, i, &state)
			continue outer

		case hlFence:
			for ; i < n; i++ {
				kinds[i] = tkString
			}
			return kinds, hlFence

		case hlTripleSQ:
			if hasPrefixAt(runes, i, `'''`) {
				kinds[i], kinds[i+1], kinds[i+2] = tkString, tkString, tkString
				i += 3
				state = hlNormal
			} else {
				kinds[i] = tkString
				i++
			}
			continue outer
		}

		// ── normal mode ───────────────────────────────────────────────────

		// line comment → rest of line is comment; no carry-over
		if lang.lineComment != "" && hasPrefixAt(runes, i, lang.lineComment) {
			for ; i < n; i++ {
				kinds[i] = tkComment
			}
			return kinds, hlNormal
		}

		// block comment open
		if lang.blockOpen != "" && hasPrefixAt(runes, i, lang.blockOpen) {
			ol := len([]rune(lang.blockOpen))
			for j := 0; j < ol && i < n; j++ {
				kinds[i] = tkComment
				i++
			}
			state = hlBlockCmt
			continue outer
		}

		// markup: <tag …>, </tag>, and &entity;
		if lang.markup {
			if runes[i] == '<' {
				kinds[i] = tkKeyword
				i++
				if i < n && runes[i] == '/' {
					kinds[i] = tkKeyword
					i++
				}
				for i < n && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) ||
					runes[i] == '-' || runes[i] == '_' || runes[i] == ':' || runes[i] == '!') {
					kinds[i] = tkKeyword
					i++
				}
				state = hlTag
				i = scanTagBody(runes, kinds, i, &state)
				continue outer
			}
			if runes[i] == '&' {
				start := i
				j := i + 1
				for j < n && j-start < 12 && runes[j] != ';' && runes[j] != ' ' {
					j++
				}
				if j < n && runes[j] == ';' {
					for ; i <= j; i++ {
						kinds[i] = tkBuiltin
					}
					continue outer
				}
			}
		}

		// CSS: @media, @import …
		if lang.atDirective && runes[i] == '@' {
			kinds[i] = tkKeyword
			i++
			for i < n && (unicode.IsLetter(runes[i]) || runes[i] == '-') {
				kinds[i] = tkKeyword
				i++
			}
			continue outer
		}

		// CSS: #rrggbb is a value, not a comment or an id here
		if lang.hashColor && runes[i] == '#' && i+1 < n && isHexRune(runes[i+1]) {
			kinds[i] = tkNumber
			i++
			for i < n && isHexRune(runes[i]) {
				kinds[i] = tkNumber
				i++
			}
			continue outer
		}

		// preprocessor directive: #word (C/C++)
		if lang.ppDirective && runes[i] == '#' {
			kinds[i] = tkKeyword
			i++
			for i < n && (unicode.IsLetter(runes[i]) || runes[i] == '_') {
				kinds[i] = tkKeyword
				i++
			}
			continue outer
		}

		// backtick / raw string
		if lang.rawStrings && runes[i] == '`' {
			kinds[i] = tkString
			i++
			for i < n {
				kinds[i] = tkString
				if runes[i] == '`' {
					i++
					continue outer
				}
				i++
			}
			state = hlRawStr
			continue outer
		}

		// Python triple-quoted strings — check BEFORE single-char quote handling
		if lang.tripleStrs {
			if hasPrefixAt(runes, i, `"""`) {
				kinds[i], kinds[i+1], kinds[i+2] = tkString, tkString, tkString
				i += 3
				for i < n {
					if hasPrefixAt(runes, i, `"""`) {
						kinds[i], kinds[i+1], kinds[i+2] = tkString, tkString, tkString
						i += 3
						state = hlNormal
						continue outer
					}
					kinds[i] = tkString
					i++
				}
				state = hlTripleDQ
				continue outer
			}
			if hasPrefixAt(runes, i, `'''`) {
				kinds[i], kinds[i+1], kinds[i+2] = tkString, tkString, tkString
				i += 3
				for i < n {
					if hasPrefixAt(runes, i, `'''`) {
						kinds[i], kinds[i+1], kinds[i+2] = tkString, tkString, tkString
						i += 3
						state = hlNormal
						continue outer
					}
					kinds[i] = tkString
					i++
				}
				state = hlTripleSQ
				continue outer
			}
		}

		// single/double-quoted string (single-line; no carry-over when unterminated)
		if runes[i] == '"' || (!lang.noSingleQ && runes[i] == '\'') {
			quote := runes[i]
			kinds[i] = tkString
			i++
			for i < n {
				kinds[i] = tkString
				ch := runes[i]
				if ch == '\\' {
					i++
					if i < n {
						kinds[i] = tkString
						i++
					}
				} else if ch == quote {
					i++
					break
				} else {
					i++
				}
			}
			continue outer
		}

		// number literal
		if unicode.IsDigit(runes[i]) {
			kinds[i] = tkNumber
			start := i
			i++
			// 0x / 0b / 0o prefix
			if runes[start] == '0' && i < n {
				switch runes[i] {
				case 'x', 'X', 'b', 'B', 'o', 'O':
					kinds[i] = tkNumber
					i++
					for i < n && isHexRune(runes[i]) {
						kinds[i] = tkNumber
						i++
					}
					continue outer
				}
			}
			for i < n {
				r := runes[i]
				if unicode.IsDigit(r) || r == '.' || r == '_' {
					kinds[i] = tkNumber
					i++
				} else if r == 'e' || r == 'E' {
					kinds[i] = tkNumber
					i++
					if i < n && (runes[i] == '+' || runes[i] == '-') {
						kinds[i] = tkNumber
						i++
					}
				} else {
					break
				}
			}
			continue outer
		}

		// identifier → keyword / type / builtin / normal
		if unicode.IsLetter(runes[i]) || runes[i] == '_' {
			start := i
			for i < n && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_' ||
				(lang.dashWords && runes[i] == '-')) {
				i++
			}
			word := string(runes[start:i])
			lookup := word
			if lang.caseInsKw {
				lookup = strings.ToUpper(word)
			}
			// In CSS a word like "flex" is a property before the colon and a
			// value after it, so which set wins depends on where we are.
			preferBuiltin := lang.valueColon && afterColon
			kind := tkNormal
			switch {
			case lang.keywords[lookup]:
				kind = tkKeyword
			case preferBuiltin && lang.builtins != nil && lang.builtins[lookup]:
				kind = tkBuiltin
			case lang.types != nil && lang.types[lookup]:
				kind = tkType
			case lang.builtins != nil && lang.builtins[lookup]:
				kind = tkBuiltin
			}
			for j := start; j < i; j++ {
				kinds[j] = kind
			}
			continue outer
		}

		if lang.valueColon {
			switch runes[i] {
			case ':':
				afterColon = true
			case ';', '{', '}':
				afterColon = false
			}
		}
		i++ // unrecognized character
	}

	return kinds, state
}

// ── MARKUP, MARKDOWN AND KEY SCANNERS ────────────────────────────────────────

// scanTagBody colours the inside of an HTML/XML tag — attribute names, their
// quoted values — until the closing angle bracket. It leaves state as hlTag if
// the tag runs past the end of the line, which long elements routinely do.
func scanTagBody(runes []rune, kinds []tokenKind, i int, state *hlState) int {
	n := len(runes)
	for i < n {
		switch {
		case runes[i] == '>':
			kinds[i] = tkKeyword
			i++
			*state = hlNormal
			return i

		case runes[i] == '/' && i+1 < n && runes[i+1] == '>':
			kinds[i], kinds[i+1] = tkKeyword, tkKeyword
			i += 2
			*state = hlNormal
			return i

		case runes[i] == '"' || runes[i] == '\'':
			quote := runes[i]
			kinds[i] = tkString
			i++
			for i < n {
				kinds[i] = tkString
				if runes[i] == quote {
					i++
					break
				}
				i++
			}

		case unicode.IsLetter(runes[i]) || runes[i] == '_' || runes[i] == ':' || runes[i] == '@':
			for i < n && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) ||
				runes[i] == '-' || runes[i] == '_' || runes[i] == ':' || runes[i] == '@' || runes[i] == '.') {
				kinds[i] = tkType
				i++
			}

		default:
			i++
		}
	}
	*state = hlTag
	return i
}

// highlightMarkdown handles a line of Markdown: fences and headings claim the
// whole line, then the rest is scanned for inline code, emphasis and links.
func highlightMarkdown(runes []rune, kinds []tokenKind, in hlState) ([]tokenKind, hlState) {
	n := len(runes)
	indent := 0
	for indent < n && (runes[indent] == ' ' || runes[indent] == '\t') {
		indent++
	}

	// a fence opens or closes whatever came before it
	if hasPrefixAt(runes, indent, "```") || hasPrefixAt(runes, indent, "~~~") {
		for i := 0; i < n; i++ {
			kinds[i] = tkString
		}
		if in == hlFence {
			return kinds, hlNormal
		}
		return kinds, hlFence
	}
	if in == hlFence {
		for i := 0; i < n; i++ {
			kinds[i] = tkString
		}
		return kinds, hlFence
	}

	// headings and quotes take the line
	if indent < n && runes[indent] == '#' {
		for i := 0; i < n; i++ {
			kinds[i] = tkKeyword
		}
		return kinds, hlNormal
	}
	if indent < n && runes[indent] == '>' {
		for i := 0; i < n; i++ {
			kinds[i] = tkComment
		}
		return kinds, hlNormal
	}

	i := indent
	// list bullets and numbers
	if i < n && (runes[i] == '-' || runes[i] == '*' || runes[i] == '+') &&
		i+1 < n && runes[i+1] == ' ' {
		kinds[i] = tkBuiltin
		i += 2
	} else if i < n && unicode.IsDigit(runes[i]) {
		j := i
		for j < n && unicode.IsDigit(runes[j]) {
			j++
		}
		if j+1 < n && runes[j] == '.' && runes[j+1] == ' ' {
			for ; i <= j; i++ {
				kinds[i] = tkBuiltin
			}
			i++
		}
	}

	for i < n {
		switch {
		case runes[i] == '`':
			kinds[i] = tkString
			i++
			for i < n {
				kinds[i] = tkString
				if runes[i] == '`' {
					i++
					break
				}
				i++
			}

		case hasPrefixAt(runes, i, "**") || hasPrefixAt(runes, i, "__"):
			kinds[i], kinds[i+1] = tkBuiltin, tkBuiltin
			i += 2

		case runes[i] == '[':
			kinds[i] = tkBuiltin
			i++
			for i < n && runes[i] != ']' {
				i++
			}
			if i < n {
				kinds[i] = tkBuiltin
				i++
			}
			// the (url) half, when it is there, reads as a string
			if i < n && runes[i] == '(' {
				for i < n {
					kinds[i] = tkString
					if runes[i] == ')' {
						i++
						break
					}
					i++
				}
			}

		default:
			i++
		}
	}
	return kinds, hlNormal
}

// markYAMLKey claims "key:" at the head of a line, including under a "- "
// list marker, and returns where the general scan should pick up.
func markYAMLKey(runes []rune, kinds []tokenKind) int {
	n := len(runes)
	i := 0
	for i < n && (runes[i] == ' ' || runes[i] == '\t') {
		i++
	}
	// document separators
	if hasPrefixAt(runes, i, "---") || hasPrefixAt(runes, i, "...") {
		for j := i; j < n; j++ {
			kinds[j] = tkKeyword
		}
		return n
	}
	if i < n && runes[i] == '-' && i+1 < n && runes[i+1] == ' ' {
		kinds[i] = tkBuiltin
		i += 2
		for i < n && runes[i] == ' ' {
			i++
		}
	}
	start := i
	for i < n && runes[i] != ':' && runes[i] != '#' {
		i++
	}
	if i < n && runes[i] == ':' && i > start {
		for j := start; j < i; j++ {
			kinds[j] = tkType
		}
		return i // leave the colon and the value to the general scan
	}
	return start
}

// markTOMLKey claims a [table] header or a "key =" at the head of a line.
func markTOMLKey(runes []rune, kinds []tokenKind) int {
	n := len(runes)
	i := 0
	for i < n && (runes[i] == ' ' || runes[i] == '\t') {
		i++
	}
	if i < n && runes[i] == '[' {
		for j := i; j < n; j++ {
			kinds[j] = tkKeyword
			if runes[j] == ']' {
				return j + 1
			}
		}
		return n
	}
	start := i
	for i < n && runes[i] != '=' && runes[i] != '#' {
		i++
	}
	if i < n && runes[i] == '=' && i > start {
		for j := start; j < i; j++ {
			if runes[j] != ' ' && runes[j] != '\t' {
				kinds[j] = tkType
			}
		}
		return i
	}
	return start
}

// ── LANGUAGE DEFINITIONS ──────────────────────────────────────────────────────

var (
	hlLangGo     = buildLangGo()
	hlLangPython = buildLangPython()
	hlLangJS     = buildLangJS()
	hlLangC      = buildLangC()
	hlLangRust   = buildLangRust()
	hlLangShell  = buildLangShell()
	hlLangJSON   = buildLangJSON()
	hlLangSQL    = buildLangSQL()
	hlLangHTML   = buildLangHTML()
	hlLangCSS    = buildLangCSS()
	hlLangMD     = buildLangMarkdown()
	hlLangYAML   = buildLangYAML()
	hlLangTOML   = buildLangTOML()
)

func buildLangGo() *langDef {
	return &langDef{
		lineComment: "//",
		blockOpen:   "/*",
		blockClose:  "*/",
		rawStrings:  true,
		keywords: mkWords(
			"break", "case", "chan", "const", "continue", "default", "defer",
			"else", "fallthrough", "for", "func", "go", "goto", "if", "import",
			"interface", "map", "package", "range", "return", "select", "struct",
			"switch", "type", "var",
		),
		types: mkWords(
			"bool", "byte", "complex64", "complex128", "error",
			"float32", "float64",
			"int", "int8", "int16", "int32", "int64",
			"rune", "string",
			"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
			"any", "comparable",
		),
		builtins: mkWords(
			"append", "cap", "clear", "close", "copy", "delete", "imag", "len",
			"make", "max", "min", "new", "panic", "print", "println", "real", "recover",
			"false", "true", "nil", "iota",
		),
	}
}

func buildLangPython() *langDef {
	return &langDef{
		lineComment: "#",
		tripleStrs:  true,
		keywords: mkWords(
			"and", "as", "assert", "async", "await", "break", "class", "continue",
			"def", "del", "elif", "else", "except", "finally", "for", "from",
			"global", "if", "import", "in", "is", "lambda", "nonlocal", "not",
			"or", "pass", "raise", "return", "try", "while", "with", "yield",
			"match", "case",
		),
		types: mkWords(
			"int", "str", "float", "complex", "list", "dict", "tuple", "set",
			"frozenset", "bool", "bytes", "bytearray", "memoryview", "type",
			"object", "Exception", "BaseException", "ValueError", "TypeError",
			"KeyError", "IndexError", "AttributeError", "RuntimeError",
			"StopIteration", "OSError", "IOError", "FileNotFoundError",
		),
		builtins: mkWords(
			"abs", "all", "any", "bin", "callable", "chr", "compile",
			"delattr", "dir", "divmod", "enumerate", "eval", "exec", "filter",
			"format", "getattr", "globals", "hasattr", "hash", "help", "hex",
			"id", "input", "isinstance", "issubclass", "iter", "len", "locals",
			"map", "max", "min", "next", "oct", "open", "ord", "pow", "print",
			"range", "repr", "reversed", "round", "setattr", "slice", "sorted",
			"staticmethod", "classmethod", "property", "sum", "super", "vars",
			"zip", "True", "False", "None",
		),
	}
}

func buildLangJS() *langDef {
	return &langDef{
		lineComment: "//",
		blockOpen:   "/*",
		blockClose:  "*/",
		rawStrings:  true, // template literals
		keywords: mkWords(
			"async", "await", "break", "case", "catch", "class", "const",
			"continue", "debugger", "default", "delete", "do", "else", "export",
			"extends", "finally", "for", "from", "function", "if", "import",
			"in", "instanceof", "let", "new", "of", "return", "static", "super",
			"switch", "this", "throw", "try", "typeof", "var", "void", "while",
			"with", "yield",
			// TypeScript extras
			"abstract", "as", "declare", "enum", "implements", "interface",
			"keyof", "namespace", "readonly", "satisfies", "type", "override",
		),
		types: mkWords(
			"string", "number", "boolean", "void", "null", "undefined", "never",
			"any", "unknown", "object", "symbol", "bigint",
			"Array", "Promise", "Object", "Function", "Symbol", "Map", "Set",
			"WeakMap", "WeakSet", "Error", "Date", "RegExp", "URL",
		),
		builtins: mkWords(
			"console", "Math", "JSON", "parseInt", "parseFloat",
			"isNaN", "isFinite", "encodeURI", "decodeURI",
			"encodeURIComponent", "decodeURIComponent",
			"setTimeout", "clearTimeout", "setInterval", "clearInterval",
			"fetch", "Proxy", "Reflect",
			"true", "false", "null", "undefined", "NaN", "Infinity",
			"window", "document", "global", "globalThis", "process",
			"require", "module", "exports",
		),
	}
}

func buildLangC() *langDef {
	return &langDef{
		lineComment: "//",
		blockOpen:   "/*",
		blockClose:  "*/",
		ppDirective: true,
		keywords: mkWords(
			"auto", "break", "case", "const", "continue", "default", "do",
			"else", "enum", "extern", "for", "goto", "if", "inline", "register",
			"restrict", "return", "sizeof", "static", "struct", "switch",
			"typedef", "union", "volatile", "while",
			// C++
			"alignas", "alignof", "and", "and_eq", "bitand", "bitor", "catch",
			"class", "compl", "concept", "consteval", "constexpr", "constinit",
			"delete", "dynamic_cast", "explicit", "export", "friend", "mutable",
			"namespace", "new", "noexcept", "not", "not_eq", "nullptr",
			"operator", "or", "or_eq", "private", "protected", "public",
			"reinterpret_cast", "requires", "static_assert", "static_cast",
			"template", "this", "thread_local", "throw", "try", "typeid",
			"typename", "using", "virtual", "xor", "xor_eq",
		),
		types: mkWords(
			"bool", "char", "double", "float", "int", "long", "short",
			"signed", "unsigned", "void", "wchar_t", "auto",
			"int8_t", "int16_t", "int32_t", "int64_t",
			"uint8_t", "uint16_t", "uint32_t", "uint64_t",
			"size_t", "ptrdiff_t", "intptr_t", "uintptr_t",
			"string", "vector", "map", "set", "list", "array", "pair",
			"shared_ptr", "unique_ptr", "weak_ptr", "optional", "variant",
		),
		builtins: mkWords(
			"NULL", "nullptr", "true", "false", "EOF", "STDIN_FILENO",
			"printf", "fprintf", "sprintf", "snprintf", "scanf", "fscanf",
			"malloc", "calloc", "realloc", "free",
			"memcpy", "memmove", "memset", "memcmp",
			"strlen", "strcpy", "strncpy", "strcmp", "strncmp", "strcat",
			"abs", "labs", "fabs", "sqrt", "pow", "exit", "abort",
			"assert", "offsetof",
		),
	}
}

func buildLangRust() *langDef {
	return &langDef{
		lineComment: "//",
		blockOpen:   "/*",
		blockClose:  "*/",
		keywords: mkWords(
			"as", "async", "await", "break", "const", "continue", "crate", "dyn",
			"else", "enum", "extern", "false", "fn", "for", "if", "impl", "in",
			"let", "loop", "match", "mod", "move", "mut", "pub", "ref", "return",
			"self", "Self", "static", "struct", "super", "trait", "true", "type",
			"union", "unsafe", "use", "where", "while",
		),
		types: mkWords(
			"bool", "char", "f32", "f64",
			"i8", "i16", "i32", "i64", "i128", "isize",
			"u8", "u16", "u32", "u64", "u128", "usize",
			"str", "String", "Vec", "Box", "Rc", "Arc",
			"Option", "Result", "HashMap", "HashSet", "BTreeMap", "BTreeSet",
			"Cell", "RefCell", "Mutex", "RwLock", "Cow",
		),
		builtins: mkWords(
			"Some", "None", "Ok", "Err",
			"println", "print", "eprintln", "eprint",
			"format", "vec", "assert", "assert_eq", "assert_ne",
			"panic", "todo", "unimplemented", "unreachable", "dbg",
			"write", "writeln", "include_str",
			"Default", "Clone", "Copy", "Debug", "Display",
			"PartialEq", "Eq", "PartialOrd", "Ord", "Hash",
			"From", "Into", "Iterator", "Drop",
		),
	}
}

func buildLangShell() *langDef {
	return &langDef{
		lineComment: "#",
		keywords: mkWords(
			"if", "then", "else", "elif", "fi",
			"for", "while", "until", "do", "done",
			"case", "esac", "in", "function",
			"return", "exit", "break", "continue",
			"local", "declare", "typeset", "readonly", "export",
			"select", "time", "coproc",
		),
		builtins: mkWords(
			"echo", "printf", "read", "cd", "pwd", "ls", "mkdir", "rm",
			"cp", "mv", "cat", "grep", "sed", "awk", "find", "xargs",
			"sort", "uniq", "head", "tail", "wc", "cut", "tr",
			"chmod", "chown", "ln", "source", "test", "true", "false",
			"set", "unset", "shift", "eval", "exec", "env",
			"alias", "unalias", "history", "jobs", "kill", "wait",
			"trap", "getopts", "mapfile", "readarray",
		),
	}
}

func buildLangJSON() *langDef {
	return &langDef{
		noSingleQ: true,
		builtins:  mkWords("true", "false", "null"),
	}
}

func buildLangSQL() *langDef {
	return &langDef{
		lineComment: "--",
		blockOpen:   "/*",
		blockClose:  "*/",
		caseInsKw:   true,
		keywords: mkWords(
			"SELECT", "FROM", "WHERE", "AND", "OR", "NOT", "IN", "IS", "NULL",
			"INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE",
			"CREATE", "TABLE", "ALTER", "DROP", "INDEX", "VIEW",
			"JOIN", "INNER", "OUTER", "LEFT", "RIGHT", "FULL", "CROSS", "ON",
			"AS", "GROUP", "BY", "ORDER", "HAVING", "DISTINCT", "LIMIT", "OFFSET",
			"UNION", "ALL", "INTERSECT", "EXCEPT",
			"EXISTS", "LIKE", "BETWEEN", "CASE", "WHEN", "THEN", "ELSE", "END",
			"BEGIN", "COMMIT", "ROLLBACK", "TRANSACTION", "SAVEPOINT",
			"WITH", "RECURSIVE", "OVER", "PARTITION", "WINDOW",
			"PRIMARY", "KEY", "FOREIGN", "REFERENCES", "UNIQUE", "CHECK",
			"DEFAULT", "CONSTRAINT", "IF",
		),
		types: mkWords(
			"INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT", "SERIAL", "BIGSERIAL",
			"FLOAT", "DOUBLE", "DECIMAL", "NUMERIC", "REAL",
			"CHAR", "VARCHAR", "TEXT", "NCHAR", "NVARCHAR",
			"BLOB", "CLOB", "BINARY", "VARBINARY",
			"DATE", "TIME", "DATETIME", "TIMESTAMP", "INTERVAL",
			"BOOLEAN", "BOOL", "BIT",
			"JSON", "JSONB", "XML", "UUID", "OID", "ARRAY",
		),
		builtins: mkWords(
			"COUNT", "SUM", "AVG", "MIN", "MAX",
			"COALESCE", "NULLIF", "ISNULL", "NVL", "IFNULL",
			"CAST", "CONVERT", "TO_CHAR", "TO_DATE", "TO_NUMBER",
			"NOW", "CURDATE", "CURTIME", "SYSDATE", "CURRENT_TIMESTAMP",
			"YEAR", "MONTH", "DAY", "HOUR", "MINUTE", "SECOND",
			"SUBSTR", "SUBSTRING", "LENGTH", "LEN", "UPPER", "LOWER",
			"TRIM", "LTRIM", "RTRIM", "REPLACE", "CONCAT",
			"ROUND", "FLOOR", "CEIL", "CEILING", "ABS", "POWER", "SQRT",
			"ROW_NUMBER", "RANK", "DENSE_RANK", "LAG", "LEAD",
			"FIRST_VALUE", "LAST_VALUE", "NTILE",
		),
	}
}

// ── LANGUAGE DETECTION ────────────────────────────────────────────────────────

func buildLangHTML() *langDef {
	return &langDef{
		markup:    true,
		blockOpen: "<!--", blockClose: "-->",
	}
}

func buildLangCSS() *langDef {
	return &langDef{
		blockOpen: "/*", blockClose: "*/",
		atDirective: true,
		hashColor:   true,
		dashWords:   true,
		valueColon:  true,
		noSingleQ:   false,
		keywords: mkWords(
			"important", "from", "to", "and", "or", "not", "only",
		),
		types: mkWords(
			// the properties you actually reach for
			"color", "background", "background-color", "background-image", "background-size",
			"background-position", "background-repeat", "border", "border-radius", "border-color",
			"border-width", "border-style", "border-top", "border-right", "border-bottom", "border-left",
			"margin", "margin-top", "margin-right", "margin-bottom", "margin-left",
			"padding", "padding-top", "padding-right", "padding-bottom", "padding-left",
			"width", "height", "min-width", "min-height", "max-width", "max-height",
			"display", "position", "top", "right", "bottom", "left", "inset", "z-index",
			"flex", "flex-direction", "flex-wrap", "flex-grow", "flex-shrink", "flex-basis",
			"grid", "grid-template-columns", "grid-template-rows", "grid-column", "grid-row", "gap",
			"align-items", "align-content", "align-self", "justify-content", "justify-items", "justify-self",
			"font", "font-family", "font-size", "font-weight", "font-style", "line-height",
			"letter-spacing", "word-spacing", "text-align", "text-decoration", "text-transform",
			"text-shadow", "box-shadow", "box-sizing", "opacity", "overflow", "overflow-x", "overflow-y",
			"transition", "transform", "transform-origin", "animation", "animation-delay", "animation-name",
			"cursor", "content", "visibility", "white-space", "list-style", "outline", "outline-offset",
			"filter", "backdrop-filter", "mask", "clip-path", "aspect-ratio", "object-fit", "pointer-events",
		),
		builtins: mkWords(
			"none", "auto", "inherit", "initial", "unset", "block", "inline", "inline-block",
			"flex", "grid", "absolute", "relative", "fixed", "sticky", "static", "hidden", "visible",
			"center", "left", "right", "top", "bottom", "solid", "dashed", "dotted", "transparent",
			"bold", "normal", "italic", "pointer", "ease", "ease-in", "ease-out", "ease-in-out",
			"linear", "infinite", "forwards", "wrap", "nowrap", "cover", "contain", "currentColor",
			"var", "calc", "rgb", "rgba", "hsl", "hsla", "url", "clamp", "min", "max",
			"true", "false",
		),
	}
}

func buildLangMarkdown() *langDef {
	return &langDef{markdown: true}
}

func buildLangYAML() *langDef {
	return &langDef{
		lineComment: "#",
		keyColon:    true,
		builtins: mkWords(
			"true", "false", "yes", "no", "on", "off", "null", "~",
			"True", "False", "Null", "None",
		),
	}
}

func buildLangTOML() *langDef {
	return &langDef{
		lineComment: "#",
		tomlKeys:    true,
		builtins:    mkWords("true", "false", "inf", "nan"),
	}
}

func detectLang(filename string) *langDef {
	if filename == "" {
		return nil
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".go":
		return hlLangGo
	case ".py":
		return hlLangPython
	case ".js", ".ts", ".jsx", ".tsx", ".mjs", ".cjs":
		return hlLangJS
	case ".c", ".h", ".cc", ".cpp", ".cxx", ".hh", ".hpp", ".hxx":
		return hlLangC
	case ".rs":
		return hlLangRust
	case ".sh", ".bash", ".zsh":
		return hlLangShell
	case ".json", ".jsonc":
		return hlLangJSON
	case ".sql":
		return hlLangSQL
	case ".html", ".htm", ".xhtml", ".xml", ".svg", ".vue":
		return hlLangHTML
	case ".css", ".scss", ".sass", ".less":
		return hlLangCSS
	case ".md", ".markdown", ".mdown":
		return hlLangMD
	case ".yml", ".yaml":
		return hlLangYAML
	case ".toml", ".ini", ".cfg", ".conf":
		return hlLangTOML
	default:
		return nil
	}
}
