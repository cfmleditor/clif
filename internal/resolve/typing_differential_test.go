package resolve

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/testutil"
)

// TestReceiverTypingAgreesWithTheLineReader is the differential that moved
// receiver typing's assignment reads onto the parse's assignments: for every
// file under CLIF_TYPING_CORPUS (a list of directories, as PATH is) and
// testdata/, every name the line reader or the parse finds assigned, and every
// line that name is written on, the question each reader is asked there (the
// last assignment before the line, from its function's start; whether a
// native claim was overwritten or may not reach it) answered by the line
// reader it replaced (typing_line_oracle_test.go) and by the records. Every
// difference is classified (typingClass) and written as a JSON line to
// CLIF_TYPING_OUT; the counts per class are logged. Skipped unless
// CLIF_TYPING_OUT is set.
func TestReceiverTypingAgreesWithTheLineReader(t *testing.T) {
	out := os.Getenv("CLIF_TYPING_OUT")
	if out == "" {
		t.Skip("CLIF_TYPING_OUT not set")
	}

	dirs := []string{testutil.Testdata()}
	if c := os.Getenv("CLIF_TYPING_CORPUS"); c != "" {
		dirs = append(dirs, filepath.SplitList(c)...)
	}

	var files []string

	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil //nolint:nilerr // an unreadable entry is skipped
			}

			switch strings.ToLower(filepath.Ext(p)) {
			case ".cfc", ".cfm", ".cfml":
				files = append(files, p)
			}

			return nil
		})
	}

	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = f.Close() }()

	var (
		mu      sync.Mutex
		enc     = json.NewEncoder(f)
		classes = map[string]int{}
		asked   = map[string]int{}
	)

	work := make(chan string)

	var wg sync.WaitGroup

	for range runtime.NumCPU() {
		wg.Go(func() {
			for p := range work {
				data, err := os.ReadFile(p)
				if err != nil || len(data) > 2<<20 {
					continue
				}

				diffs, n := typingDifferences(p, lex.NormalizeLineBreaks(string(data)))

				mu.Lock()
				for check, k := range n {
					asked[check] += k
				}

				for _, d := range diffs {
					classes[d.Check+" / "+d.Class]++
					_ = enc.Encode(d)
				}
				mu.Unlock()
			}
		})
	}

	for _, p := range files {
		work <- p
	}

	close(work)
	wg.Wait()

	t.Logf("%d files", len(files))

	for _, check := range slices.Sorted(mapKeys(asked)) {
		t.Logf("%-12s asked %9d", check, asked[check])
	}

	for _, class := range slices.Sorted(mapKeys(classes)) {
		t.Logf("%-70s %7d", class, classes[class])
	}
}

func mapKeys[V any](m map[string]V) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// typingDifference is one question the two readers answer differently.
type typingDifference struct {
	File  string `json:"file"`
	Check string `json:"check"`
	Class string `json:"class"`
	Name  string `json:"name"`
	Line  int    `json:"line"`
	Old   string `json:"old"`
	New   string `json:"new"`
	// Site is the line of the assignment one reader found and the other did
	// not, or that both found and read differently.
	Site string `json:"site"`
}

// typingDifferences asks each reader's questions of content.
func typingDifferences(path, content string) ([]typingDifference, map[string]int) {
	pr := parser.ParseWithOptions(cfpath.ToURI(path), content, &parser.ParseOptions{})
	records := pr.Assignments()
	table := scanLineAssignments(content)
	lines := strings.Split(content, "\n")

	names := map[string]bool{}
	for _, a := range table.assignments {
		names[strings.ToLower(a.name)] = true
	}

	for i := range records {
		a := &records[i]
		if a.Op != parser.AssignSet {
			continue
		}

		target := strings.ToLower(a.Target)
		target = strings.TrimPrefix(strings.TrimPrefix(target, "local."), "variables.")

		if target != "" && strings.IndexFunc(target, func(r rune) bool { return !isWordRune(r) }) < 0 {
			names[target] = true
		}
	}

	q := typingAsk{path: path, content: content, pr: pr, records: records, table: table, lines: lines, asked: map[string]int{}}

	uses := wordLines(lines)

	for name := range names {
		prev := -1

		for _, use := range uses[name] {
			start := pr.FuncScopeAt(use).Start

			q.last(name, start, use)

			// nativeReassignedAt from the function's start, and from the
			// line after the name's previous use.
			for _, from := range []int{start, prev + 1} {
				if from >= start && from < use {
					q.reassigned(name, from, use)
				}
			}

			// nativeClaimMayNotReach from the previous use, as the ref.
			if prev >= start {
				q.claim(name, start, prev, use)
			}

			prev = use
		}
	}

	return q.out, q.asked
}

