package resolve

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/uri"
)

// windowDeclaresLocal answers from declarations found once per file what
// textDeclaresLocal answered by scanning the window; for every window of a
// text built from the shapes that decide a declaration, the two must agree.
func TestWindowDeclaresLocalAgreesWithTheScan(t *testing.T) {
	fragments := []string{
		"var svc = new A();", "VAR\tSvc = 1", "myvar svc = 1", "var svcs = 1", "var\n svc = 1",
		"local.svc = 1", "xlocal.svc = 1", "local.svc2 = 1", "LOCAL.SVC.x()", "local.\nsvc",
		`<cfargument name="svc" type="any">`, `<CFARGUMENT required="true" NAME = 'svc'>`,
		`<cfargument type="x"> name="svc"`, `<cfargument typename="svc">`, `<cfargumentx name="svc">`,
		"<cfargument\n type=\"x\"\n name=\"svc\">", `<cfargument name="row" >`, "svc = 1; variables.svc = 2",
		"var é = 1", "var row_2 = 1", "for (var row in q) {", "}", "",
	}
	names := []string{"svc", "SVC", "row", "row_2", "é", "sv", "svc.x", ""}

	rng := rand.New(rand.NewPCG(1, 2))

	for trial := range 200 {
		var b strings.Builder

		lines := 1 + rng.IntN(12)
		for range lines {
			b.WriteString(fragments[rng.IntN(len(fragments))])
			b.WriteString(fragments[rng.IntN(len(fragments))])
			b.WriteByte('\n')
		}

		content := b.String()
		n := strings.Count(content, "\n") + 2

		for from := -1; from < n; from++ {
			for to := from; to <= n; to++ {
				for _, name := range names {
					want := textDeclaresLocal(sourceText(content, from, to), name)
					if got := windowDeclaresLocal(content, from, to, name); got != want {
						t.Fatalf("trial %d: windowDeclaresLocal(%q, %d, %d, %q) = %v, the scan %v", trial, content, from, to, name, got, want)
					}
				}
			}
		}
	}
}

// FunctionInFile answers from the name's bucket what a walk of the file's
// definitions finds: the first of its own, case-insensitively, whichever
// list is shorter.
func TestFunctionInFileAgreesWithTheWalk(t *testing.T) {
	idx := index.New()

	var big strings.Builder

	big.WriteString("component {\n")

	for i := range 50 {
		fmt.Fprintf(&big, "function m%d() {}\n", i)
	}

	big.WriteString("function Dup() {}\nfunction dup() {}\nfunction init() {}\n}\n")

	files := map[uri.URI]string{"file:///w/Big.cfc": big.String()}
	for i := range 60 {
		files[uri.URI(fmt.Sprintf("file:///w/S%d.cfc", i))] = "component { function init() {} function m1() {} }"
	}

	for u, src := range files {
		idx.IndexFile(u, src)
	}

	for u := range files {
		for _, name := range []string{"m1", "M1", "m49", "dup", "DUP", "init", "missing", ""} {
			var want *parser.FunctionDef

			for _, d := range idx.FunctionsForFile(u) {
				if strings.EqualFold(d.Name, name) {
					want = d

					break
				}
			}

			if got := idx.FunctionInFile(u, name); got != want {
				t.Errorf("FunctionInFile(%s, %q) = %v, the walk %v", u, name, got, want)
			}
		}
	}
}

// callSiteArgumentWalk is callSiteArgument as it was, walking every token.
func callSiteArgumentWalk(tokens []parser.Token, call *parser.CallSite, pos int, name string) ([]parser.Token, bool) {
	selected := -1

	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokIdent || !strings.EqualFold(tokens[i].Value, call.FuncName) || tokens[i+1].Kind != parser.TokLParen {
			continue
		}

		if i > 0 && strings.EqualFold(tokens[i-1].Value, "function") {
			continue
		}

		if call.HasSpan {
			if tokens[i].Offset == int(call.StartByte) {
				return callArgumentAt(tokens, i, pos, name)
			}

			continue
		}

		if tokens[i].Line != int(call.Line) {
			continue
		}

		if selected >= 0 {
			return nil, false
		}

		selected = i
	}

	if selected >= 0 {
		return callArgumentAt(tokens, selected, pos, name)
	}

	return nil, false
}

// callSiteArgument searches the file's tokens for the call's own; over every
// call in the repository's fixtures, with and without its span, it finds what
// the walk found.
func TestCallSiteArgumentAgreesWithTheWalk(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoTestdata(), "*", "*.cf[cm]"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}

	checked := 0

	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}

		pr := parser.ParseWithOptions(cfpath.ToURI(f), string(src), &parser.ParseOptions{ExtractCalls: true})
		tokens := allTokens(pr.Content)

		for _, c := range pr.AllCalls() {
			for _, span := range []bool{true, false} {
				call := c
				call.HasSpan = call.HasSpan && span

				for pos := range 3 {
					got, gotOK := callSiteArgument(tokens, &call, pos, "")
					want, wantOK := callSiteArgumentWalk(tokens, &call, pos, "")

					if gotOK != wantOK || !slices.Equal(got, want) {
						t.Errorf("%s:%d %s pos %d span %v: %v %v, the walk %v %v", f, call.Line, call.FuncName, pos, call.HasSpan, got, gotOK, want, wantOK)
					}

					checked++
				}
			}
		}
	}

	if checked == 0 {
		t.Fatal("no calls checked")
	}
}

