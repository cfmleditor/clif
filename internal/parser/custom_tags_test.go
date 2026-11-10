package parser

import (
	"fmt"
	"strings"
	"testing"
)

// describe renders a call as kind:prefix:name, with flags, for comparison.
func describe(c *CustomTagCall) string {
	kinds := map[CustomTagKind]string{
		CustomTagUnderscore: "cf_", CustomTagModuleName: "name", CustomTagModuleTemplate: "template",
		CustomTagImported: "import", CustomTagPrefixed: "prefixed",
	}

	s := fmt.Sprintf("%s:%s:%s@%d", kinds[c.Kind], c.Prefix, c.Name, c.Line)
	if c.Taglib != "" {
		s += " lib=" + c.Taglib
	}

	if c.Closing {
		s += " /"
	}

	if c.Script {
		s += " script"
	}

	if c.Dynamic {
		s += " dynamic"
	}

	return s
}

func TestCustomTagCallsReadsEverySyntax(t *testing.T) {
	cases := []struct {
		name, src string
		prefixes  []string
		want      []string
	}{
		{
			name: "cf_ open and close tags",
			src:  "<cf_layout title=\"x\">\n  body\n</cf_layout>\n<cf_Br/>",
			want: []string{"cf_::layout@0", "cf_::layout@2 /", "cf_::Br@3"},
		},
		{
			name: "cfmodule by template and by name",
			src:  "<cfmodule template=\"tab.cfm\" name=\"sec\">\n<cfmodule name=\"ui.grid\" rows=\"#q#\">\n<cfmodule template=\"#dyn#.cfm\">",
			want: []string{"template::tab.cfm@0", "name::ui.grid@1", "template::#dyn#.cfm@2 dynamic"},
		},
		{
			name: "cfimport prefixed tags, and a JSP library left out",
			src:  "<cfimport taglib=\"../customtags\" prefix=\"ct\">\n<cfimport taglib=\"/WEB-INF/x.tld\" prefix=\"jsp\">\n<ct:customtag />\n<jsp:thing/>\n<CT:Other></CT:Other>",
			want: []string{"import:ct:customtag@2 lib=../customtags", "import:ct:Other@4 lib=../customtags", "import:ct:Other@4 lib=../customtags /"},
		},
		{
			name: "an unimported prefix is not a custom tag",
			src:  "<x:thing/>\n<svg:path d=\"x\"/>",
		},
		{
			name: "script cf_ and cfmodule calls",
			src:  "<cfscript>\ncf_redden (message=\"x\");\ncfmodule( template=\"cfmodule_target.cfm\", foo=\"FUNC\" );\nobj.cf_notatag();\n</cfscript>",
			want: []string{"cf_::redden@1 script", "template::cfmodule_target.cfm@2 script"},
		},
		{
			name: "Lucee's module statement, with and without a body",
			src:  "component {\n\tfunction f() {\n\t\tmodule template=\"Jira3165/index.cfm\";\n\t\tmodule template='./customtags/sometag2.cfc' {\n\t\t\techo(1);\n\t\t}\n\t\tmodule name=\"a.b\" x=1;\n\t\tvar module = 1;\n\t}\n}",
			want: []string{"template::Jira3165/index.cfm@2 script", "template::./customtags/sometag2.cfc@3 script", "name::a.b@6 script"},
		},
		{
			name:     "a configured tag library prefix, longest first",
			src:      "<cfabridge x=\"1\">\n<cfoutput>#x#</cfoutput>\n</cfabridge>",
			prefixes: []string{"cf", "cfa"},
			want:     []string{"prefixed:cfa:bridge@0", "prefixed:cf:output@1", "prefixed:cf:output@1 /", "prefixed:cfa:bridge@2 /"},
		},
		{
			name: "calls inside a CFML comment are not calls",
			src:  "<!--- <cf_old> <!--- nested ---> <cfmodule template=\"x.cfm\"> --->\n<cf_new>",
			want: []string{"cf_::new@1"},
		},
		{
			name: "script comments are not calls",
			src:  "<cfscript>\n// cf_old();\n/* cfmodule(template=\"x.cfm\"); */\ncf_new();\n</cfscript>",
			want: []string{"cf_::new@3 script"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := CustomTagCalls(tc.src, tc.prefixes)

			var got []string

			for i := range calls {
				got = append(got, describe(&calls[i]))
			}

			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Errorf("got\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(tc.want, "\n  "))
			}
		})
	}
}