// typingAsk asks one file's questions of both readers.
type typingAsk struct {
	path, content string
	pr            *parser.ParseResult
	records       []parser.Assignment
	table         lineAssignmentTable
	lines         []string
	asked         map[string]int
	out           []typingDifference
}

// answer is one reader's last assignment: its value, its line and whether
// there is one.
type answer struct {
	rhs  string
	line int
	ok   bool
}

func (a answer) String() string { return fmt.Sprint(a.rhs, " @", a.line, " ", a.ok) }

// differ records a difference of check about name at use, classified.
func (q *typingAsk) differ(check, name string, use int, old, now string, o, n answer) *typingDifference {
	d := typingDifference{File: q.path, Check: check, Name: name, Line: use, Old: old, New: now}
	d.Class, d.Site = classifyTyping(q.pr, q.records, q.lines, name, use, o, n)
	q.out = append(q.out, d)

	return &q.out[len(q.out)-1]
}

// last asks localAssignmentAt (the table) and the line walk.
func (q *typingAsk) last(name string, start, use int) {
	var o, w, n answer

	o.rhs, o.line, o.ok = q.table.assignmentAt(q.content, name, start, use)
	w.rhs, w.line, w.ok = localAssignmentAt(q.content, name, start, use)
	n.rhs, n.line, n.ok = lastAssignment(q.records, q.content, name, start, use)

	q.asked["last"]++
	q.asked["walk"]++

	if o != n {
		q.differ("last", name, use, o.String(), n.String(), o, n)
	}

	if w != n {
		q.differ("walk", name, use, w.String(), n.String(), w, n)
	}
}

// reassigned asks nativeReassignedAt.
func (q *typingAsk) reassigned(name string, from, use int) {
	q.asked["reassigned"]++

	o, n := lineNativeReassignedAt(q.content, name, from, use), nativeReassignedAt(q.records, q.content, name, from, use)
	if o == n {
		return
	}

	var oa, na answer

	oa.rhs, oa.line, oa.ok = localAssignmentAt(q.content, name, from, use)
	na.rhs, na.line, na.ok = lastAssignment(q.records, q.content, name, from, use)
	d := q.differ("reassigned", name, use, strconv.FormatBool(o), strconv.FormatBool(n), oa, na)

	if oa.ok && na.ok && oa.line == na.line && !o {
		switch shape, ok := statementShape(contentBeforeLine(q.content, oa.line)); {
		case !ok:
			d.Class = "same site: old gave up past 4,096 tokens above it"
		case shape.parens != 0 || shape.brackets != 0 || shape.expressionKey():
			d.Class = "same site: the text above left a bracket open to the old shape test"
		}
	}
}

// claim asks nativeClaimMayNotReach.
func (q *typingAsk) claim(name string, start, ref, use int) {
	q.asked["claim"]++

	o, n := lineNativeClaimMayNotReach(q.content, name, start, ref, use), nativeClaimMayNotReach(q.records, q.content, name, start, ref, use)
	if o == n {
		return
	}

	var oa, na answer

	oa.rhs, oa.line, oa.ok = localAssignmentAt(q.content, name, start, ref)
	na.rhs, na.line, na.ok = lastAssignment(q.records, q.content, name, start, ref)
	q.differ("claim", name, use, strconv.FormatBool(o), strconv.FormatBool(n), oa, na)
}

// wordLines is, for each word lowercased, the lines it is written on, once
// each and in order.
func wordLines(lines []string) map[string][]int {
	out := map[string][]int{}

	for n, line := range lines {
		for i := 0; i < len(line); {
			if !lex.IsWord(line[i]) {
				i++

				continue
			}

			j := i
			for j < len(line) && lex.IsWord(line[j]) {
				j++
			}

			w := strings.ToLower(line[i:j])
			if l := out[w]; len(l) == 0 || l[len(l)-1] != n {
				out[w] = append(l, n)
			}

			i = j
		}
	}

	return out
}