func repoTestdata() string {
	_, file, _, _ := runtime.Caller(0)

	return filepath.Join(filepath.Dir(file), "..", "..", "testdata")
}

// A textCache keeps no more than textCacheBytes of text alive, however many
// slots are free, and still answers what it built.
func TestTextCacheIsBoundedByBytes(t *testing.T) {
	const budget = 1 << 20

	c := &textCache[int]{budget: budget}
	text := strings.Repeat("x", budget/4)

	for i := range 10 {
		v := text + strconv.Itoa(i)
		if got := c.of(v, func(s string) int { return len(s) }); got != len(v) {
			t.Fatalf("of = %d, want %d", got, len(v))
		}
	}

	held := 0

	for i := range c.slots {
		held += len(c.slots[i].content)
	}

	if held > budget || held != c.held {
		t.Errorf("holding %d bytes (counted %d), budget %d", held, c.held, budget)
	}

	// A text over the whole budget is still kept, as the newest.
	huge := strings.Repeat("z", 2*budget)
	builds := 0

	for range 2 {
		c.of(huge, func(s string) int {
			builds++

			return len(s)
		})
	}

	if builds != 1 {
		t.Errorf("a text over the budget was built %d times, want once", builds)
	}
}

// FunctionInFile's bucket path finds a definition whose URI is spelled
// differently from the file's first (case, escaping) as the walk does.
func TestFunctionInFileMatchesTheFileHoweverItsURIIsSpelled(t *testing.T) {
	idx := index.New()
	funcs := []parser.FunctionDef{
		{Name: "a", URI: "file:///w/My%20Svc.cfc"},
		{Name: "b", URI: "file:///w/My%20Svc.cfc"},
		{Name: "target", URI: "file:///W/MY%20SVC.cfc"},
		{Name: "escaped", URI: "file:///w/My Svc.cfc"},
	}
	idx.IndexFileFromResult("file:///w/My%20Svc.cfc", funcs, nil)

	for _, name := range []string{"target", "escaped"} {
		if d := idx.FunctionInFile("file:///w/My%20Svc.cfc", name); d == nil || d.Name != name {
			t.Errorf("FunctionInFile(%q) = %v, want the file's definition", name, d)
		}
	}

	plain := []parser.FunctionDef{
		{Name: "a", URI: "file:///w/Svc.cfc"},
		{Name: "b", URI: "file:///w/Svc.cfc"},
		{Name: "cased", URI: "file:///W/SVC.cfc"},
	}
	idx.IndexFileFromResult("file:///w/Svc.cfc", plain, nil)

	if d := idx.FunctionInFile("file:///w/Svc.cfc", "cased"); d == nil || d.Name != "cased" {
		t.Errorf("FunctionInFile(%q) = %v, want the file's definition", "cased", d)
	}
}

// ShareCallerIndex's restore gives each resolver back the index it had and
// forgets the answers read from the shared one.
func TestShareCallerIndexRestoreForgetsWhatItAnswered(t *testing.T) {
	files := callerIndexFixture(t)
	a := &Resolver{FS: newCountingFS(), Index: index.New(), InferArgsFiles: files}

	restore := ShareCallerIndex([]*Resolver{a})
	a.argCache = map[string]string{"k": "from the shared index"}

	restore()

	if a.callerIdx != nil || a.argCache != nil {
		t.Errorf("after restore: callerIdx %v, argCache %v; want both as before the share", a.callerIdx, a.argCache)
	}
}

// LimitTextCaches hands back the budgets it replaced, so a test or command
// that limits the caches does not change them for whatever runs after it;
// and a batch running meanwhile is held to the batch budget.
func TestLimitTextCachesRestoresAndBatchesLiftIt(t *testing.T) {
	// Not parallel: it changes the process's caches, and puts them back.
	before := lineStarts.budgetNow()

	restore := LimitTextCaches(1 << 10)
	defer restore()

	if lineStarts.budgetNow() != 1<<10 || declaredNames.budgetNow() != 1<<10 {
		t.Fatalf("budgets %d and %d, want the limit", lineStarts.budgetNow(), declaredNames.budgetNow())
	}

	text := strings.Repeat("y\n", 1<<12)
	done := BatchTextCaches()

	if lineStarts.budgetNow() != textCacheBytes {
		t.Errorf("budget %d while a batch runs, want the batch default", lineStarts.budgetNow())
	}

	lineStarts.of(text, lineStartsOf)
	lineStarts.of(text+"other", lineStartsOf)
	done()

	if held := heldBytes(lineStarts); held > 1<<10 {
		t.Errorf("after the batch: holding %d bytes, want the limit's 1KB at most", held)
	}

	restore()

	if lineStarts.budgetNow() != before || declaredNames.budgetNow() != before {
		t.Errorf("after restore: budgets %d and %d, want %d", lineStarts.budgetNow(), declaredNames.budgetNow(), before)
	}
}

func heldBytes[T any](c *textCache[T]) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.held
}