// Start and End cover exactly the text naming the tag, which is what the
// cursor is matched against and a document link underlines.
func TestCustomTagCallSpansNameTheTag(t *testing.T) {
	src := "<cfimport taglib=\"lib\" prefix=\"ui\">\n<cf_layout>\n<ui:button/>\n<cfscript>cfmodule(template=\"a/b.cfm\");</cfscript>\n<cfmodule name=\"x.y\">"
	want := []string{"layout", "button", "a/b.cfm", "x.y"}

	calls := CustomTagCalls(src, nil)
	if len(calls) != len(want) {
		t.Fatalf("got %d calls, want %d: %+v", len(calls), len(want), calls)
	}

	for i, c := range calls {
		if got := src[c.Start:c.End]; got != want[i] {
			t.Errorf("call %d spans %q, want %q", i, got, want[i])
		}
	}
}

func TestCustomTagCallAtFindsTheCallUnderTheCursor(t *testing.T) {
	src := "<p>\n  <cf_layout title=\"x\">\n</cf_layout>"

	for _, tc := range []struct {
		line, char int
		want       bool
	}{
		{1, 6, true},   // the start of "layout"
		{1, 11, true},  // its end
		{1, 15, false}, // the attribute
		{2, 6, true},   // the end tag
		{0, 1, false},
	} {
		c, ok := CustomTagCallAt(src, tc.line, tc.char, nil)
		if ok != tc.want || ok && c.Name != "layout" {
			t.Errorf("at %d:%d got %+v, %v; want a call: %v", tc.line, tc.char, c, ok, tc.want)
		}
	}
}

// A cfimport prefix applies from where it stands, as the engines compile it:
// the walk that finds the tags reads the imports too.
func TestCustomTagImportAppliesFromWhereItStands(t *testing.T) {
	src := "<ui:early/>\n<cfimport taglib=\"lib\" prefix=\"ui\">\n<ui:late/>"

	calls := CustomTagCalls(src, nil)
	if len(calls) != 1 || calls[0].Name != "late" {
		t.Fatalf("got %+v, want only ui:late", calls)
	}
}

// The parse keeps its answer, and reads the regions it split rather than
// splitting the file again; an edit inside a function changes Content and
// leaves those regions, which must then not be read.
func TestParseResultCustomTagCallsFollowItsContent(t *testing.T) {
	src := "<cfcomponent>\n<cffunction name=\"f\">\n<cf_old>\n</cffunction>\n</cfcomponent>"
	pr := ParseWithOptions("file:///t.cfc", src, nil)

	first := pr.CustomTagCalls(nil)
	if len(first) != 1 || first[0].Name != "old" {
		t.Fatalf("got %+v, want cf_old", first)
	}

	if again := pr.CustomTagCalls(nil); &again[0] != &first[0] {
		t.Error("asked twice for the same content, the calls were read twice")
	}

	edited := strings.Replace(src, "cf_old", "cf_newer", 1)
	if kind := pr.ApplyEditResult(2, 4, 2, 7, "newer", edited); kind != EditInFunc {
		t.Fatalf("the edit was %v, want one inside the function, which keeps the regions", kind)
	}

	got := pr.CustomTagCalls(nil)
	if len(got) != 1 || got[0].Name != "newer" || got[0].Line != 2 || edited[got[0].Start:got[0].End] != "newer" {
		t.Fatalf("after the edit got %+v, want cf_newer on line 2", got)
	}
}

func TestTagAttributes(t *testing.T) {
	got := TagAttributes(` Name = "a b" mode='x' size=3 flag title="has name=fake" Name2=last/>`)
	want := map[string]string{"name": "a b", "mode": "x", "size": "3", "title": "has name=fake", "name2": "last"}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

// A script cfmodule's template is static only when it is the string literal
// alone; a key inside a nested struct is not the tag's; and a function
// declared as cf_name is no call.
func TestScriptCustomTagCallsReadOnlyTheirOwnLiterals(t *testing.T) {
	content := `component {
	function cf_helper() {}
	function run() {
		cfmodule(template="/tags/" & name & ".cfm");
		cfmodule(template="a.cfm", args={template: "b.cfm"});
		module template="c" & ext;
		if (nav) module template="nav.cfm";
		else module template="none.cfm";
		cf_helper();
	}
}`

	var got []string

	for _, c := range CustomTagCalls(content, nil) {
		got = append(got, describe(&c))
	}

	want := []string{
		"template::/tags/@3 script dynamic",
		"template::a.cfm@4 script",
		"template::c@5 script dynamic",
		"template::nav.cfm@6 script",
		"template::none.cfm@7 script",
		"cf_::helper@8 script",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
