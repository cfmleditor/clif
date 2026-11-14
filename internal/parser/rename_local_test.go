package parser

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// renameLocal plans the rename of the variable at the ¦ in src and applies it,
// returning the text with ¦ removed and the variable renamed to to.
func renameLocal(t *testing.T, src, to string) (string, error) {
	t.Helper()

	at := strings.Index(src, "¦")
	if at < 0 {
		t.Fatal("no cursor in the source")
	}

	content := src[:at] + src[at+len("¦"):]

	plan, err := PlanLocalRename(content, FunctionSpansOf("t.cfc", content), at)
	if err != nil {
		return "", err
	}

	out := content

	for _, r := range slices.Backward(plan.Ranges) {
		out = out[:r[0]] + to + out[r[1]:]
	}

	return out, nil
}

func TestPlanLocalRename(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"a var and its uses", "component {\n function f( a ) {\n  var ¦count = 0;\n  count = count + 1;\n  return count;\n }\n}", "component {\n function f( a ) {\n  var N = 0;\n  N = N + 1;\n  return N;\n }\n}"},
		{"from a use", "component {\n function f() {\n  var count = 0;\n  return ¦count;\n }\n}", "component {\n function f() {\n  var N = 0;\n  return N;\n }\n}"},
		{"a typed var", "component {\n function f() {\n  var numeric count = 0;\n  return ¦count;\n }\n}", "component {\n function f() {\n  var numeric N = 0;\n  return N;\n }\n}"},
		{"local scope", "component {\n function f() {\n  local.count = 0;\n  return ¦count + local.count;\n }\n}", "component {\n function f() {\n  local.N = 0;\n  return N + local.N;\n }\n}"},
		{"case is not significant", "component {\n function f() {\n  var Count = 0;\n  return ¦COUNT;\n }\n}", "component {\n function f() {\n  var N = 0;\n  return N;\n }\n}"},
		{"another object's member and the other scopes are left", "component {\n function f( o ) {\n  var count = 0;\n  o.count = ¦count;\n  variables.count = this.count;\n  return arguments.count;\n }\n}", "component {\n function f( o ) {\n  var N = 0;\n  o.count = N;\n  variables.count = this.count;\n  return arguments.count;\n }\n}"},
		{"a struct key and a named argument are not the variable", "component {\n function f() {\n  var count = 1;\n  var s = { count = count, other: ¦count };\n  g( count = count );\n  return s;\n }\n}", "component {\n function f() {\n  var N = 1;\n  var s = { count = N, other: N };\n  g( count = N );\n  return s;\n }\n}"},
		{"a ternary is not a key", "component {\n function f( ok ) {\n  var x = 1;\n  return g( ok ? ¦x : x );\n }\n}", "component {\n function f( ok ) {\n  var N = 1;\n  return g( ok ? N : N );\n }\n}"},
		{"a comparison is not a label", "component {\n function f() {\n  var x = 1;\n  return g( ¦x == 1 );\n }\n}", "component {\n function f() {\n  var N = 1;\n  return g( N == 1 );\n }\n}"},
		{"a loop variable", "component {\n function f( items ) {\n  for ( var ¦i = 0; i < 3; i++ ) { items[ i ] = i; }\n }\n}", "component {\n function f( items ) {\n  for ( var N = 0; N < 3; N++ ) { items[ N ] = N; }\n }\n}"},
		{"for in", "component {\n function f( items ) {\n  for ( var ¦item in items ) { g( item ); }\n }\n}", "component {\n function f( items ) {\n  for ( var N in items ) { g( N ); }\n }\n}"},
		{"inside a closure", "component {\n function f( items ) {\n  var ¦total = 0;\n  items.each( function( i ) { total += i; } );\n  return total;\n }\n}", "component {\n function f( items ) {\n  var N = 0;\n  items.each( function( i ) { N += i; } );\n  return N;\n }\n}"},
		{"in a string's hash expression", "component {\n function f() {\n  var name = \"x\";\n  return \"hello #¦name# and #ucase( name )# ## name\";\n }\n}", "component {\n function f() {\n  var N = \"x\";\n  return \"hello #N# and #ucase( N )# ## name\";\n }\n}"},
		{"a comment is left", "component {\n function f() {\n  var count = 0; // count it\n  return ¦count;\n }\n}", "component {\n function f() {\n  var N = 0; // count it\n  return N;\n }\n}"},
		{"a variable a closure declares for itself", "component {\n function f( l ) {\n  var x = 1;\n  l.each( function( i ) {\n   var ¦x = i;\n   g( x );\n  } );\n  return x;\n }\n}", "component {\n function f( l ) {\n  var x = 1;\n  l.each( function( i ) {\n   var N = i;\n   g( N );\n  } );\n  return x;\n }\n}"},
		{"the outer variable, from inside a closure that only uses it", "component {\n function f( l ) {\n  var x = 1;\n  l.each( function( i ) {\n   g( ¦x + i );\n  } );\n  return x;\n }\n}", "component {\n function f( l ) {\n  var N = 1;\n  l.each( function( i ) {\n   g( N + i );\n  } );\n  return N;\n }\n}"},
		{"a variable of the middle of three closures", "component {\n function f( l ) {\n  l.each( function( a ) {\n   var ¦y = 1;\n   a.each( function( b ) { g( y, b ); } );\n  } );\n }\n}", "component {\n function f( l ) {\n  l.each( function( a ) {\n   var N = 1;\n   a.each( function( b ) { g( N, b ); } );\n  } );\n }\n}"},
		{"a named function expression's name is not the variable", "component {\n function f( l ) {\n  var ¦helper = 1;\n  l.each( function helper( i ) { g( i ); } );\n  return helper;\n }\n}", "component {\n function f( l ) {\n  var N = 1;\n  l.each( function helper( i ) { g( i ); } );\n  return N;\n }\n}"},
		{"safe navigation and elvis leave no ternary open", "component {\n function f( o ) {\n  var ¦x = 1;\n  return [ g( o?.a, x: 2, y = x ), o.p ?: x, { k: o?.b, x: x } ];\n }\n}", "component {\n function f( o ) {\n  var N = 1;\n  return [ g( o?.a, x: 2, y = N ), o.p ?: N, { k: o?.b, x: N } ];\n }\n}"},
		{"a default naming the variable is the outer one", "component {\n function f( a = count ) {\n  var ¦count = 0;\n  return count + a;\n }\n}", "component {\n function f( a = count ) {\n  var N = 0;\n  return N + a;\n }\n}"},
		{"a function of the same name elsewhere is its own", "component {\n function g() {\n  var count = 5;\n  return count;\n }\n function f() {\n  var ¦count = 0;\n  return count;\n }\n}", "component {\n function g() {\n  var count = 5;\n  return count;\n }\n function f() {\n  var N = 0;\n  return N;\n }\n}"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := renameLocal(t, c.src, "N")
			if err != nil {
				t.Fatalf("refused: %v", err)
			}

			if got != c.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, c.want)
			}
		})
	}
}