// classifyTyping names why the line reader's answer o and the records' n to
// the question of name's last assignment before use differ, and the line of
// the assignment that differs.
func classifyTyping(pr *parser.ParseResult, records []parser.Assignment, lines []string, name string, use int, o, n answer) (class, site string) {
	oRHS, oLine, oOK := o.rhs, o.line, o.ok
	nRHS, nLine, nOK := n.rhs, n.line, n.ok

	siteOf := func(l int) string {
		if l < 0 || l >= len(lines) {
			return ""
		}

		return fmt.Sprintf("%d: %s", l, strings.TrimSpace(lines[l]))
	}

	// The same assignment, read differently.
	if oOK && nOK && oLine == nLine {
		return "same site: " + rhsClass(oRHS, nRHS), siteOf(oLine)
	}

	// Which assignment is the later one is the one the other reader skipped:
	// the old reader's when it found one after the records', else the records'.
	oldLater := oOK && (!nOK || oLine > nLine)
	if !oOK && !nOK {
		// Both refused; they differ only in... nothing.
		return "both none", ""
	}

	if oldLater {
		return "old only: " + oldOnlyClass(pr, records, lines, name, oLine, use), siteOf(oLine)
	}

	// The records found one after the old reader's, or where it found none.
	// The old reader may have found the same site and refused it.
	if oLine == nLine && !oOK {
		return "old refused the same site: " + refusedClass(oLine, use), siteOf(nLine)
	}

	return "new only: " + newOnlyClass(lines, name, nLine), siteOf(nLine)
}

// rhsClass names how two readings of the same assignment's value differ.
func rhsClass(o, n string) string {
	squash := func(s string) string {
		return strings.Map(func(r rune) rune {
			if r == ' ' || r == '\t' || r == '\r' || r == '\n' {
				return -1
			}

			return r
		}, s)
	}

	semis := func(s string) string { return strings.ReplaceAll(squash(s), ";", "") }

	switch {
	case squash(o) == squash(n):
		return "spacing only"
	case semis(o) == semis(n):
		return "old dropped the ; ending each line of a joined body"
	case strings.HasPrefix(squash(o), squash(n)) && strings.HasPrefix(squash(o)[len(squash(n)):], ";"):
		return "old ran on into the next statement"
	case strings.HasSuffix(o, ">") && squash(strings.TrimSuffix(o, ">")) == squash(n):
		return "old kept the <cfset>'s >"
	case strings.HasPrefix(o, n) && strings.ContainsAny(o[len(n):], ";,") && strings.Contains(o, "="):
		return "old ran on into the next statement"
	case strings.HasPrefix(squash(o), squash(n)) && (strings.Contains(o[len(n)-min(len(n), len(o)):], "<") || strings.Contains(o, "</")):
		return "old ran on into markup after the tag"
	case strings.Contains(o, "//") || strings.Contains(o, "/*") || strings.Contains(o, "<!---"):
		return "old kept a comment"
	case strings.HasPrefix(squash(n), squash(o)):
		return "new joined lines the old reader did not"
	case strings.HasPrefix(squash(o), squash(n)):
		return "old joined lines past the statement's end"
	default:
		return "other"
	}
}

// oldOnlyClass names why the records have no assignment where the line
// reader found one.
func oldOnlyClass(pr *parser.ParseResult, records []parser.Assignment, lines []string, name string, line, use int) string {
	var found *parser.Assignment

	for i := range records {
		a := &records[i]
		if a.Line != line {
			continue
		}

		t := strings.ToLower(a.Target)
		t = strings.TrimPrefix(strings.TrimPrefix(t, "local."), "variables.")

		if strings.EqualFold(t, name) {
			found = a
		}
	}

	if found != nil {
		switch {
		case found.Op != parser.AssignSet:
			return "the parse reads another operator"
		case !found.StartsStatement() && !found.Local:
			return "an inner write"
		case found.Local && forAt(strings.Join(lines, "\n"), found.Statement):
			return "a for loop's initializer"
		case found.Value[1] > lineStartOffset(lines, use):
			return "the use is inside the assignment's own value (old cut it at a line the statement goes on past)"
		default:
			return "the records refused its value"
		}
	}

	if parser.IndexFold(lines[line], "<cfset") >= 0 && pr.RegionKindFrom(lineStartOffset(lines, line)) == parser.RegionScript {
		return "a Lucee tag island in script (parse gap)"
	}

	at := lineStartOffset(lines, line) + len(lines[line]) - len(strings.TrimLeft(lines[line], " \t"))

	switch pr.RegionKindFrom(at) {
	case parser.RegionScript:
	case parser.RegionSkip:
		return "not CFML (a <script> or skipped block)"
	default:
		if inOpenCFScript(lines, line) {
			return "in a <cfscript> the region split took for comment (parse gap)"
		}

		if !inCFSet(lines[line]) {
			return "markup or text in a tag region"
		}
	}

	trimmed := strings.TrimSpace(lines[line])
	if trimmed == "" {
		return "other"
	}

	if prevCode := previousCodeLine(lines, line); prevCode != "" {
		last := prevCode[len(prevCode)-1]
		if last == '(' || last == ',' || last == '{' || last == '[' || strings.HasPrefix(trimmed, ",") {
			return "a named argument or struct key"
		}
	}

	if insideComment(lines, line) {
		return "in a comment"
	}

	if scriptTagStatement(lines, line) {
		return "an attribute of a property or script-syntax tag"
	}

	if insideOpenGroup(lines, line) {
		return "a named argument or struct key"
	}

	return "other"
}

