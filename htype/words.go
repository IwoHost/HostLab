package main

import "math/rand"

var commonWords = []string{
	"about", "above", "across", "after", "again", "against", "almost", "along",
	"already", "although", "always", "among", "another", "answer", "around",
	"back", "because", "before", "being", "below", "between", "both", "bring",
	"build", "call", "carry", "cause", "certain", "change", "clear", "close",
	"come", "common", "cover", "dark", "data", "deep", "different", "door",
	"down", "draw", "drive", "early", "earth", "east", "even", "ever", "every",
	"example", "fact", "fall", "family", "fast", "feel", "feet", "field",
	"find", "first", "five", "follow", "food", "force", "form", "found",
	"four", "front", "full", "future", "gave", "give", "good", "great",
	"green", "group", "grow", "hand", "hard", "head", "hear", "heart",
	"help", "here", "high", "hold", "home", "hour", "house", "idea",
	"keep", "kind", "know", "land", "large", "last", "late", "lead",
	"learn", "left", "less", "level", "life", "light", "line", "list",
	"live", "long", "look", "love", "make", "many", "mark", "mind",
	"miss", "more", "most", "move", "much", "must", "name", "near",
	"need", "next", "night", "none", "north", "note", "number", "often",
	"once", "only", "open", "order", "other", "over", "page", "part",
	"pass", "path", "people", "place", "plan", "play", "point", "power",
	"press", "pull", "push", "read", "real", "reason", "rest", "right",
	"rise", "road", "rock", "room", "rule", "same", "save", "send",
	"side", "sign", "since", "size", "slow", "small", "some", "soon",
	"sort", "space", "speak", "stand", "start", "stay", "step", "still",
	"stone", "stop", "story", "such", "take", "talk", "tell", "than",
	"them", "then", "there", "these", "thing", "think", "those", "though",
	"time", "today", "told", "town", "tree", "turn", "type", "under",
	"unit", "upon", "view", "voice", "want", "warm", "watch", "water",
	"wave", "west", "when", "which", "while", "white", "wide", "wind",
	"word", "work", "world", "write", "year", "your",
}

var codeWords = []string{
	// keywords
	"return", "const", "func", "type", "struct", "interface", "import",
	"export", "default", "class", "extends", "async", "await", "defer",
	"range", "switch", "select", "package", "module", "break", "continue",
	// control flow
	"else", "while", "throw", "catch", "finally",
	// types
	"string", "number", "boolean", "error", "null", "void", "true", "false",
	"int", "float", "byte", "bool", "uint", "rune", "slice", "array",
	// common short identifiers
	"result", "value", "index", "count", "size", "length", "data", "item",
	"node", "root", "left", "right", "next", "prev", "head", "tail",
	"key", "name", "text", "line", "col", "row", "path", "body",
	"ctx", "req", "res", "err", "ok", "buf", "src", "dst", "msg", "tmp",
	// functions
	"parse", "format", "render", "handle", "update", "create", "delete",
	"filter", "reduce", "find", "match", "replace", "split", "join",
	"trim", "push", "pop", "append", "remove", "insert", "read", "write",
	"open", "close", "send", "encode", "decode", "validate", "build", "run",
	"fetch", "cache", "flush", "reset", "clone", "merge", "diff", "patch",
	// misc
	"static", "public", "private", "readonly", "override", "abstract",
	"input", "output", "stdin", "stdout", "stderr", "socket", "channel",
	"mutex", "goroutine", "promise", "callback", "closure", "iterator",
}

var quotes = []string{
	"the best time to plant a tree was twenty years ago the second best time is now",
	"in the middle of difficulty lies opportunity",
	"it always seems impossible until it is done",
	"the journey of a thousand miles begins with a single step",
	"be the change you wish to see in the world",
	"whether you think you can or think you cannot you are right",
	"life is what happens when you are busy making other plans",
	"the only way to do great work is to love what you do",
	"imagination is more important than knowledge",
	"an investment in knowledge pays the best interest",
	"do or do not there is no try",
	"stay hungry stay foolish",
	"talk is cheap show me the code",
	"first make it work then make it right then make it fast",
	"simplicity is the ultimate sophistication",
	"any fool can write code that a computer can understand but good programmers write code that humans can understand",
	"programs must be written for people to read and only incidentally for machines to execute",
	"the most dangerous phrase in the language is we have always done it this way",
	"perfection is achieved not when there is nothing more to add but when there is nothing left to take away",
	"before software can be reusable it first has to be usable",
}

func pickN(pool []string, n int) []string {
	if n <= 0 {
		return nil
	}
	result := make([]string, n)
	perm := rand.Perm(len(pool))
	for i := range result {
		result[i] = pool[perm[i%len(pool)]]
	}
	return result
}
