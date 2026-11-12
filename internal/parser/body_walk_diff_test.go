package parser

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"go.lsp.dev/uri"
)

// TestLazyBodyReadsAreTheParses compares, for every function of every file
// under CFML_CORPUS, what each lazy reader answers with what a whole parse
// (ExtractCalls, ScanAllScopes) files for that function:
//
//   - old: the readers as they were (body_walk_oracle_test.go), which walked
//     a body with walkers of their own;
//   - pre: the readers now, on a parse that walked every body;
//   - lazy: the readers now, with every body marked changed, so each is read
//     by the parse's walk of that body alone (ParseOptions.Body).
//
// It prints each difference by class, with counts and examples, and fails
// when pre differs from the parse at all. Run it with CLIF_BODY_DIFF=1:
//
//	CFML_CORPUS=$PWD/testdata:$HOME/corpus CLIF_BODY_DIFF=1 \
//	    go test -count=1 -timeout 60m -run TestLazyBodyReadsAreTheParses ./internal/parser/
func TestLazyBodyReadsAreTheParses(t *testing.T) {
	if os.Getenv("CLIF_BODY_DIFF") == "" {
		t.Skip("set CLIF_BODY_DIFF=1 and CFML_CORPUS")
	}

	files := corpusFiles(t)
	d := &bodyDiff{classes: map[string]*diffClass{}}

	work := make(chan string)

	var wg sync.WaitGroup

	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for f := range work {
				d.file(f)
			}
		})
	}

	for _, f := range files {
		work <- f
	}

	close(work)
	wg.Wait()

	d.report(t, len(files))
}

type diffClass struct {
	n        int
	examples []string
}

type bodyDiff struct {
	mu        sync.Mutex
	classes   map[string]*diffClass
	functions int
}

func (d *bodyDiff) note(class, example string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	c := d.classes[class]
	if c == nil {
		c = &diffClass{}
		d.classes[class] = c
	}

	c.n++

	if len(c.examples) < 4 {
		c.examples = append(c.examples, example)
	}
}

func (d *bodyDiff) report(t *testing.T, files int) {
	t.Helper()

	names := slices.Sorted(func(yield func(string) bool) {
		for k := range d.classes {
			if !yield(k) {
				return
			}
		}
	})

	fmt.Printf("%d files, %d functions\n", files, d.functions)

	for _, k := range names {
		c := d.classes[k]
		fmt.Printf("%7d  %s\n", c.n, k)

		for _, e := range c.examples {
			fmt.Printf("           %s\n", e)
		}

		if strings.HasPrefix(k, "pre ") {
			t.Errorf("a reader on a parse that walked every body differs from the parse: %s", k)
		}
	}
}

func refKey(r *ComponentRef) string { return fmt.Sprintf("%s@%d", strings.ToLower(r.Variable), r.Line) }

func bodyCallKey(c *CallSite) string {
	return fmt.Sprintf("%s.%s@%d", strings.ToLower(c.Variable), strings.ToLower(c.FuncName), c.Line)
}