func TestPlanLocalRenameRefuses(t *testing.T) {
	for _, c := range []struct{ name, src, why string }{
		{"an argument", "component {\n function f( ¦a ) {\n  return a;\n }\n}", "argument"},
		{"an argument used in the body", "component {\n function f( a ) {\n  return ¦a;\n }\n}", "argument"},
		{"not declared here", "component {\n function f() {\n  return ¦count;\n }\n}", "not declared"},
		{"declared only in a closure", "component {\n function f( l ) {\n  l.each( function() { var x = 1; return x; } );\n  return ¦x;\n }\n}", "inside this one declares"},
		{"a name in a string", "component {\n function f() {\n  var x = 1;\n  return structKeyExists( local, \"x\" ) ? ¦x : 0;\n }\n}", "string"},
		{"a scoped name in a string", "component {\n function f() {\n  var x = 1;\n  return isDefined( \"local.x\" ) ? ¦x : 0;\n }\n}", "string"},
		{"evaluate", "component {\n function f() {\n  var x = 1;\n  return evaluate( \"¦x\" );\n }\n}", "evaluate"},
		{"a closure that declares the name itself", "component {\n function f( l ) {\n  var ¦x = 1;\n  l.each( function() { var x = 2; } );\n }\n}", "inside this one declares"},
		{"a closure that takes the name", "component {\n function f( l ) {\n  var ¦x = 1;\n  l.each( function( x ) { return x; } );\n }\n}", "own argument"},
		{"an arrow function that takes the name", "component {\n function f( l ) {\n  var ¦x = 1;\n  return l.map( ( x ) => x * 2 );\n }\n}", "arrow"},
		{"a catch that binds it", "component {\n function f() {\n  var ¦e = 1;\n  try { g(); } catch ( any e ) { h( e ); }\n }\n}", "catch"},
		{"a tag function", "<cfcomponent>\n<cffunction name=\"f\">\n<cfset var ¦x = 1>\n</cffunction>\n</cfcomponent>", "script function"},
		{"CF tags in the body", "component {\n function f() {\n  var ¦x = 1;\n  ```<cfoutput>#x#</cfoutput>```\n }\n}", "CF tags"},
		{"outside any function", "component {\n ¦x = 1;\n}", "script function"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := renameLocal(t, c.src, "N")
			if err == nil {
				t.Fatal("a rename that cannot be done safely was planned")
			}

			if !errors.Is(err, err) || !strings.Contains(err.Error(), "cannot rename") || !strings.Contains(err.Error(), c.why) {
				t.Errorf("refused with %q, want a refusal mentioning %q", err, c.why)
			}
		})
	}
}

