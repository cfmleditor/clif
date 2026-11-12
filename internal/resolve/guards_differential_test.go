package resolve

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/testutil"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestTheGuardsAgreeWithTheTokenWalk is the differential that moved the
// guards and the request write counts onto the parse: for every file under
// CLIF_GUARDS_CORPUS (a list of directories, as PATH is) and testdata/, and
// every call the file makes, each guard's answer from the token walk it
// replaced (tokenExistsGuards, tokenInstanceGuards, tokenMethodExistence)
// against its answer from the parse's calls, read from an ExtractCalls parse
// and from the plain parse; and the file's request write counts from the
// text search (tokenRequestWriteCounts) against the parse's. Every
// difference is written as a JSON line to CLIF_GUARDS_OUT; the run fails on
// none, since what is left are corrections classified there, except a
// difference between the two parses, which must answer alike. Skipped
// unless CLIF_GUARDS_OUT is set.
func TestTheGuardsAgreeWithTheTokenWalk(t *testing.T) {
	out := os.Getenv("CLIF_GUARDS_OUT")
	if out == "" {
		t.Skip("CLIF_GUARDS_OUT not set")
	}

	dirs := []string{testutil.Testdata()}
	if c := os.Getenv("CLIF_GUARDS_CORPUS"); c != "" {
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
		mu    sync.Mutex
		enc   = json.NewEncoder(f)
		total = map[string]int{}
		asked = map[string]int{}
	)

	work := make(chan string)

	var wg sync.WaitGroup

	for range runtime.NumCPU() {
		wg.Go(func() {
			r := &Resolver{FS: vfs.OS{}}

			for p := range work {
				data, err := os.ReadFile(p)
				if err != nil {
					continue
				}

				diffs, n := guardDifferences(r, p, lex.NormalizeLineBreaks(string(data)))

				mu.Lock()
				for check, k := range n {
					asked[check] += k
				}

				for _, d := range diffs {
					total[d.Check]++
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

	for _, check := range slices.Sorted(maps.Keys(asked)) {
		t.Logf("%-14s asked %9d, differ %6d", check, asked[check], total[check])

		if strings.Contains(check, "-plain") && total[check] > 0 {
			t.Errorf("%s: the plain parse and the ExtractCalls parse answer differently %d times", check, total[check])
		}
	}
}

// guardDifferences asks every guard's question of content for every call it
// makes, and its request write counts, and answers where the token walk and
// the parse differ, with how many questions each check asked.
func guardDifferences(r *Resolver, path, content string) ([]writeDifference, map[string]int) {
	uri := cfpath.ToURI(path)
	full := parser.ParseWithOptions(uri, content, &parser.ParseOptions{ExtractCalls: true})
	plain := plainParseText(path, content)
	asked := map[string]int{}

	var out []writeDifference

	differ := func(check, name, before, after string, line int) {
		asked[check]++

		if after != "false" && after != "[]" && !strings.HasPrefix(check, "request") {
			asked[check+" held"]++
		}

		if before != after {
			at, _ := lineOffset(content, line)
			out = append(out, writeDifference{File: path, Check: check, Name: fmt.Sprint(name, "@", line+1), Old: before, New: after, Context: lineAround(content, at)})
		}
	}

	type question struct {
		variable, funcName string
		line               int
	}

	seen := map[question]bool{}

	calls := full.AllCalls()
	for i := range calls {
		call := &calls[i]

		q := question{call.Variable, call.FuncName, int(call.Line)}
		if seen[q] {
			continue
		}

		seen[q] = true

		if q.variable != "" {
			oldE := strconv.FormatBool(tokenGuardedByExistsCheck(full, q.variable, q.funcName, q.line))
			newE := strconv.FormatBool(r.guardedByExistsCheck(full, q.variable, q.funcName, q.line))
			plainE := strconv.FormatBool(r.guardedByExistsCheck(plain, q.variable, q.funcName, q.line))
			differ("exists", q.variable+"."+q.funcName, oldE, newE, q.line)
			differ("exists-plain", q.variable+"."+q.funcName, newE, plainE, q.line)

			oldI := fmt.Sprint(tokenGuardedInstanceOf(full, q.variable, q.funcName, q.line))
			newI := fmt.Sprint(r.guardedInstanceOf(full, q.variable, q.funcName, q.line))
			plainI := fmt.Sprint(r.guardedInstanceOf(plain, q.variable, q.funcName, q.line))
			differ("instance", q.variable+"."+q.funcName, oldI, newI, q.line)
			differ("instance-plain", q.variable+"."+q.funcName, newI, plainI, q.line)
		}

		site := &parser.CallSite{Line: call.Line}
		oldM := strconv.FormatBool(tokenMethodExistence(full, site, q.funcName))
		newM := strconv.FormatBool(r.methodExistenceGuarded(full, site, q.funcName))
		plainM := strconv.FormatBool(r.methodExistenceGuarded(plain, site, q.funcName))
		differ("registry", q.variable+"."+q.funcName, oldM, newM, q.line)
		differ("registry-plain", q.variable+"."+q.funcName, newM, plainM, q.line)
	}

	oldR := tokenRequestWriteCounts(withoutComments(content))
	newR := requestWritesIn(path, content)
	plainR := requestWriteCounts(plain.Assignments(), parsedCalls(plain, "structInsert"))

	for _, key := range slices.Sorted(maps.Keys(mergeKeys(oldR, newR))) {
		line := 0
		if i := parser.IndexFold(content, key); i >= 0 {
			line = strings.Count(content[:i], "\n")
		}

		differ("request", "request."+key, strconv.Itoa(oldR[key]), strconv.Itoa(newR[key]), line)
		differ("request-plain", "request."+key, strconv.Itoa(newR[key]), strconv.Itoa(plainR[key]), line)
	}

	return out, asked
}

func mergeKeys(a, b map[string]int) map[string]int {
	out := maps.Clone(a)
	for k, v := range b {
		out[k] += v
	}

	return out
}

// The token walk the guards and the request counts replaced, kept as this
// test's oracle.

func tokenCodeOn(pr *parser.ParseResult, from, to int) [][]parser.Token {
	tokens := fileTokens(pr)
	lo := sort.Search(len(tokens), func(i int) bool { return int(tokens[i].Line) >= from })
	hi := sort.Search(len(tokens), func(i int) bool { return int(tokens[i].Line) > to })

	if lo >= hi {
		return nil
	}

	return codeLists(tokens[lo:hi])
}

func tokenCodeBetween(pr *parser.ParseResult, from, to int) [][]parser.Token {
	tokens := fileTokens(pr)
	lo := sort.Search(len(tokens), func(i int) bool { return tokens[i].Offset >= from })
	hi := sort.Search(len(tokens), func(i int) bool { return tokens[i].Offset >= to })

	if lo >= hi {
		return nil
	}

	return codeLists(tokens[lo:hi])
}

func tokenGuardedByExistsCheck(pr *parser.ParseResult, variable, funcName string, line int) bool {
	if funcName == "" {
		return false
	}

	text, start, ok := guardText(pr, variable, funcName, line)
	if !ok || !mentions(text, funcName) {
		return false
	}

	end := start + len(text)

	for _, m := range tokenExistsGuards(tokenCodeBetween(pr, start, end), variable, funcName) {
		if m[1] <= end && guards(text, m[0]-start, m[1]-start) {
			return true
		}
	}

	return false
}

func tokenGuardedInstanceOf(pr *parser.ParseResult, variable, funcName string, line int) []string {
	text, start, ok := guardText(pr, variable, funcName, line)
	if !ok || !mentions(text, "InstanceOf(") {
		return nil
	}

	var comps []string

	end := start + len(text)

	for _, g := range tokenInstanceGuards(tokenCodeBetween(pr, start, end), variable) {
		if g.end <= end && guards(text, g.start-start, g.end-start) {
			comps = append(comps, g.comp)
		}
	}

	return comps
}

func tokenExistsGuards(lists [][]parser.Token, variable, funcName string) [][2]int {
	var out [][2]int

	for _, c := range codeCalls(lists, "structKeyExists", "keyExists", "isDefined") {
		pieces := c.pieces()

		switch strings.ToLower(c.name().Value) {
		case "structkeyexists":
			if len(pieces) == 2 && spells(pieces[0], variable) && soleStringIs(pieces[1], funcName) {
				out = append(out, [2]int{c.name().Offset, c.end()})
			}
		case "keyexists":
			if c.before(1).Kind != parser.TokDot || len(pieces) != 1 || !soleStringIs(pieces[0], funcName) {
				continue
			}

			if start, ok := spelledBefore(c.tokens, c.at-1, variable); ok {
				out = append(out, [2]int{c.tokens[start].Offset, c.end()})
			}
		default:
			if len(pieces) == 1 && soleStringIs(pieces[0], variable+"."+funcName) {
				out = append(out, [2]int{c.name().Offset, c.end()})
			}
		}
	}

	return out
}

func tokenInstanceGuards(lists [][]parser.Token, variable string) []instanceGuard {
	var out []instanceGuard

	for _, c := range codeCalls(lists, "isInstanceOf", "checkForInstanceOf") {
		pieces := c.pieces()
		if len(pieces) != 2 || !spells(pieces[0], variable) {
			continue
		}

		if comp, ok := soleString(pieces[1]); ok && comp != "" && strings.IndexFunc(comp, func(r rune) bool { return r != '.' && !isWordRune(r) }) < 0 {
			out = append(out, instanceGuard{start: c.name().Offset, end: c.end(), comp: comp})
		}
	}

	return out
}

func tokenMethodExistence(pr *parser.ParseResult, call *parser.CallSite, funcName string) bool {
	line := int(call.Line)
	if !mentions(sourceText(pr.Content, line-3, line+4), funcName) {
		return false
	}

	for _, c := range codeCalls(tokenCodeOn(pr, line-3, line+3), "isNull", "structKeyExists") {
		if isWord(c.name(), "isNull") {
			args := c.args()
			if len(args) > 0 && isWord(args[len(args)-1], funcName) &&
				!slices.ContainsFunc(args, func(t parser.Token) bool { return t.Kind == parser.TokLParen }) {
				return true
			}

			continue
		}

		pieces := c.pieces()
		if len(pieces) != 2 || c.before(1).Kind == parser.TokBang || isWord(c.before(1), "not") {
			continue
		}

		if lit, ok := soleString(pieces[1]); ok && strings.EqualFold(strings.TrimSpace(lit), funcName) {
			return true
		}
	}

	return false
}

func tokenRequestWriteCounts(content string) map[string]int {
	counts := map[string]int{}
	add := func(key string) {
		if key != "" {
			counts[strings.ToLower(key)]++
		}
	}

	for at := 0; ; {
		i := parser.IndexFoldFrom(content, "request", at)
		if i < 0 {
			break
		}

		at = i + len("request")

		add(tokenRequestKeyWrittenAt(content, i, at))
	}

	for at := 0; ; {
		i := parser.IndexFoldFrom(content, "structinsert", at)
		if i < 0 {
			break
		}

		at = i + len("structinsert")

		j := tokenSkipSpace(content, at)
		if j >= len(content) || content[j] != '(' {
			continue
		}

		j = tokenSkipSpace(content, j+1)
		if !parser.HasPrefixFold(content[j:], "request") {
			continue
		}

		j = tokenSkipSpace(content, j+len("request"))
		if j >= len(content) || content[j] != ',' {
			continue
		}

		if key, _, ok := quotedAt(content, tokenSkipSpace(content, j+1)); ok {
			add(key)
		}
	}

	for at := 0; ; {
		i := parser.IndexFoldFrom(content, "name", at)
		if i < 0 {
			break
		}

		at = i + len("name")

		j := tokenSkipSpace(content, at)
		if j >= len(content) || content[j] != '=' {
			continue
		}

		value, _, ok := quotedAt(content, tokenSkipSpace(content, j+1))
		if ok && parser.HasPrefixFold(value, "request.") {
			add(value[len("request."):])
		}
	}

	return counts
}

func tokenRequestKeyWrittenAt(content string, start, end int) string {
	if start > 0 && lex.IsWord(content[start-1]) {
		return ""
	}

	j := tokenSkipSpace(content, end)
	if j >= len(content) {
		return ""
	}

	var key string

	switch content[j] {
	case '.':
		j = tokenSkipSpace(content, j+1)
		k := j

		for k < len(content) && lex.IsWord(content[k]) {
			k++
		}

		key, j = content[j:k], k
	case '[':
		var ok bool
		if key, j, ok = quotedAt(content, tokenSkipSpace(content, j+1)); !ok {
			return ""
		}

		j = tokenSkipSpace(content, j)
		if j >= len(content) || content[j] != ']' {
			return ""
		}

		j++
	default:
		return ""
	}

	j = tokenSkipSpace(content, j)
	if j+1 >= len(content) || content[j] != '=' || content[j+1] == '=' {
		return ""
	}

	return key
}

func tokenSkipSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == '\f' || s[i] == '\v') {
		i++
	}

	return i
}
