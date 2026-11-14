// Package knownissues reads a file of documented findings — known issues, TODOs,
// a baseline of unresolved calls — and turns each into an editor diagnostic.
//
// The format is one finding per line:
//
//	path:line: message
//	path:line:col: message
//
// A message may start with a tag naming its code, and its severity ahead of
// that, for a finding from a tool with rule IDs:
//
//	path:line:col: [warning CFQUERYPARAM_REQ] <cfquery> should use <cfqueryparam/>
//	path:line: [TODO] move this to the service
//
// A tagged entry's diagnostic carries that code, where an untagged one's is
// known-issue, and its severity overrides the file's.
//
// Blank lines and lines starting with # are ignored. A path is relative to the
// directory holding the file, and never absolute: an absolute path is ignored,
// since the file is committed to a project and read on other machines, where it
// would not name the same file. A "../" path is kept — it reaches a sibling
// project the way a .clif.json's workspacePaths do, and holds wherever
// the projects are checked out side by side. It is the format
// `clif unresolved --known-issues` writes, and it is short enough to
// write by hand.
package knownissues

import (
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
)

// Entry is one finding from a known-issues file.
type Entry struct {
	Path     string // the file's path joined to the known-issues file's directory
	Line     int    // 0-based
	Col      int    // 0-based, -1 when the entry names none
	Message  string
	Code     string // from the message's [tag], empty when it has none
	Severity string // from the message's [severity CODE] tag, empty when it names none
}

// entryLine reads path:line[:col]: message as the expression
//
//	^((?:[A-Za-z]:)?[^:]+):(\d+)(?::(\d+))?:\s*(.+)$
//
// read it, groups one to four. A drive letter is matched only so that an
// absolute Windows path is recognised, and refused, rather than misread as a
// path named "C". The drive is tried first, so `a:12:5: x` is the path `a:12`
// on line 5; and a column is given up when nothing follows it, so `p.cfc:12:34`
// is the message `34`.
func entryLine(s string) (path, line, col, msg string, ok bool) {
	// The path is a run of bytes that are not a colon, a letter and a colon
	// before it when the text begins with one.
	if len(s) >= 2 && lex.IsASCIILetter(s[0]) && s[1] == ':' {
		if end := strings.IndexByte(s[2:], ':'); end > 0 {
			if line, col, msg, ok = entryNumbers(s[2+end+1:]); ok {
				return s[:2+end], line, col, msg, true
			}
		}
	}

	end := strings.IndexByte(s, ':')
	if end <= 0 {
		return "", "", "", "", false
	}

	if line, col, msg, ok = entryNumbers(s[end+1:]); ok {
		return s[:end], line, col, msg, true
	}

	return "", "", "", "", false
}

// entryNumbers reads `LINE[:COL]:\s*MESSAGE` after a path and its colon.
func entryNumbers(s string) (line, col, msg string, ok bool) {
	digits := entryDigits(s)
	if digits == 0 || digits >= len(s) || s[digits] != ':' {
		return "", "", "", false
	}

	line, rest := s[:digits], s[digits+1:]

	// A column, when a colon and a message follow it.
	if n := entryDigits(rest); n > 0 && n < len(rest) && rest[n] == ':' {
		if msg, ok = entryMessage(rest[n+1:]); ok {
			return line, rest[:n], msg, true
		}
	}

	if msg, ok = entryMessage(rest); ok {
		return line, "", msg, true
	}

	return "", "", "", false
}

// entryMessage reads `\s*(.+)$`: the text after any leading whitespace, which
// holds no newline and is not empty. When only whitespace follows, the last
// character of it is the message, as the expression backtracked to.
func entryMessage(s string) (string, bool) {
	start := lex.SkipRegexpSpace(s, 0)
	if start == len(s) {
		if start == 0 || s[len(s)-1] == '\n' {
			return "", false
		}

		return s[len(s)-1:], true
	}

	if strings.IndexByte(s[start:], '\n') >= 0 {
		return "", false
	}

	return s[start:], true
}

// entryDigits is the length of the run of ASCII digits at the start of s.
func entryDigits(s string) int {
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}

	return n
}

// entryTagWords are the severity words a tag may lead with.
var entryTagWords = []string{"error", "warning", "warn", "information", "info", "hint"}

