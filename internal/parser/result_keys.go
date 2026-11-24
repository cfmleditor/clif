package parser

import (
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// fnKey is the key the lazy per-function caches (funcVars, funcRefsMap,
// funcCallsMap, funcLinksMap) and the parsers' inFunc are kept under: the
// function's first and last lines. The zero value is no function, as the
// empty string was when the key was "start:end": in is set for every real
// one, since {0, 0} is a function on line 0.
//
// It was a string, which an edit inside a function paid for on every
// keystroke: each key parsed back into its lines to see whether the edit's
// body held it, and, when the edit added or removed a line, each formatted
// again under its new lines.
type fnKey struct {
	start, end int32
	in         bool
}

// noFn is the key of no function: code outside every function, and the
// closures the parse files there.
var noFn fnKey

// funcKey is the key of the function on lines start through end.
func funcKey(start, end int) fnKey {
	return fnKey{start: conv.Int32(start), end: conv.Int32(end), in: true}
}

// scope is the lines k names.
func (k fnKey) scope() FuncScope {
	return FuncScope{Start: int(k.start), End: int(k.end)}
}

// String is k as "start:end", or empty for no function, as it was written
// when it was a string.
func (k fnKey) String() string {
	if !k.in {
		return ""
	}

	return strconv.Itoa(int(k.start)) + ":" + strconv.Itoa(int(k.end))
}

func atoi(s string) int {
	neg := false

	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}

	n := 0

	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}

	if neg {
		return -n
	}

	return n
}

// lineOffsets converts line numbers to byte offsets: start is where startLine
// begins and end is just past endLine's newline (or the end of the content).
// The line table is built once per content, since every function body's
// parse asks and walking from the top each time made a file with many
// functions quadratic in its length. Comparing the cached content is cheap:
// two strings sharing their bytes compare equal without reading them.
func (pr *ParseResult) lineOffsets(startLine, endLine int) (start, end int) {
	return pr.lineStartsTable().Span(startLine, endLine, len(pr.Content))
}

// lineAt is the 0-based line holding byte offset off of Content, from the
// same table lineOffsets reads.
func (pr *ParseResult) lineAt(off int) int {
	return pr.lineStartsTable().LineOf(off)
}

// LineOf is the 0-based line of Content holding byte offset off, from the
// line table the parse result keeps: a caller asking for many offsets in one
// file does not count from the top for each.
func (pr *ParseResult) LineOf(off int) int {
	return pr.lineAt(off)
}

func (pr *ParseResult) lineStartsTable() LineTable {
	pr.lineStartsMu.Lock()
	defer pr.lineStartsMu.Unlock()

	if pr.lineStarts == nil || pr.lineStartsContent != pr.Content {
		pr.lineStarts = NewLineTable(pr.Content)
		pr.lineStartsContent = pr.Content
	}

	return pr.lineStarts
}

func (pr *ParseResult) logDebug(msg string, keysAndValues ...any) {
	if pr.log != nil {
		pr.log.Debug(msg, keysAndValues...)
	}
}

// IsMemberMethod returns true if the method name is a known CFML member function
// on native types (Array, Struct, Query, String, List).
func IsMemberMethod(name string) bool {
	return isMemberMethod(name)
}

// isMemberMethod returns true if the method name is a known CFML member function
// on native types (Array, Struct, Query, String, List).
func isMemberMethod(name string) bool {
	var buf foldScratch

	switch string(buf.lowerFold(name)) {
	// Array
	case "append", "prepend", "clear", "delete", "deleteat", "each", "every", "filter",
		"find", "findall", "findallnocase", "findnocase", "first", "getat", "indexexists",
		"insertat", "isdefined", "isempty", "last", "len", "map", "max", "median", "merge",
		"mid", "min", "new", "pop", "push", "range", "reduce", "reduceright",
		"removeduplicates", "resize", "reverse", "set", "shift", "slice", "some", "sort",
		"splice", "sum", "swap", "tolist", "tostruct", "unshift", "avg", "contains",
		"containsnocase", "addall", "getduplicates", "compact", "rest",
		// Struct
		"copy", "count", "equals", "findkey", "findvalue", "get", "getmetadata",
		"insert", "iscasesensitive", "isordered", "keyarray", "keyexists", "keylist",
		"keytranslate", "listnew", "setmetadata", "toquerystring", "tosorted", "update",
		"valuearray",
		// Query
		"addcolumn", "addrow", "close", "columnarray", "columncount", "columndata",
		"columnexists", "columnlist", "currentrow", "deletecolumn", "deleterow", "execute",
		"getcell", "getcellbyindex", "getresult", "getrow", "lazy",
		"recordcount", "renamecolumn", "rowbyindex", "rowdata", "rowdatabyindex", "rowswap",
		"setcell", "setrow", "convertforgrid",
		// String/List
		"changedelims", "qualifiedtoarray", "qualify", "itemtrim", "trim",
		"valuecount", "valuecountnocase",
		// Common global/Java methods
		"getbytes", "tostring", "hashcode", "getclass", "init", "tobytearray", "tochararray",
		// java.lang.Class / java.lang.reflect methods
		"getcomponenttype", "getname", "getsimplename", "getdeclaredmethods",
		"getmethods", "getdeclaredfields", "getfields", "newinstance",
		"isarray", "isassignablefrom", "isinstance", "getinterfaces",
		"getsuperclass", "forname",
		// JavaScript String/Array/RegExp methods
		"split", "substr", "substring", "indexof", "lastindexof", "tolowercase",
		"touppercase", "charat", "concat", "search", "test", "exec", "join",
		"replace", "match", "startswith", "endswith", "includes", "padstart",
		"padend", "repeat", "charcodeat", "normalize", "trimstart", "trimend",
		// JavaScript DOM/Event/BOM methods
		"focus", "submit", "add", "remove", "item", "preventdefault",
		"stoppropagation", "closest", "on", "configure", "setattribute",
		"getattribute", "removeattribute", "insertcell", "hasownproperty",
		"setdate", "getdate", "getday", "gettime", "getmonth", "setmonth",
		"getfullyear", "setfullyear", "getdocumentelement", "getelementsbytagname",
		"getelementbyid", "getelementsbyclassname", "queryselector", "queryselectorall",
		// jQuery/JS common
		"is", "has", "call", "apply", "bind", "then", "catch", "finally",
		// Common property-like methods (Java/iText objects)
		"width", "height", "length":
		return true
	}

	return false
}
