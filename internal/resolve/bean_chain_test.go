package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// A bean the configured policy names: the page takes the scope from a
// construction, and the container's getBean( "content" ) is the bean the
// rule spells — the chain past it reads against that bean's own contracts:
//
//	<cfset $ = new mura.MuraScope()>
//	<cfset content = $.getBean("content").loadBy(contentID=rc.contentID)>
//	<cfset ok = content.hasDrafts()>
//
// A method the bean does not declare stays a finding.
func TestTheBeanThePolicyNamesTypesTheChainPastIt(t *testing.T) {
	dir := t.TempDir()

	writeFiles(t, dir, map[string]string{
		"mura/MuraScope.cfc": `<cfcomponent>
	<cffunction name="getBean" access="public" returntype="any">
		<cfargument name="beanName" type="string">
		<cfreturn application.serviceFactory.getBean(arguments.beanName)>
	</cffunction>
</cfcomponent>`,
		"mura/content/contentBean.cfc": `<cfcomponent>
	<cffunction name="loadBy" access="public" returntype="any">
		<cfreturn this>
	</cffunction>

	<cffunction name="hasDrafts" access="public" returntype="boolean">
		<cfreturn false>
	</cffunction>
</cfcomponent>`,
		"page.cfm": `<cfset $ = new mura.MuraScope()>
<cfset content = $.getBean("content").loadBy(contentID=rc.contentID)>
<cfset ok = content.hasDrafts()>
<cfset bad = content.getUndeclared()>`,
	})

	r := &Resolver{Resolvers: []parser.Resolver{
		{Match: `(?i)^\$$`, Resolve: "mura.MuraScope", NameOnly: true},
		{Match: `(?i)(?:^|\.)getBean\(\s*["']content["']\s*\)$`, Resolve: "mura.content.contentBean", Prefix: "getBean"},
	}}

	got := reasonsWith(t, r, dir, "page.cfm")
	expectReasons(t, got, map[string]string{
		"$.getBean":             "",
		"$.getBean.loadBy":      "",
		"content.hasDrafts":     "",
		"content.getUndeclared": "method 'getUndeclared' not found in contentBean",
	})
}