// entryTag reads a message's leading [CODE] or [severity CODE] as the
// expression
//
//	^\[(?:(?i:(error|warning|warn|information|info|hint))\s+)?([A-Za-z0-9_.:-]+)\]\s*(.*)$
//
// read it, groups one to three: the severity word is a word of its own, and
// with no whitespace after it the word is the code (`[error]`).
func entryTag(s string) (severity, code, msg string, ok bool) {
	if s == "" || s[0] != '[' {
		return "", "", "", false
	}

	body := s[1:]

	// The severity first: a word, then whitespace, then the code.
	word := 0
	for word < len(body) && !lex.IsRegexpSpace(body[word]) {
		word++
	}

	if word > 0 && word < len(body) {
		if gap := lex.SkipRegexpSpace(body, word); gap > word && slices.ContainsFunc(entryTagWords, func(w string) bool { return strings.EqualFold(body[:word], w) }) {
			if code, msg, ok = entryTagCode(body[gap:]); ok {
				return body[:word], code, msg, true
			}
		}
	}

	if code, msg, ok = entryTagCode(body); ok {
		return "", code, msg, true
	}

	return "", "", "", false
}

// entryTagCode reads `CODE]\s*(.*)$`: a code, its bracket, and the rest, which
// holds no newline once its leading whitespace is gone.
func entryTagCode(s string) (code, msg string, ok bool) {
	n := 0
	for n < len(s) && (lex.IsWord(s[n]) || s[n] == '.' || s[n] == ':' || s[n] == '-') {
		n++
	}

	if n == 0 || n >= len(s) || s[n] != ']' {
		return "", "", false
	}

	rest := s[n+1:]

	rest = rest[lex.SkipRegexpSpace(rest, 0):]
	if strings.IndexByte(rest, '\n') >= 0 {
		return "", "", false
	}

	return s[:n], rest, true
}

