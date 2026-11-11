package resolve

import (
	"math/rand"
	"regexp"
	"slices"
	"strings"
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

// computedIncludeOffsets reads the file's code where this expression read its
// text; in code, it must find what the expression found. (A commented-out
// include is TestTheCallReadersReadOnlyCode's.)
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

// mayIncludeComputed is only a check in front of the reader: on every text
// where the reader finds a computed include, it must say yes.
func TestTheComputedIncludeCheckHidesNoInclude(t *testing.T) {
	texts := []string{
		`<cfinclude template="#x#.cfm">`, `<cfinclude template = '#x#'>`, "<cfinclude\ttemplate\n=\n\"#a#\">",
		`include "#x#";`, "include\f'#x#';", `include /* c */ "#x#";`, `include // c` + "\n" + `"#x#";`,
		`<cfinclude <!--- c ---> template="#x#">`, `cfinclude( template = "#x#" );`, `include "x.cfm";`, `x = "#include#";`,
	}

	r := rand.New(rand.NewSource(7))
	pieces := []string{"include", "INCLUDE", "<cfinclude", "template", " ", "\t", "\n", "\v", "\f", "=", "\"", "'", "#", "x", "/*", "*/", "//", "<!---", "--->", ";", "(", ")", ">"}

	for range 20000 {
		var b strings.Builder
		for range r.Intn(14) {
			b.WriteString(pieces[r.Intn(len(pieces))])
		}

		texts = append(texts, b.String())
	}

	for _, text := range texts {
		if len(computedIncludesRead(text)) > 0 && !mayIncludeComputed(text) {
			t.Fatalf("mayIncludeComputed(%q) is false, but the reader finds %v", text, computedIncludesRead(text))
		}
	}
}
