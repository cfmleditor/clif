package resolve

import (
	"encoding/json"
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

// TestTheWriteSitesAgreeWithTheTokenWalk sorts the differential's
// differences by where they are: each write the token walk finds and the
// parse does not, or the reverse, and each write both find whose value
// differs, with the kind of text it stands in. Written to
// CLIF_WRITES_SITES as JSON lines; skipped when that is not set.
func TestTheWriteSitesAgreeWithTheTokenWalk(t *testing.T) {
	out := os.Getenv("CLIF_WRITES_SITES")
	if out == "" {
		t.Skip("CLIF_WRITES_SITES not set")
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
		mu     sync.Mutex
		enc    = json.NewEncoder(f)
		counts = map[string]int{}
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

				sites := siteDifferences(p, lex.NormalizeLineBreaks(string(data)))

				mu.Lock()
				for _, s := range sites {
					counts[s.Kind+" / "+s.Where]++
					_ = enc.Encode(s)
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

	for _, k := range slices.Sorted(func(yield func(string) bool) {
		for k := range counts {
			if !yield(k) {
				return
			}
		}
	}) {
		t.Logf("%-60s %7d", k, counts[k])
	}
}

// siteDifference is one write site the two readings disagree on.
type siteDifference struct {
	File  string `json:"file"`
	Kind  string `json:"kind"`
	Where string `json:"where"`
	Line  int    `json:"line"`
	Old   string `json:"old"`
	New   string `json:"new"`
	Text  string `json:"text"`
}

func siteDifferences(path, content string) []siteDifference {
	code := textCode(content)
	records := parser.AssignmentsOf(path, content)
	pr := parser.Parse(cfpath.ToURI(path), content)
	where := siteClassifier(pr, content)

	// The token walk's writes, by the offset of the name written: every
	// written name asked whole, as the broadest rule takes it.
	old := map[int]codeWrite{}

	for _, name := range askedNames(code, records) {
		for _, rule := range []writeRule{{qualified: true, compound: true}, {qualified: true, compound: true, member: true}} {
			for _, w := range codeWrites(code, name, rule) {
				if w.start > 0 && w.tokens[w.start-1].Kind == parser.TokDot {
					continue
				}

				if _, ok := old[w.at()]; !ok {
					old[w.at()] = w
				}
			}
		}
	}

	recorded := map[int]*parser.Assignment{}

	for i := range records {
		a := &records[i]
		if a.Op == parser.AssignSet || a.Op == parser.AssignCompound || a.Op == parser.AssignStep {
			recorded[a.At] = a
		}
	}

	var out []siteDifference

	line := func(at int) int { return strings.Count(content[:at], "\n") }

	for at, w := range old {
		a, ok := recorded[at]
		if !ok {
			out = append(out, siteDifference{File: path, Kind: "token only", Where: where(at) + bracketOf(w), Line: line(at), Old: tokensText(content, w.value()), Text: lineAround(content, at)})

			continue
		}

		if ov, nv := tokensText(content, w.value()), a.ValueText(content); ov != nv && a.Op != parser.AssignStep {
			out = append(out, siteDifference{File: path, Kind: "value", Where: valueClass(ov, nv), Line: line(at), Old: ov, New: nv, Text: lineAround(content, at)})
		}

		if w.startsStatement() != a.StartsStatement() {
			out = append(out, siteDifference{File: path, Kind: "statement start", Where: where(at), Line: line(at), Old: boolText(w.startsStatement()), New: boolText(a.StartsStatement()), Text: lineAround(content, at)})
		}

		if w.line() != a.Line {
			out = append(out, siteDifference{File: path, Kind: "line", Where: where(at), Line: line(at), Old: strconv.Itoa(w.line()), New: strconv.Itoa(a.Line), Text: lineAround(content, at)})
		}
	}

	for at, a := range recorded {
		if _, ok := old[at]; !ok {
			out = append(out, siteDifference{File: path, Kind: "parse only", Where: where(at), Line: line(at), New: a.Target + " " + a.Operator + " " + a.ValueText(content), Text: lineAround(content, at)})
		}
	}

	return out
}

// bracketOf names the innermost bracket the token walk's write stands in,
// as the parse reads brackets: "" for none or a block.
func bracketOf(w codeWrite) string {
	var stack []int

	for i := range w.start {
		switch w.tokens[i].Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			stack = append(stack, i)
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		default:
		}
	}

	if len(stack) == 0 {
		return ""
	}

	open := stack[len(stack)-1]
	before := parser.Token{Kind: parser.TokEOF}

	if open > 0 {
		before = w.tokens[open-1]
	}

	switch w.tokens[open].Kind {
	case parser.TokLBracket:
		return " in [ ]"
	case parser.TokLBrace:
		switch before.Kind {
		case parser.TokEquals, parser.TokLParen, parser.TokLBracket, parser.TokComma, parser.TokColon, parser.TokQuestion:
			return " in a struct literal"
		default:
			if isWord(before, "return") {
				return " in a struct literal"
			}

			return ""
		}
	default:
		switch {
		case isWord(before, "for"):
			return ""
		case isWord(before, "if") || isWord(before, "while"):
			return " in a condition"
		case before.Kind == parser.TokIdent || before.Kind == parser.TokRParen || before.Kind == parser.TokRBracket:
			return " in a call's arguments"
		default:
			return " in ( )"
		}
	}
}

func boolText(b bool) string {
	if b {
		return "true"
	}

	return "false"
}

// valueClass sorts a value difference.
func valueClass(before, after string) string {
	switch {
	case strings.HasPrefix(after, before) && strings.ContainsAny(after[len(before):], "\n"):
		return "parse runs on over a line break"
	case strings.HasPrefix(before, after) && strings.HasPrefix(strings.TrimSpace(before[len(after):]), ","):
		return "parse ends at a comma"
	case strings.HasPrefix(before, after):
		return "parse ends earlier"
	case strings.HasPrefix(after, before):
		return "parse ends later"
	default:
		return "other"
	}
}

// siteClassifier says what kind of text an offset of content stands in, as
// the parse splits it.
func siteClassifier(pr *parser.ParseResult, content string) func(int) string {
	tags := parser.CFTags(content)

	return func(at int) string {
		for i := range pr.Regions {
			r := &pr.Regions[i]
			if at < r.Offset || at >= r.Offset+len(r.Text) {
				continue
			}

			switch r.Kind {
			case parser.RegionScript:
				for _, s := range parser.InterpolatedSpans(content[r.Offset : r.Offset+len(r.Text)]) {
					if at >= r.Offset+s[0] && at < r.Offset+s[1] {
						return "script: #...# in a string"
					}
				}

				return "script"
			case parser.RegionSkip:
				return "skipped <script>"
			default:
			}

			for j := range tags {
				t := &tags[j]
				if at >= t.Start && at < t.End {
					if t.Name == "cfset" {
						return "tag: <cfset>"
					}

					return "tag: <" + t.Name + "> attribute"
				}
			}

			lt := strings.LastIndexByte(content[:at], '<')
			gt := strings.LastIndexByte(content[:at], '>')

			if lt > gt {
				return "tag: markup tag attribute"
			}

			if h := strings.LastIndexByte(content[:at], '#'); h > gt {
				return "tag: #...# in text"
			}

			return "tag: text"
		}

		return "no region"
	}
}