// Parse reads the entries in content. baseDir anchors relative paths.
func Parse(content, baseDir string) []Entry {
	var out []Entry

	for raw := range strings.SplitSeq(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		path, lineNo, colNo, message, ok := entryLine(line)
		if !ok {
			continue
		}

		n, err := strconv.Atoi(lineNo)
		if err != nil || n < 1 {
			continue
		}

		col := -1

		if colNo != "" {
			if c, err := strconv.Atoi(colNo); err == nil && c >= 1 {
				col = c - 1
			}
		}

		if isAbsoluteAnywhere(path) {
			continue
		}

		rel := filepath.Clean(filepath.FromSlash(strings.ReplaceAll(path, `\`, "/")))

		e := Entry{Path: filepath.Join(baseDir, rel), Line: n - 1, Col: col, Message: strings.TrimSpace(message)}
		if severity, code, rest, ok := entryTag(e.Message); ok {
			e.Severity, e.Code, e.Message = strings.ToLower(severity), code, rest
		}

		out = append(out, e)
	}

	return out
}

// isAbsoluteAnywhere reports whether p is absolute on any platform — a drive
// letter, a leading slash or backslash — since a file written on one machine is
// read on others.
func isAbsoluteAnywhere(p string) bool {
	if p == "" {
		return false
	}

	if p[0] == '/' || p[0] == '\\' {
		return true
	}

	return len(p) >= 2 && p[1] == ':' && (p[0]|0x20 >= 'a' && p[0]|0x20 <= 'z')
}

// Severity maps a config severity name to the LSP value. Unknown or empty is
// warning, the least severe level every editor's problems panel lists: Zed's
// lists only errors and warnings, and VS Code's leaves out hints. Information
// was the default, and its entries showed inline in Zed but not in the panel.
func Severity(name string) protocol.DiagnosticSeverity {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "error":
		return protocol.DiagnosticSeverityError
	case "information", "info":
		return protocol.DiagnosticSeverityInformation
	case "hint":
		return protocol.DiagnosticSeverityHint
	default:
		return protocol.DiagnosticSeverityWarning
	}
}

// SeverityName is the tag word for an LSP severity, the inverse of Severity.
func SeverityName(sev protocol.DiagnosticSeverity) string {
	switch sev {
	case protocol.DiagnosticSeverityError:
		return "error"
	case protocol.DiagnosticSeverityWarning:
		return "warning"
	case protocol.DiagnosticSeverityHint:
		return "hint"
	case protocol.DiagnosticSeverityInformation:
		return "information"
	default:
		return "information"
	}
}

// Row is one entry to write: 1-based line, and column (0 for none).
type Row struct {
	Path     string // relative, with forward slashes
	Line     int
	Col      int
	Severity string
	Code     string
	Message  string
}

// Write writes rows under header, each header line as a # comment, sorted by
// path, line and column. A row's severity and code become its [tag].
func Write(w io.Writer, header []string, rows []Row) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}

		if a.Line != b.Line {
			return a.Line < b.Line
		}

		if a.Col != b.Col {
			return a.Col < b.Col
		}

		if a.Code != b.Code {
			return a.Code < b.Code
		}

		return a.Message < b.Message
	})

	for _, h := range header {
		_, _ = fmt.Fprintf(w, "# %s\n", h)
	}

	for i := range rows {
		r := &rows[i]

		pos := strconv.Itoa(r.Line)
		if r.Col > 0 {
			pos += ":" + strconv.Itoa(r.Col)
		}

		tag := strings.TrimSpace(r.Severity + " " + r.Code)
		if r.Code == "" {
			tag = ""
		}

		if tag != "" {
			tag = "[" + tag + "] "
		}

		// A message is one line in this format.
		msg := strings.Join(strings.Fields(r.Message), " ")

		_, _ = fmt.Fprintf(w, "%s:%s: %s%s\n", r.Path, pos, tag, msg)
	}
}

// DeepestTarget is the one of targets, generated report files, whose
// directory holds file: the deepest when directories nest, empty when none
// does. It is how a report split across several files decides where a finding
// goes.
func DeepestTarget(file string, targets []string) string {
	best, depth := "", -1

	for _, t := range targets {
		dir := filepath.Dir(t)

		rel, err := filepath.Rel(dir, file)
		if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}

		if len(dir) > depth {
			best, depth = t, len(dir)
		}
	}

	return best
}

// reanchorWindow is how far from its recorded line an entry's call is looked
// for when the line no longer holds it: an edit above a finding moves it, and
// a baseline is only as fresh as its last regeneration.
const reanchorWindow = 25

// Diagnostic builds the diagnostic for e over the file's lines (nil when the
// file cannot be read, which leaves the entry on its recorded line).
//
// An entry in the unresolved format, `svc.method (reason)`, is underlined at
// the method on its line, and found again within reanchorWindow lines when an
// edit has moved it. Anything else covers the line's text.
func Diagnostic(e *Entry, lines []string, severity protocol.DiagnosticSeverity, source string) protocol.Diagnostic {
	// start and end are byte offsets into text until the end, where they
	// become LSP characters, which count UTF-16 units. Sent as bytes, the
	// underline stopped short of the end of a line holding an é or an emoji,
	// and landed early on a method written after one.
	line, start, end, text := e.Line, 0, 0, ""

	if e.Line < len(lines) {
		text = lines[e.Line]
		end = len(text)
		start = len(text) - len(strings.TrimLeft(text, " \t"))
	}

	if name := calledName(e.Message); name != "" && e.Col < 0 {
		if l, c, ok := findNear(lines, e.Line, name); ok {
			line, start, end, text = l, c, c+len(name), lines[l]
		}
	}

	start, end = lex.UTF16Len(text[:start]), lex.UTF16Len(text[:end])

	// An explicit column is already an LSP one: the CFLint report writes the
	// column CFLint gave, and CFLint counts UTF-16 units.
	if e.Col >= 0 {
		start = e.Col
		if end < start {
			end = start
		}
	}

	code := "known-issue"
	if e.Code != "" {
		code = e.Code
	}

	if e.Severity != "" {
		severity = Severity(e.Severity)
	}

	return protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: conv.Uint32(line), Character: conv.Uint32(start)},
			End:   protocol.Position{Line: conv.Uint32(line), Character: conv.Uint32(end)},
		},
		Severity: severity,
		Source:   protocol.NewOptional(source),
		Code:     protocol.String(code),
		Message:  protocol.String(e.Message),
	}
}

// calledName is the method an unresolved-format message names: the last
// segment of the text before " (", when that text is a call expression.
func calledName(msg string) string {
	head, _, ok := strings.Cut(msg, " (")
	if !ok || head == "" || strings.ContainsAny(head, " \t") {
		return ""
	}

	if dot := strings.LastIndexByte(head, '.'); dot >= 0 {
		head = head[dot+1:]
	}

	head = strings.TrimSuffix(head, "[]")
	for _, c := range head {
		if c != '_' && c != '$' && (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return ""
		}
	}

	return head
}

// findNear finds name as a whole word nearest to line, within reanchorWindow.
func findNear(lines []string, line int, name string) (foundLine, col int, found bool) {
	for d := 0; d <= reanchorWindow; d++ {
		for _, l := range []int{line - d, line + d} {
			if l < 0 || l >= len(lines) || (d > 0 && l == line) {
				continue
			}

			if c := wordIndexFold(lines[l], name); c >= 0 {
				return l, c, true
			}

			if d == 0 {
				break
			}
		}
	}

	return 0, 0, false
}

func wordIndexFold(s, word string) int {
	ls, lw := strings.ToLower(s), strings.ToLower(word)

	for from := 0; ; {
		i := strings.Index(ls[from:], lw)
		if i < 0 {
			return -1
		}

		i += from
		before := i == 0 || !lex.IsIdent(ls[i-1])
		after := i+len(lw) >= len(ls) || !lex.IsIdent(ls[i+len(lw)])

		if before && after {
			return i
		}

		from = i + 1
	}
}