// file compares the readers on one file.
func (d *bodyDiff) file(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	src := string(data)
	u := uri.File(path)

	full := ParseWithOptions(u, src, &ParseOptions{ExtractCalls: true, ScanAllScopes: true, KeepVars: true})
	old := ParseWithOptions(u, src, &ParseOptions{ScanAllScopes: true})
	pre := ParseWithOptions(u, src, &ParseOptions{ExtractCalls: true, ScanAllScopes: true})
	lazy := ParseWithOptions(u, src, &ParseOptions{ExtractCalls: true, ScanAllScopes: true})
	noCalls := ParseWithOptions(u, src, &ParseOptions{ScanAllScopes: true})

	for _, s := range lazy.Scopes {
		lazy.InvalidateFunc(s.Start, s.End)
	}

	for _, s := range noCalls.Scopes {
		noCalls.InvalidateFunc(s.Start, s.End)
	}

	d.mu.Lock()
	d.functions += len(full.Scopes)
	d.mu.Unlock()

	tag := func(s FuncScope) string {
		if full.RegionKindFrom(mustOffset(full, s.Start)) == RegionScript {
			return "script"
		}

		return "tag"
	}

	for _, s := range full.Scopes {
		at := fmt.Sprintf("%s:%d %s", path, s.Start+1, s.Name)
		kind := tag(s)

		want := slices.Clone(full.funcRefsMap[funcKey(s.Start, s.End)])
		for i := range full.funcRefsMap[noFn] {
			if line := int(full.funcRefsMap[noFn][i].Line); line >= s.Start && line <= s.End {
				want = append(want, full.funcRefsMap[noFn][i])
			}
		}

		oldRefs := old.oldCachedFuncRefs(s.Start, s.End)
		preRefs, _ := pre.FuncRefs(s.Start, s.End)
		lazyRefs, _ := lazy.FuncRefs(s.Start, s.End)

		d.refs("old", kind, at, full, s, oldRefs, want)
		d.refs("pre", kind, at, full, s, preRefs, want)
		d.refs("lazy", kind, at, full, s, lazyRefs, want)

		wantVars := fullFuncVars(full, s)
		d.vars("old", kind, at, full, s, old.oldParseFuncBody(s.Start, s.End), wantVars)
		d.vars("pre", kind, at, full, s, pre.FuncVars(s.Start, s.End), wantVars)
		d.vars("lazy", kind, at, full, s, lazy.FuncVars(s.Start, s.End), wantVars)

		wantCalls := full.funcCallsMap[funcKey(s.Start, s.End)]
		d.calls("old", kind, at, full, old.oldFuncCallsUncached(s.Start, s.End), wantCalls)
		d.calls("pre", kind, at, full, pre.FuncCalls(s.Start, s.End), wantCalls)
		d.calls("lazy", kind, at, full, lazy.FuncCalls(s.Start, s.End), wantCalls)
		d.calls("uncalled", kind, at, full, noCalls.FuncCalls(s.Start, s.End), wantCalls)
	}

	for _, scope := range []Scope{ScopeVariables, ScopeThis} {
		want := fullAnyScoped(full, scope)
		at := fmt.Sprintf("%s scope %d", path, scope)
		d.names("old", "any-scoped", at, old.oldComputeAnyScopedVars(scope), want)
		d.names("pre", "any-scoped", at, pre.anyScopedNames(scope), want)
		d.names("lazy", "any-scoped", at, lazy.anyScopedNames(scope), want)
	}

	if init := full.initFuncScope(); init.Start >= 0 {
		var want, oldNames []string

		full.bodies.mu.Lock()
		for _, v := range full.bodies.varsOn(init.Start, init.End) {
			if v.Scope == ScopeVariables || v.Scope == ScopeThis {
				want = append(want, v.Name)
			}
		}
		full.bodies.mu.Unlock()

		for _, v := range old.oldInitBodyVars() {
			if v.Scope == ScopeVariables || v.Scope == ScopeThis {
				oldNames = append(oldNames, v.Name)
			}
		}

		at := path + " init"
		d.names("old", "init-scoped", at, oldNames, want)

		var lazyNames []string

		lazy.bodies.mu.Lock()

		for _, v := range lazy.bodies.varsOn(init.Start, init.End) {
			if v.Scope == ScopeVariables || v.Scope == ScopeThis {
				lazyNames = append(lazyNames, v.Name)
			}
		}
		lazy.bodies.mu.Unlock()

		d.names("lazy", "init-scoped", at, lazyNames, want)
	}

	if !slices.Equal(pre.VariablesVars(), lazy.VariablesVars()) || !slices.Equal(pre.ThisVars(), lazy.ThisVars()) {
		d.note("lazy variables/this names differ from pre", path)
	}
}

func mustOffset(pr *ParseResult, line int) int {
	off, _ := pr.lineOffsets(line, line)

	return max(off, 0)
}

// fullFuncVars is what FuncVars means, read from the whole walk's record of
// what it declared (the parse's AllVars keeps one declaration per name and
// line, where the walk can record two): the local and arguments names
// declared on the function's lines.
func fullFuncVars(full *ParseResult, s FuncScope) []string {
	var names []string

	full.bodies.mu.Lock()
	defer full.bodies.mu.Unlock()

	for _, v := range full.bodies.varsOn(s.Start, s.End) {
		if v.Scope == ScopeLocal || v.Scope == ScopeArguments {
			names = append(names, v.Name)
		}
	}

	return names
}

func fullAnyScoped(full *ParseResult, scope Scope) []string {
	var names []string

	full.bodies.mu.Lock()
	defer full.bodies.mu.Unlock()

	in := scopeIndex(full.Scopes, true)

	for _, v := range full.bodies.vars {
		if v.Scope == scope && in(int(v.Line)) >= 0 {
			names = append(names, v.Name)
		}
	}

	return names
}

func nameSet(names []string) map[string]bool {
	out := map[string]bool{}
	for _, n := range names {
		out[n] = true
	}

	return out
}