func TestPlanLocalRenameTaken(t *testing.T) {
	src := "component {\n function f( a, string b = limit ) {\n  var ¦x = 1;\n  var numeric y = 2;\n  local.z = 3;\n  var o = new Widget();\n  var s = { key: 1, other = 2 };\n  g( label = 3, o.member, local.w );\n  try { run(); } catch ( any err ) { log( err ); }\n  for ( k in a ) { z += k; }\n  g( function( string q, r = cap ) {} );\n  if ( a ) { g(); } else { g(); }\n  switch ( k ) { case 1: g(); break; default: g(); }\n  return helper( x ) + config.size;\n }\n}"
	at := strings.Index(src, "¦")
	content := src[:at] + src[at+len("¦"):]

	plan, err := PlanLocalRename(content, FunctionSpansOf("t.cfc", content), at)
	if err != nil {
		t.Fatal(err)
	}

	// What the function means by each name, and what that is.
	want := map[string]string{
		"a": takenArgument, "b": takenArgument, "limit": TakenReadOrCalled,
		"x": takenLocal, "y": takenLocal, "z": takenLocal, "o": takenLocal, "s": takenLocal, "w": takenLocal,
		"err": takenCatch, "k": TakenReadOrCalled, "q": takenArgument, "r": takenArgument, "cap": TakenReadOrCalled,
		"helper": TakenReadOrCalled, "config": TakenReadOrCalled, "run": TakenReadOrCalled, "g": TakenReadOrCalled,
	}
	for name, why := range want {
		if plan.Taken[name] != why {
			t.Errorf("%q is %q, want %q", name, plan.Taken[name], why)
		}
	}

	// Not names the function means anything by: a type, a key, a label, another
	// object's member, a word the language reserves.
	for _, name := range []string{"string", "numeric", "any", "widget", "key", "other", "label", "member", "size", "var", "try", "catch", "for", "in", "return", "new", "local", "default", "else"} {
		if why, taken := plan.Taken[name]; taken {
			t.Errorf("%q is taken as %q, though it is a type, key, label or member", name, why)
		}
	}
}

