package parser

import (
	"slices"
	"testing"
)

// Code outside any function is read by the parse loop's own dispatch; these
// pin the arms it has for a catch, a throw and a dotted chain.

func varNames(vars []VarDef) []string {
	names := make([]string, 0, len(vars))
	for i := range vars {
		names = append(names, vars[i].Name)
	}

	return names
}

func TestATopLevelCatchDeclaresItsVariable(t *testing.T) {
	src := `component {
	try { x = 1; } catch ( any err ) { y = 2; }
}`
	if !slices.Contains(varNames(ParseVars(src)), "err") {
		t.Errorf("declarations are %v, want err from the catch", varNames(ParseVars(src)))
	}
}

func TestATopLevelThrowDeclaresNothingFromItsArguments(t *testing.T) {
	src := `component {
	throw( type = "bad", message = "no" );
}`
	for _, v := range varNames(ParseVars(src)) {
		if v == "type" || v == "message" {
			t.Errorf("a throw's argument %q was declared as a variable", v)
		}
	}
}

func TestATopLevelDottedChainRecordsItsCallWithItsReceiver(t *testing.T) {
	src := `component {
	app.util.helper( 1 );
	app.util.next( a.b() ).done();
}`
	pr := ParseWithOptions(testURI, src, &ParseOptions{ExtractCalls: true})

	var got []string

	calls := pr.AllCalls()
	for i := range calls {
		got = append(got, calls[i].Variable+"."+calls[i].FuncName)
	}

	for _, want := range []string{"app.util.helper", "app.util.next", "a.b"} {
		if !slices.Contains(got, want) {
			t.Errorf("no call %s among %v", want, got)
		}
	}
}

// A dot not followed by a name ends the chain, and a call left open ends the
// parse quietly: neither may panic nor record a call.
func TestATopLevelChainThatDoesNotFinishRecordsNothing(t *testing.T) {
	for _, src := range []string{
		"component {\n\tfoo.\n}",
		"component {\n\tfoo.bar.(\n}",
		"component {\n\tfoo.bar(\n",
	} {
		pr := ParseWithOptions(testURI, src, &ParseOptions{ExtractCalls: true})

		for _, c := range pr.AllCalls() {
			if c.FuncName == "foo" {
				t.Errorf("%q recorded a call to foo", src)
			}
		}
	}
}
