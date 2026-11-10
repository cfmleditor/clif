package parser

import (
	"maps"
	"testing"
)

// The shapes the five attribute readers disagreed on, read as the tree-sitter
// grammar reads them (.ai/docs/SCANNER-CONSOLIDATION.md, section 7).
func TestAttributesAreReadAsTheGrammarReadsThem(t *testing.T) {
	for _, tc := range []struct {
		tag  string
		mode attrMode
		want map[string]string
	}{
		// A doubled quote escapes one in a CF tag, and ends the value in markup.
		{`<cfparam name="x" default="a""b">`, cfAttrs, map[string]string{"name": "x", "default": `a""b`}},
		{`<a href="a""b">`, htmlAttrs, map[string]string{"href": "a"}},
		// A #...# holding strings in the value's own quote.
		{
			`<cfparam name="d" default="#StructKeyExists(form,'default')?form.default:""#">`, cfAttrs,
			map[string]string{"name": "d", "default": `#StructKeyExists(form,'default')?form.default:""#`},
		},
		{`<cfset x="#f("a")#" y="1">`, cfAttrs, map[string]string{"x": `#f("a")#`, "y": "1"}},
		// A stray # is text, not an expression running into the next value.
		{`<cfchart color="#fff" size="2">`, cfAttrs, map[string]string{"color": "#fff", "size": "2"}},
		// The other quote inside a value is its text.
		{`<cfthrow detail="The 'CacheFactory' attribute">`, cfAttrs, map[string]string{"detail": "The 'CacheFactory' attribute"}},
		// Space around =, names with digits, unquoted values to a space or />.
		{"<cffunction\n\tname      =\"toXML\"\n\taccess    =\"public\">", cfAttrs, map[string]string{"name": "toXML", "access": "public"}},
		{`<cfchart show3d="false" default=0 basictab=false/>`, cfAttrs, map[string]string{"show3d": "false", "default": "0", "basictab": "false"}},
		{`<cfinclude template=dir/file.cfm>`, cfAttrs, map[string]string{"template": "dir/file.cfm"}},
		// A name given twice keeps its first value; one with no value is left out.
		{`<input checked name="a" name="b">`, htmlAttrs, map[string]string{"name": "a"}},
	} {
		if got := tagAttrMap(tc.tag, tc.mode); !maps.Equal(got, tc.want) {
			t.Errorf("%s\n got  %v\n want %v", tc.tag, got, tc.want)
		}
	}

	// A name is found only where an attribute stands, not inside a value.
	if got := getAttr(`<cfargument hint="type it: name = x" name="real">`, "name"); got != "real" {
		t.Errorf("getAttr found %q inside a value, want real", got)
	}
}