// A closure's variable is renamed among the names of the function around it,
// the same way as a function's own: one walk finds them for both.
func TestPlanLocalRenameOfAClosureCountsTheFunctionAroundIt(t *testing.T) {
	src := "component {\n function f( l, count ) {\n  var total = config.size;\n  l.each( function( i ) {\n   var ¦u = i + total;\n   return u;\n  } );\n }\n}"
	at := strings.Index(src, "¦")
	content := src[:at] + src[at+len("¦"):]

	plan, err := PlanLocalRename(content, FunctionSpansOf("t.cfc", content), at)
	if err != nil {
		t.Fatal(err)
	}

	for name, why := range map[string]string{"i": takenArgument, "u": takenLocal, "total": TakenReadOrCalled, "count": takenArgument, "config": TakenReadOrCalled} {
		if plan.Taken[name] == "" {
			t.Errorf("%q is not taken, want %q", name, why)
		}
	}

	// The strongest thing a name is wins: `total` is the outer variable, read
	// in the closure too.
	if plan.Taken["total"] != takenLocal {
		t.Errorf("total is %q, want %q", plan.Taken["total"], takenLocal)
	}
}

func TestPlanLocalRenameNotesAStatementStart(t *testing.T) {
	for _, c := range []struct {
		name, src string
		want      bool
	}{
		{"written at the start of a statement", "component {\n function f() {\n  var ¦x = 0;\n  x = x + 1;\n  return x;\n }\n}", true},
		{"after a block's brace", "component {\n function f( a ) {\n  var ¦x = 0;\n  if ( a ) {\n   x++;\n  }\n  return x;\n }\n}", true},
		{"after the condition of an if", "component {\n function f( a ) {\n  var ¦x = 0;\n  if ( a ) x = 1;\n  return g( x );\n }\n}", true},
		{"after else", "component {\n function f( a ) {\n  var ¦x = 0;\n  if ( a ) { g(); } else x = 2;\n  return g( x );\n }\n}", true},
		{"after do", "component {\n function f() {\n  var ¦x = 0;\n  do x++; while ( x < 3 );\n  return g( x );\n }\n}", true},
		{"after a while's condition", "component {\n function f( a ) {\n  var ¦x = 0;\n  while ( a ) x++;\n  return g( x );\n }\n}", true},
		{"after a case label", "component {\n function f( a ) {\n  var ¦x = 0;\n  switch ( a ) { case 1: x = 1; break; }\n  return g( x );\n }\n}", true},
		{"after default:", "component {\n function f( a ) {\n  var ¦x = 0;\n  switch ( a ) { default: x = 1; }\n  return g( x );\n }\n}", true},
		{"after a call's paren is no statement", "component {\n function f( a ) {\n  var ¦x = 0;\n  return g( a ) + x;\n }\n}", false},
		{"after a ternary's colon is no statement", "component {\n function f( a ) {\n  var ¦x = 0;\n  return a ? 1 : x;\n }\n}", false},
		{"in a string's hash expression is no statement", "component {\n function f() {\n  var ¦x = 1;\n  return \"#x#\" & g( \"#x#\" );\n }\n}", false},
		{"after a closed block", "component {\n function f( a ) {\n  var ¦x = 0;\n  if ( a ) { g(); }\n  x = 2;\n  return g( x );\n }\n}", true},
		{"only in expressions", "component {\n function f( a ) {\n  var ¦x = a * 2;\n  return g( x ) + x;\n }\n}", false},
		{"first in a struct literal is no statement", "component {\n function f() {\n  var ¦x = 1;\n  return { x: x };\n }\n}", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			at := strings.Index(c.src, "¦")
			content := c.src[:at] + c.src[at+len("¦"):]

			plan, err := PlanLocalRename(content, FunctionSpansOf("t.cfc", content), at)
			if err != nil {
				t.Fatal(err)
			}

			if plan.StatementStart != c.want {
				t.Errorf("StatementStart is %v, want %v", plan.StatementStart, c.want)
			}
		})
	}
}