// scriptTagStatement reports whether line goes on a statement that begins
// `ident ident =`, `ident ident` or `property`: a script-syntax tag or a
// property, whose lines after the first are attributes.
func scriptTagStatement(lines []string, line int) bool {
	at := line

	for i := line - 1; i >= 0 && i > line-60; i-- {
		t := strings.TrimSpace(lines[i])
		if t == "" || strings.HasPrefix(t, "//") || strings.HasPrefix(t, "*") || strings.HasPrefix(t, "/*") {
			continue
		}

		if strings.HasSuffix(t, ";") || strings.HasSuffix(t, "{") || strings.HasSuffix(t, "}") || strings.HasSuffix(t, ">") || strings.HasSuffix(t, "*/") {
			break
		}

		at = i
	}

	first := strings.Fields(strings.TrimSpace(lines[at]))
	if len(first) == 0 {
		return false
	}

	if strings.EqualFold(first[0], "property") || strings.EqualFold(first[0], "param") {
		return true
	}

	return at < line && len(first) > 1 && isWordText(first[0]) && (isWordText(first[1]) || strings.Contains(first[1], "="))
}

// refusedClass names why the line reader refused an assignment the records
// took.
func refusedClass(line, use int) string {
	if use-line > maxAssignmentLines {
		return "old gave up after 24 lines"
	}

	return "old joined past the statement's end"
}

// newOnlyClass names why the line reader had no assignment where the records
// found one.
func newOnlyClass(lines []string, name string, line int) string {
	text := lines[line]

	if n, _, ok := matchSourceAssignment(text); ok && strings.EqualFold(n, name) {
		return "after a line ending in ( or , (the table's rule)"
	}

	trimmed := strings.ToLower(strings.TrimSpace(text))
	trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "<cfset"))
	trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "var "))
	lower := strings.ToLower(name)

	switch {
	case !strings.HasPrefix(strings.TrimPrefix(strings.TrimPrefix(trimmed, "local."), "variables."), lower):
		return "not first on its line"
	default:
		return "first on its line, value or name the matcher refused"
	}
}

func lineStartOffset(lines []string, line int) int {
	at := 0
	for i := range line {
		at += len(lines[i]) + 1
	}

	return at
}

// inOpenCFScript reports whether the text before line opens a <cfscript> it
// does not close, read as text.
func inOpenCFScript(lines []string, line int) bool {
	text := strings.ToLower(strings.Join(lines[:line], "\n"))

	return strings.LastIndex(text, "<cfscript") > strings.LastIndex(text, "</cfscript")
}

// inCFSet reports whether line holds a <cfset.
func inCFSet(line string) bool {
	return parser.IndexFold(line, "<cfset") >= 0
}

// previousCodeLine is the last non-blank line before line, trimmed.
func previousCodeLine(lines []string, line int) string {
	for i := line - 1; i >= 0; i-- {
		if t := strings.TrimSpace(lines[i]); t != "" {
			return t
		}
	}

	return ""
}

// insideComment reports whether line is inside a block comment, roughly.
func insideComment(lines []string, line int) bool {
	t := strings.TrimSpace(lines[line])
	if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "*") || strings.HasPrefix(t, "/*") || strings.HasPrefix(t, "<!---") {
		return true
	}

	open := 0

	for i := max(line-400, 0); i < line; i++ {
		open += strings.Count(lines[i], "/*") - strings.Count(lines[i], "*/")
		open += strings.Count(lines[i], "<!---") - strings.Count(lines[i], "--->")
	}

	return open > 0
}

// insideOpenGroup reports whether the lines from 200 before line leave a
// parenthesis, a bracket or a brace open, roughly.
func insideOpenGroup(lines []string, line int) bool {
	p, _, k := parser.OpenGroups(strings.Join(lines[max(line-200, 0):line], "\n"))

	return p > 0 || k > 0
}
