package parser

import "testing"

// A function's refs are what the parse files under it, typed as the parse
// types them, whether the parse walked the body or an edit has changed it
// since and it is walked on its own. That walk once resolved no pending
// calls, so `y = y.next()` and `z = make()` added nothing there and the
// variable kept its first type.
func TestLazyFuncRefsAreTypedAsTheParseTypesThem(t *testing.T) {
	src := `component {
function make() { return new models.B(); }
function scoped() {
	var y = new models.A();
	y = y.next();
	y.onlyB();
}
function bare() {
	var z = new models.A();
	z = make();
}
}`
	lookup := func(comp, fn string) string {
		if comp == "models.A" && fn == "next" {
			return "models.B"
		}

		return ""
	}

	pr := ParseWithOptions(testURI, src, &ParseOptions{FuncLookup: lookup})

	check := func(when string) {
		t.Helper()

		for _, tc := range []struct{ fn, name string }{{"scoped", "y"}, {"bare", "z"}} {
			scope := scopeNamed(pr, tc.fn)

			refs, _ := pr.FuncRefs(scope.Start, scope.End)
			if got := latestRef(refs, tc.name); got != "models.B" {
				t.Errorf("%s: %s's latest %s ref is %q, want models.B (refs %+v)", when, tc.fn, tc.name, got, refs)
			}
		}
	}

	check("parsed")

	for _, s := range pr.Scopes {
		pr.InvalidateFunc(s.Start, s.End)
	}

	check("after an edit")
}

// A write to the variables scope inside a function, scoped or not, is filed
// for the file, where every function reads it; the function's own refs hold
// its locals. The body walk FuncRefs fell back to filed both copies under a
// function the parse filed nothing under, so whether a function held the
// write depended on whether it also had a local, and on whether an edit had
// made FuncRefs walk it again.
func TestLazyFuncRefsAreFiledAsTheParseFilesThem(t *testing.T) {
	src := `component {
function scoped() {
	variables.y = new models.A();
}
function bare() {
	z = new models.A();
}
function both() {
	var w = new models.B();
	z = new models.A();
}
}`
	pr := ParseWithOptions(testURI, src, &ParseOptions{})

	check := func(when string) {
		t.Helper()

		for _, tc := range []struct {
			fn, name, want string
		}{{"scoped", "y", ""}, {"bare", "z", ""}, {"both", "z", ""}, {"both", "w", "models.B"}} {
			scope := scopeNamed(pr, tc.fn)

			refs, _ := pr.FuncRefs(scope.Start, scope.End)
			if got := latestRef(refs, tc.name); got != tc.want {
				t.Errorf("%s: %s's %s ref is %q, want %q", when, tc.fn, tc.name, got, tc.want)
			}
		}

		for _, name := range []string{"y", "z"} {
			if got := latestRef(pr.ComponentRefs, name); got != "models.A" {
				t.Errorf("%s: the file's %s ref is %q, want models.A", when, name, got)
			}
		}
	}

	check("parsed")

	for _, s := range pr.Scopes {
		pr.InvalidateFunc(s.Start, s.End)
	}

	check("after an edit")
}

func scopeNamed(pr *ParseResult, name string) FuncScope {
	for _, s := range pr.Scopes {
		if s.Name == name {
			return s
		}
	}

	return FuncScope{Start: -1, End: -1}
}

// latestRef is the component of the last ref to name in refs, by line.
func latestRef(refs []ComponentRef, name string) string {
	var best *ComponentRef

	for i := range refs {
		if refs[i].Variable == name && (best == nil || refs[i].Line >= best.Line) {
			best = &refs[i]
		}
	}

	if best == nil {
		return ""
	}

	return best.Component
}
