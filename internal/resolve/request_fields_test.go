package resolve

import (
	"regexp"
	"strings"
	"testing"
)

// requestWriteReReference is the regular expression requestWriteCounts
// replaced, kept as the reference the scanner is held to.
func requestWriteReReference(name string) *regexp.Regexp {
	q := regexp.QuoteMeta(name)

	return regexp.MustCompile(`(?i)\brequest\s*\.\s*` + q + `\s*=[^=]` +
		`|\brequest\s*\[\s*["']` + q + `["']\s*\]\s*=[^=]` +
		`|structInsert\s*\(\s*request\s*,\s*["']` + q + `["']` +
		`|name\s*=\s*["']request\.` + q + `["']`)
}

// TestRequestWriteCountsMatchTheRegexItReplaced: one pass per write form over
// a file replaced a regex compiled and run once per request key; they must
// count the same writes.
func TestRequestWriteCountsMatchTheRegexItReplaced(t *testing.T) {
	content := `<cfset request.currentUser = createObject("component", "user")>
<cfset REQUEST . CurrentUser=x>
<cfif request.currentUser == y></cfif>
<cfset request.currentUserX = 1>
<cfset myrequest.currentUser = 2>
<cfset request.a.b = 3>
<cfset request[ "siteID" ] = 4>
<cfset request['siteid']=5>
<cfset request["siteid"] == 6>
<cfset structInsert( request, "flash", {} )>
<cfset StructInsert(request,'Flash',{})>
<cfparam name="request.layout" default="">
<cfparam NAME = 'request.Layout' default="">
<cfset x = request.currentUser>
`
	counts := requestWriteCounts(content)

	for _, name := range []string{"currentUser", "currentUserX", "a", "siteID", "flash", "layout", "missing"} {
		want := len(requestWriteReReference(name).FindAllStringIndex(content, -1))
		if got := counts[strings.ToLower(name)]; got != want {
			t.Errorf("request.%s: %d writes, the regex finds %d", name, got, want)
		}
	}
}
