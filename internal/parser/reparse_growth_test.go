package parser

import (
	"reflect"
	"strings"
	"testing"
)

// fieldLengths is the length of each slice and map the parse holds, by field.
func fieldLengths(pr *ParseResult) map[string]int {
	out := map[string]int{}
	v := reflect.ValueOf(pr).Elem()

	for i := range v.NumField() {
		switch f := v.Field(i); f.Kind() {
		case reflect.Slice, reflect.Map:
			out[v.Type().Field(i).Name] = f.Len()
		default:
		}
	}

	return out
}

const reparseSource = `component accessors=true {
	property name="a" inject="svc";
	property name="b" type="Foo";
	property name="c";

	function init() { variables.x = new Bar(); }
	function go() { return variables.x; }
}
`

// A parse reparsed after an edit outside a function (the edit every
// keystroke between functions is) holds what a fresh parse of the same text
// does. Properties was not reset, so each reparse listed every property once
// more, and the refs made from them grew with it: 27 KB of an editor's heap
// per keystroke, 72 MB after 2,400.
func TestAReparsedParseHoldsWhatAFreshOneDoes(t *testing.T) {
	for name, source := range map[string]string{"script": reparseSource, "tags": reparseTagSource} {
		t.Run(name, func(t *testing.T) { reparsedHoldsWhatFreshDoes(t, source) })
	}
}

const reparseTagSource = `<cfcomponent accessors="true" extends="Base">
<cfproperty name="a" inject="svc">
<cfproperty name="b" type="Foo">
<cfproperty name="c">
<cfinclude template="helper.cfm">
<cffunction name="init">
	<cfset variables.x = new Bar()>
	<cfinvoke component="Other" method="run">
	<cfreturn this>
</cffunction>
<cffunction name="go">
	<cfscript>var y = createObject("component", "Thing");</cfscript>
	<cfreturn variables.x>
</cffunction>
</cfcomponent>
`

func reparsedHoldsWhatFreshDoes(t *testing.T, source string) {
	t.Helper()

	opts := &ParseOptions{ExtractCalls: true}
	pr := ParseWithOptions(testURI, source, opts)

	text := source
	for i := range 25 {
		text = strings.Replace(text, "go", "go"+strings.Repeat("x", i%3), 1)
		if i%2 == 0 {
			pr.ApplyFullReplace(text)
		} else {
			pr.ApplyEdit(0, 0, 0, 0, " ")
			text = pr.Content
		}
	}

	pr.ApplyFullReplace(text)

	fresh := ParseWithOptions(testURI, text, opts)
	got, want := fieldLengths(pr), fieldLengths(fresh)

	for field, n := range want {
		if got[field] != n {
			t.Errorf("%s holds %d after the reparses, %d in a fresh parse", field, got[field], n)
		}
	}

	if want["Properties"] != 3 || want["Funcs"] == 0 {
		t.Errorf("the source's %d properties are not what the test is built on", want["Properties"])
	}
}