func (d *bodyDiff) names(who, what, at string, got, want []string) {
	g, w := nameSet(got), nameSet(want)

	for n := range g {
		if !w[n] {
			d.note(fmt.Sprintf("%s %s: extra name", who, what), at+" "+n)
		}
	}

	for n := range w {
		if !g[n] {
			d.note(fmt.Sprintf("%s %s: missing name", who, what), at+" "+n)
		}
	}
}

func (d *bodyDiff) vars(who, kind, at string, full *ParseResult, s FuncScope, got, want []string) {
	g, w := nameSet(got), nameSet(want)

	for n := range g {
		if w[n] {
			continue
		}

		class := "declared nowhere on the function's lines by the parse"

		full.bodies.mu.Lock()
		for _, v := range full.bodies.varsOn(s.Start, s.End) {
			if strings.EqualFold(v.Name, n) {
				class = fmt.Sprintf("the parse declares it in scope %d there", v.Scope)

				break
			}
		}
		full.bodies.mu.Unlock()

		d.note(fmt.Sprintf("%s vars (%s): extra, %s", who, kind, class), at+" "+n)
	}

	for n := range w {
		if !g[n] {
			d.note(fmt.Sprintf("%s vars (%s): missing", who, kind), at+" "+n)
		}
	}
}

func (d *bodyDiff) refs(who, kind, at string, full *ParseResult, s FuncScope, got, want []ComponentRef) {
	byKey := func(refs []ComponentRef) map[string][]string {
		out := map[string][]string{}

		for i := range refs {
			k := refKey(&refs[i])
			out[k] = append(out[k], refs[i].Component)
		}

		for k := range out {
			sort.Strings(out[k])
		}

		return out
	}

	g, w := byKey(got), byKey(want)

	for k, comps := range g {
		wc, ok := w[k]

		switch {
		case ok && slices.Equal(comps, wc):
		case ok:
			d.note(fmt.Sprintf("%s refs (%s): typed differently", who, kind), fmt.Sprintf("%s %s %v want %v", at, k, comps, wc))
		default:
			d.note(fmt.Sprintf("%s refs (%s): extra, %s", who, kind, fullFiling(full, k, s)), at+" "+k+" "+strings.Join(comps, ","))
		}
	}

	for k, comps := range w {
		if _, ok := g[k]; !ok {
			class := "missing"
			if v, _, _ := strings.Cut(k, "@"); strings.Contains(v, ".") {
				class = "missing a member ref"
			}

			d.note(fmt.Sprintf("%s refs (%s): %s", who, kind, class), at+" "+k+" "+strings.Join(comps, ","))
		}
	}
}

// fullFiling is where the whole parse filed a ref keyed k that a reader gave
// function s.
func fullFiling(full *ParseResult, k string, s FuncScope) string {
	for i := range full.ComponentRefs {
		if refKey(&full.ComponentRefs[i]) == k {
			return "the parse files it for the file"
		}
	}

	for key, refs := range full.funcRefsMap {
		for i := range refs {
			if refKey(&refs[i]) != k {
				continue
			}

			switch {
			case !key.in:
				return "the parse files it under a top-level closure"
			case within(key, s):
				return "the parse files it under a function nested in this one"
			default:
				return "the parse files it under another function"
			}
		}
	}

	return "the parse files it nowhere"
}

func (d *bodyDiff) calls(who, kind, at string, full *ParseResult, got, want []CallSite) {
	count := func(calls []CallSite) map[string]int {
		out := map[string]int{}
		for i := range calls {
			out[bodyCallKey(&calls[i])]++
		}

		return out
	}

	g, w := count(got), count(want)

	for k, n := range g {
		if w[k] >= n {
			continue
		}

		class := "the parse records it nowhere"

		for i := range full.Calls {
			if bodyCallKey(&full.Calls[i]) == k {
				class = "the parse records it outside every function"
			}
		}

		for key, calls := range full.funcCallsMap {
			for i := range calls {
				if bodyCallKey(&calls[i]) == k {
					class = "the parse records it under another function (" + map[bool]string{true: "top-level closure", false: "keyed"}[!key.in] + ")"
				}
			}
		}

		d.note(fmt.Sprintf("%s calls (%s): extra, %s", who, kind, class), at+" "+k)
	}

	for k, n := range w {
		if g[k] < n {
			d.note(fmt.Sprintf("%s calls (%s): missing", who, kind), at+" "+k)
		}
	}
}
