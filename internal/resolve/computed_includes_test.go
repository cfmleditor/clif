package resolve

import (
	"regexp"
	"slices"
	"testing"
)

// TestATemplateIncludedByAComputedPathReadsItsIncluder: Masa's form builder
// names its templates' directory in a string and includes
// "#templatePath#" from functions that set mmRBF first. A template in a
// directory no string names is not reached.
func TestATemplateIncludedByAComputedPathReadsItsIncluder(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Rb.cfc": `component { function getKeyValue( k ){ return k; } }`,
		"core/Manager.cfc": `component {
	variables.templatePath = "/app/core/utilities/formbuilder/templates";
	function render( name ){
		var mmRBF = new Rb();
		var templatePath = variables.templatePath & "/" & arguments.name;
		include "#templatePath#";
	}
}`,
		"core/utilities/formbuilder/templates/field.cfm": `<cfoutput>#mmRBF.getKeyValue( "x" )##mmRBF.nope()#</cfoutput>`,
		"core/utilities/other/templates/field.cfm":       `<cfoutput>#mmRBF.getKeyValue( "x" )#</cfoutput>`,
	})

	r := &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}

	expectReasons(t, reasonsWith(t, r, dir, "core/utilities/formbuilder/templates/field.cfm"), map[string]string{
		"mmRBF.getKeyValue": "",
		"mmRBF.nope":        "method 'nope' not found in Rb",
	})
	expectReasons(t, reasonsWith(t, r, dir, "core/utilities/other/templates/field.cfm"), map[string]string{
		"mmRBF.getKeyValue": "variable 'mmRBF' has no component ref",
	})
}

// computedIncludeOffsets finds computed includes where the word include is,
// rather than running this expression over the whole file; it must find what
// the expression found.
var computedIncludeReference = regexp.MustCompile(`(?i)<cfinclude\s+template\s*=\s*["']#[^"']*#["']|\binclude\s+["']#[^"']*#["']`)

func TestComputedIncludeOffsetsMatchTheExpression(t *testing.T) {
	for _, src := range []string{
		`<cfinclude template="#path#/x.cfm">`,
		`<CFINCLUDE TEMPLATE = '#variables.templatePath#'>`,
		`include "#templatePath#/a.cfm";`,
		`xinclude "#a#"; myinclude "#b#"; include  '#c#'`,
		`<cfinclude template="#includeDir#include.cfm"> include "#d#"`,
		`<cfinclude template="literal.cfm"> include "plain.cfm";`,
		`<cf_include template="#x#"> <cfinclude template="#a"`,
		"include\n\"#multi#\"",
		"",
	} {
		var want []int
		for _, m := range computedIncludeReference.FindAllStringIndex(src, -1) {
			want = append(want, m[0])
		}

		if got := computedIncludeOffsets(src); !slices.Equal(got, want) {
			t.Errorf("computedIncludeOffsets(%q) = %v, want %v", src, got, want)
		}
	}
}