// The cursor just after the name, where an editor puts it at the end of a
// word, is on it, and two names are never close enough to share the boundary.
func TestRangeAtIncludesTheEndOfTheName(t *testing.T) {
	src := "component {\n function f() {\n  var ¦x = 1;\n  return x+x;\n }\n}"
	at := strings.Index(src, "¦")
	content := src[:at] + src[at+len("¦"):]

	plan, err := PlanLocalRename(content, FunctionSpansOf("t.cfc", content), at)
	if err != nil {
		t.Fatal(err)
	}

	first := strings.LastIndex(content, "x+x")

	for _, c := range []struct {
		name  string
		at    int
		start int
		ok    bool
	}{
		{"the start of the first use", first, first, true},
		{"just after the first use", first + 1, first, true},
		{"the second use", first + 2, first + 2, true},
		{"just after the second", first + 3, first + 2, true},
		{"the space after the declaration", strings.Index(content, "x = 1") + 1, strings.Index(content, "x = 1"), true},
		{"the keyword before it", strings.Index(content, "var x"), 0, false},
	} {
		start, _, ok := plan.RangeAt(c.at)
		if ok != c.ok || ok && start != c.start {
			t.Errorf("%s: RangeAt(%d) = %d, %v, want %d, %v", c.name, c.at, start, ok, c.start, c.ok)
		}
	}
}

func TestStringContentOfACutOffString(t *testing.T) {
	for in, want := range map[string]string{
		`"abc"`: "abc", `'abc'`: "abc", `""`: "", `"abc`: "abc", `"#x#`: "#x#", `"`: "", ``: "", `'a"`: `a"`,
		`"a""b"`: `a""b`, `"a""`: `a""`, `'it''s'`: `it''s`, `'#x#''`: `#x#''`, `"a""b`: `a""b`,
	} {
		if got := stringContent(in); got != want {
			t.Errorf("stringContent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHashSegments(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"hello #name# and #ucase( n )#", []string{"name", "ucase( n )"}},
		{"## not one #x#", []string{"x"}},
		{"cut off #x", nil},
		{"#x# then #y", []string{"x"}},
		{"none", nil},
		{"##", nil},
	} {
		var got []string
		for _, seg := range hashSegments(c.in) {
			got = append(got, c.in[seg[0]:seg[1]])
		}

		if !slices.Equal(got, c.want) {
			t.Errorf("hashSegments(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParamNamesSkipAttributes(t *testing.T) {
	for in, want := range map[string][]string{
		`string x default="y"`:                      {"x"},
		`x hint="h", y`:                             {"x", "y"},
		`required string z`:                         {"z"},
		`a = 1, string b = "q" hint="h", c`:         {"a", "b", "c"},
		`required numeric n default="3" hint="how"`: {"n"},
		`type = "a"`:                                {"type"},
		`default = 1, hint`:                         {"default", "hint"},
		``:                                          nil,
	} {
		if got := paramNames(in); !slices.Equal(got, want) {
			t.Errorf("paramNames(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClassifyVariableName(t *testing.T) {
	for name, want := range map[string]NameWord{
		"count": NameFree, "result": NameFree, "Default2": NameFree,
		"return": NameReserved, "LOCAL": NameReserved, "default": NameReserved, "true": NameReserved, "else": NameReserved,
		"lock": NameStatementWord, "Param": NameStatementWord, "thread": NameStatementWord, "static": NameStatementWord, "import": NameStatementWord,
	} {
		if got := ClassifyVariableName(name); got != want {
			t.Errorf("ClassifyVariableName(%q) = %v, want %v", name, got, want)
		}
	}
}
