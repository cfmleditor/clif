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
	src := "component {\n function f( a, b = 1 ) {\n  var ¦x = 1;\n  var y = 2;\n  local.z = 3;\n  g( function( q ) {} );\n }\n}"
	at := strings.Index(src, "¦")
	content := src[:at] + src[at+len("¦"):]

	plan, err := PlanLocalRename(content, FunctionSpansOf("t.cfc", content), at)
	if err != nil {
		t.Fatal(err)
	}

	for _, n := range []string{"a", "b", "x", "y", "q"} {
		if !plan.Taken[n] {
			t.Errorf("%q is not among the names the function uses: %v", n, plan.Taken)
		}
	}
}
