package resolve

import (
	"strings"
	"testing"
)

// TestACustomTagReadsWhatItsCallsPass: a custom tag's attributes scope is what
// the `<cf_name …>` calling it passes. MasaCMS's admin calls its file picker
// as `<cf_fileselector bean="#rc.contentBean#" …>`, and the tag's
// attributes.bean was typed nowhere. Each call whose attribute is one `#expr#`
// types it where the call is made, and the calls are the alternatives; a tag
// handing an attribute on to another tag passes what it holds. A call passing
// the attribute as text, or an attributeCollection, leaves it untyped, as
// does a tag name two files in the workspace answer to, and a tag that sets
// the attribute itself; a call that omits it says nothing, and a cfparam
// default fills only what was not passed.
func TestACustomTagReadsWhatItsCallsPass(t *testing.T) {
	base := map[string]string{
		"models/Item.cfc":  `component { function getValue( k ) {} function itemOnly() {} }`,
		"models/Other.cfc": `component { function getValue( k ) {} }`,
		"customtags/picker.cfm": `<cfparam name="attributes.label" default="">
<cfparam name="attributes.bean" default="">
<cfoutput>#attributes.bean.getValue( "x" )#</cfoutput>
<cf_tools bean="#attributes.bean#">`,
		"customtags/tools.cfm": `<cfset v = attributes.bean.getValue( "y" )>`,
		"views/a.cfm": `<cfset item = new models.Item()>
<cf_picker bean="#item#" label="A">`,
		"views/b.cfm": `<cfset other = new models.Other()>
<cf_picker label="B"
	bean="#other#">
<cf_picker label="none">`,
	}

	for _, tc := range []struct {
		name  string
		extra map[string]string
		want  string
	}{
		{"every call typed", nil, ""},
		{"an attribute passed as text", map[string]string{"views/c.cfm": `<cf_picker bean="none">`}, "no component ref"},
		{"an attributeCollection", map[string]string{"views/c.cfm": `<cfset args = {}><cf_picker attributeCollection="#args#">`}, "no component ref"},
		{"two templates of the name", map[string]string{"other/picker.cfm": `<p>x</p>`}, "no component ref"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, base)
			writeFiles(t, dir, tc.extra)

			r := &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}

			got := reasonsWith(t, r, dir, "customtags/picker.cfm")
			if g := got["attributes.bean.getValue"]; tc.want == "" && g != "" || tc.want != "" && !strings.Contains(g, tc.want) {
				t.Errorf("picker's attributes.bean.getValue: %q, want %q", g, tc.want)
			}

			if tc.want != "" {
				return
			}

			// The tag the picker hands its bean to reads the same alternatives.
			got = reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "customtags/tools.cfm")
			if g := got["attributes.bean.getValue"]; g != "" {
				t.Errorf("tools' attributes.bean.getValue: %q, want resolved", g)
			}
		})
	}

	// A tag that sets the attribute itself reads its own value.
	dir := t.TempDir()
	writeFiles(t, dir, base)
	writeFiles(t, dir, map[string]string{"customtags/picker.cfm": `<cfset attributes.bean = makeBean()>
<cfoutput>#attributes.bean.getValue( "x" )#</cfoutput>`})

	got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "customtags/picker.cfm")
	if g := got["attributes.bean.getValue"]; !strings.Contains(g, "no component ref") {
		t.Errorf("a tag setting its own attribute: %q, want untyped", g)
	}
}
