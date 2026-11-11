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
	"github.com/cfmleditor/clif/internal/testutil"
)

// TestTheWriteReadersAgreeWithTheTokenWalk is the differential that moved the
// write readers onto the parse's assignments: for every file under
// CLIF_WRITES_CORPUS (a list of directories, as PATH is) and testdata/, and
// every name the file could be asked about, each reader's answer from the
// token walk it replaced (codeWrites, tokenDeclaresName) against its answer
// from parser.AssignmentsOf. Every difference is written as a JSON line to
// CLIF_WRITES_OUT; the run fails on none, since what is left are corrections
// classified there. Skipped unless CLIF_WRITES_OUT is set.
func TestTheWriteReadersAgreeWithTheTokenWalk(t *testing.T) {
	out := os.Getenv("CLIF_WRITES_OUT")
	if out == "" {
		t.Skip("CLIF_WRITES_OUT not set")
	}

	dirs := []string{testutil.Testdata()}
	if c := os.Getenv("CLIF_WRITES_CORPUS"); c != "" {
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
			for p := range work {
				data, err := os.ReadFile(p)
				if err != nil {
					continue
				}

				diffs, n := writeDifferences(p, lex.NormalizeLineBreaks(string(data)))

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

	for _, check := range slices.Sorted(func(yield func(string) bool) {
		for k := range asked {
			if !yield(k) {
				return
			}
		}
	}) {
		t.Logf("%-10s asked %8d, differ %6d", check, asked[check], total[check])
	}
}

// writeDifference is one question the two readings answer differently.
type writeDifference struct {
	File    string `json:"file"`
	Check   string `json:"check"`
	Name    string `json:"name"`
	Old     string `json:"old"`
	New     string `json:"new"`
	Context string `json:"context"`
}

// writeDifferences asks every reader's question of content for every name
// it might be asked, and answers where the token walk and the records differ,
// with how many questions each check asked.
func writeDifferences(path, content string) ([]writeDifference, map[string]int) {
	code := textCode(content)
	records := parser.AssignmentsOf(path, content)
	names := askedNames(code, records)
	asked := map[string]int{}

	var out []writeDifference

	differ := func(check, name, before, after string, at int) {
		asked[check]++

		if before != after {
			out = append(out, writeDifference{File: path, Check: check, Name: name, Old: before, New: after, Context: lineAround(content, at)})
		}
	}

	for _, name := range names {
		// setsName
		oldSets := len(codeWrites(code, name, writeRule{scopes: []string{"local", "variables"}, compound: true})) > 0
		oldDecl := tokenDeclaresName(code, name)
		newSets := len(writesTo(records, name, writeRule{scopes: []string{"local", "variables"}, compound: true})) > 0
		newDecl := declaresName(records, name)
		differ("sets", name, strconv.FormatBool(oldSets), strconv.FormatBool(newSets), firstAt(code, name))
		differ("declares", name, strconv.FormatBool(oldDecl), strconv.FormatBool(newDecl), firstAt(code, name))

		// fw1Assigns, throughAssigns: qualified writes, by line.
		oldQ, newQ := oldLines(codeWrites(code, name, writeRule{qualified: true})), newLines(writesTo(records, name, writeRule{qualified: true}))
		differ("qualified", name, oldQ, newQ, firstAt(code, name))

		// membersWrittenOn
		if !strings.Contains(name, ".") {
			oldM, newM := oldLines(codeWrites(code, name, writeRule{qualified: true, member: true})), newLines(writesTo(records, name, writeRule{qualified: true, member: true}))
			differ("member", name, oldM, newM, firstAt(code, name))
		}

		// lastMemberWrite
		if container, member, ok := strings.Cut(name, "."); ok && !strings.Contains(member, ".") {
			ov, ol, ook := oldLastMemberWrite(content, code, container, member)
			nv, nl, nok := lastMemberWrite(content, records, container, member)
			differ("last", name, fmt.Sprint(ov, "@", ol, " ", ook), fmt.Sprint(nv, "@", nl, " ", nok), firstAt(code, name))
		}

		// fieldWrites and initStoredArg: variables-scoped writes, with values.
		var ow, nw []string
		for _, w := range codeWrites(code, name, writeRule{scopes: []string{"variables"}}) {
			ow = append(ow, fmt.Sprint(w.line(), "=", tokensText(content, w.value())))
		}

		for _, w := range writesTo(records, name, writeRule{scopes: []string{"variables"}}) {
			nw = append(nw, fmt.Sprint(w.Line, "=", w.ValueText(content)))
		}

		differ("field", name, strings.Join(ow, " | "), strings.Join(nw, " | "), firstAt(code, name))

		// writtenLiteral
		differ("literal", name, oldWrittenLiteral(code, name), writtenLiteral(records, name), firstAt(code, name))
	}

	return out, asked
}

// askedNames are the names a reader could ask content about: every dotted
// name written, by the tokens or the records, and its last one, two and
// three parts, and the name before a member written; every name declared.
func askedNames(code [][]parser.Token, records []parser.Assignment) []string {
	seen := map[string]bool{}
	add := func(parts []string) {
		for k := 1; k <= 3 && k <= len(parts); k++ {
			seen[strings.ToLower(strings.Join(parts[len(parts)-k:], "."))] = true
		}
	}

	for i := range records {
		a := &records[i]
		if a.Op == parser.AssignAttribute || a.HasLiteral {
			seen[strings.ToLower(a.Literal)] = true
		}

		var run []string

		for _, s := range targetSteps(a.Target) {
			if strings.HasPrefix(s, "[") {
				add(run)

				run = nil

				continue
			}

			run = append(run, s)
		}

		add(run)

		if len(run) > 1 {
			add(run[:len(run)-1])
		}
	}

	for _, list := range code {
		tokenNames(list, seen, add)
	}

	delete(seen, "")

	return slices.Sorted(func(yield func(string) bool) {
		for k := range seen {
			if strings.IndexFunc(k, func(r rune) bool { return r != '.' && !isWordRune(r) }) < 0 && !yield(k) {
				return
			}
		}
	})
}

// tokenNames adds to seen the names list's tokens might write or declare.
func tokenNames(list []parser.Token, seen map[string]bool, add func([]string)) {
	for i := range list {
		if list[i].Kind == parser.TokString && i >= 2 && list[i-1].Kind == parser.TokEquals {
			if lit, ok := stringLiteral(list[i]); ok && len(lit) < 64 {
				seen[strings.ToLower(lit)] = true
			}
		}

		if list[i].Kind != parser.TokIdent || i > 0 && list[i-1].Kind == parser.TokDot {
			continue
		}

		parts := []string{list[i].Value}

		j := i + 1
		for j+1 < len(list) && list[j].Kind == parser.TokDot && list[j+1].Kind == parser.TokIdent {
			parts = append(parts, list[j+1].Value)
			j += 2
		}

		if j >= len(list) {
			continue
		}

		switch list[j].Kind {
		case parser.TokEquals, parser.TokPlus, parser.TokMinus, parser.TokStar, parser.TokSlash, parser.TokAmpersand, parser.TokLBracket, parser.TokIdent:
			add(parts)

			if len(parts) > 1 {
				add(parts[:len(parts)-1])
			}
		default:
		}
	}
}

func oldLines(ws []codeWrite) string {
	var out []string
	for _, w := range ws {
		out = append(out, strconv.Itoa(w.line()))
	}

	return strings.Join(out, ",")
}

func newLines(ws []*parser.Assignment) string {
	var out []string
	for _, w := range ws {
		out = append(out, strconv.Itoa(w.Line))
	}

	return strings.Join(out, ",")
}

// oldLastMemberWrite is lastMemberWrite as the token walk answered it.
func oldLastMemberWrite(content string, code [][]parser.Token, container, name string) (string, int, bool) {
	var (
		last  codeWrite
		found bool
	)

	for _, w := range codeWrites(code, container+"."+name, writeRule{scopes: []string{"arguments"}}) {
		if w.startsStatement() {
			last, found = w, true
		}
	}

	if !found {
		return "", 0, false
	}

	value := last.value()
	if len(value) == 0 || endLine(value[len(value)-1]) != value[0].Line {
		return "", 0, false
	}

	return tokensText(content, value), last.line(), true
}

// oldWrittenLiteral is writtenLiteral as the token walk answered it.
func oldWrittenLiteral(code [][]parser.Token, name string) string {
	for _, w := range codeWrites(code, name, writeRule{qualified: true}) {
		if lit, ok := soleString(w.value()); ok && lit != "" && !strings.ContainsAny(lit, "\"'#") {
			return lit
		}
	}

	return ""
}

// firstAt is the offset of the first token spelling name's last part, or 0.
func firstAt(code [][]parser.Token, name string) int {
	last := name[strings.LastIndexByte(name, '.')+1:]

	for _, list := range code {
		for _, t := range list {
			if t.Kind == parser.TokIdent && strings.EqualFold(t.Value, last) {
				return t.Offset
			}
		}
	}

	return 0
}

// lineAround is the line of content holding offset at, trimmed and cut short.
func lineAround(content string, at int) string {
	at = min(max(at, 0), len(content))
	start := strings.LastIndexByte(content[:at], '\n') + 1

	end := strings.IndexByte(content[at:], '\n')
	if end < 0 {
		end = len(content)
	} else {
		end += at
	}

	line := strings.TrimSpace(content[start:end])
	if len(line) > 160 {
		line = line[:160]
	}

	return line
}
