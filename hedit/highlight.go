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
			for i < n && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			word := string(runes[start:i])
			lookup := word
			if lang.caseInsKw {
				lookup = strings.ToUpper(word)
			}
			kind := tkNormal
			switch {
			case lang.keywords[lookup]:
				kind = tkKeyword
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

		i++ // unrecognized character
	}

	return kinds, state
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
	default:
		return nil
	}
}
