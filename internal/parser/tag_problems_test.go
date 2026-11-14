package parser

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

// problemsOf is each problem as "kind:underlined text".
func problemsOf(content string) []string {
	var out []string

	for _, p := range TagProblems(content) {
		out = append(out, string(p.Kind)+":"+content[p.Start:p.End])
	}

	slices.Sort(out)

	return out
}

func TestTagProblems(t *testing.T) {
	for _, c := range []struct {
		name, src string
		want      []string
	}{
		{"balanced", "<cfif a><cfloop from=1 to=2 index=i><cfoutput>#i#</cfoutput></cfloop><cfelse>x</cfif>", nil},
		{"a page that runs: a function, a try and a switch", `<cfcomponent>
<cffunction name="f"><cfargument name="a">
<cftry><cfset x = 1><cfcatch type="any"><cfrethrow></cfcatch><cffinally>x</cffinally></cftry>
<cfswitch expression="#a#"><cfcase value="1">a</cfcase><cfdefaultcase>b</cfdefaultcase></cfswitch>
<cfreturn x></cffunction></cfcomponent>`, nil},
		{"unclosed if", "<cfif a>text", []string{"unclosed-tag:<cfif"}},
		{"unclosed inside a closed parent", "<cfloop from=1 to=2 index=i><cfif a></cfloop>", []string{"unclosed-tag:<cfif"}},
		{"unmatched close", "<cfset a = 1>text</cfif>", []string{"unmatched-close:</cfif"}},
		{"crossed", "<cfif a><cfloop from=1 to=2 index=i></cfif></cfloop>", []string{"unclosed-tag:<cfloop", "unmatched-close:</cfloop"}},
		{"case is not significant", "<CFIF a></cfIf>", nil},
		{"self closed", "<cfloop from=1 to=2 index=i />", nil},
		{"html is not held to pairs", "<p>one<p>two<br><div></p>", nil},
		{"a tag that sometimes stands alone is left alone", "<cfhttp url=x><cftransaction action=\"commit\"></cfmail>", nil},
		{"custom tags are left alone", "<cf_box><cf_card></cf_box>", nil},
		{"catch outside try", "<cfcatch type=any>x</cfcatch>", []string{"misplaced-tag:<cfcatch"}},
		{"else outside if", "<cfelse>x", []string{"misplaced-tag:<cfelse"}},
		{"case outside switch", "<cfif a><cfcase value=1>x</cfcase></cfif>", []string{"misplaced-tag:<cfcase"}},
		{"argument outside function", "<cfcomponent><cfargument name=a></cfcomponent>", []string{"misplaced-tag:<cfargument"}},
		{"function without a name", "<cffunction><cfargument name=a></cffunction>", []string{"missing-attribute:<cffunction"}},
		{"attributeCollection gives the name", `<cflocation attributeCollection="#arguments#">`, nil},
		{"include without a template", "<cfinclude>", []string{"missing-attribute:<cfinclude"}},
		{"a valueless name is not missing", "<cfparam name>", nil},
		{"duplicate attribute", `<cfparam name="a" name="b">`, []string{"duplicate-attribute:name"}},
		{"an expression is no attribute list", "<cfset a = 1 and a = 2><cfif a = 1 and a = 2>x</cfif>", nil},
		{"comments hide tags", "<!--- <cfif a> ---><!-- <cfloop> --><cfif a>x</cfif>", nil},
		{"a close in a comment", "<cfif a>x<!--- </cfif> ---></cfif>", nil},
		{"a quoted value hides tags", `<cfset x = "<cfif a>"><cfinvoke component="<cfloop>" method="m">`, nil},
		{"CF tags written in an HTML attribute value", `<a class="x<cfif a> on</cfif>" href="#h#">y</a>`, nil},
		{"one left open in a value", `<a class="x<cfif a> on">y</a>`, []string{"unclosed-tag:<cfif"}},
		{"a string in an expression is no comment", `<cfoutput>#replace(s, "<!--", "")#</cfoutput>`, nil},
		{"a less-than is no tag", "<cfif a lt b>x</cfif> a<b <3 <=", nil},
		{"script is left to the script parse", "<cfscript>\nx = \"<cfif>\"; if ( a ) { y(); }\n</cfscript><cfif a></cfif>", nil},
		{"script and style hold code", "<script>if (a<b) { x = 'c'; }</script><style>a>b{}</style>", nil},
		{"a CF tag in a script block is still CF", "<script>x = '<cfif a>';</script>", []string{"unclosed-tag:<cfif"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := problemsOf(c.src)
			if !slices.Equal(got, c.want) {
				t.Errorf("%q: got %q, want %q", c.src, got, c.want)
			}
		})
	}
}

func TestTagProblemSeverity(t *testing.T) {
	for _, p := range TagProblems("<cfparam name=a name=b><cfif a>") {
		if want := p.Kind == ProblemDuplicateAttribute; p.Warning != want {
			t.Errorf("%s: Warning is %v, want %v", p.Kind, p.Warning, want)
		}
	}
}

// The problems of any text lie within it, whatever the text.
func TestTagProblemsStayWithinTheText(t *testing.T) {
	pieces := []string{
		"<cfif a>", "</cfif>", "<cfloop x=1>", "</cfloop>", "<cfelse>", "<cfcatch>", "<cftry>", "</cftry>",
		"<cffunction name=\"f\">", "<cffunction>", "</cffunction>", "<cfargument name=a>", "<!---", "--->", "<!--", "-->", `"`, "'", "<", ">", "</", "/>",
		"<cfscript>", "</cfscript>", "<script>", "</script>", "#", "<a ", "class=\"<cfif b>", "x", " ", "\n", "é", "<cf_x>", "<cfset a = 1>", "<cfparam name=a name=b>",
	}
	rng := rand.New(rand.NewPCG(5, 11))

	for range 20_000 {
		var b strings.Builder

		for range 1 + rng.IntN(14) {
			b.WriteString(pieces[rng.IntN(len(pieces))])
		}

		src := b.String()

		for _, p := range TagProblems(src) {
			if p.Start < 0 || p.Start >= p.End || p.End > len(src) || p.Message == "" {
				t.Fatalf("%q: problem %+v is outside the text", src, p)
			}
		}
	}
}

func FuzzTagProblems(f *testing.F) {
	for _, s := range []string{"<cfif a>", "</cfif>", `<a class="<cfif b>">`, "<cfparam name=a name=b>", "<cffunction>x"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		for _, p := range TagProblems(s) {
			if p.Start < 0 || p.Start >= p.End || p.End > len(s) {
				t.Fatalf("problem %+v is outside %q", p, s)
			}
		}
	})
}
