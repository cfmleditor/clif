package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestAMuraBeansLoadByReturnsTheBean: Mura documents loadBy() as loading the
// bean and handing it back, and declares it nowhere: contentBean's returns
// what contentManager.read() does, which is the bean it was handed or one from
// a cache. On a component whose chain reaches mura.bean.bean it is the
// receiver, in a chain and through a variable assigned from it. beanORM's
// loadBy takes a returnFormat and is the bean only when the call asks for
// "self" or names none; a component outside Mura with a loadBy of its own is
// left as it was.
func TestAMuraBeansLoadByReturnsTheBean(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"mura/cfobject.cfc":               `component { function getBean( name ) {} }`,
		"mura/bean/bean.cfc":              `component extends="mura.cfobject" { function save() {} }`,
		"mura/bean/beanExtendable.cfc":    `component extends="mura.bean.bean" {}`,
		"mura/content/contentManager.cfc": `component { function read( contentBean ) { return arguments.contentBean; } }`,
		"mura/content/contentBean.cfc": `component extends="mura.bean.beanExtendable" {
	function loadBy() {
		arguments.contentBean = this;
		return variables.contentManager.read( argumentCollection = arguments );
	}
	function hasDrafts() {}
}`,
		"mura/bean/beanORM.cfc": `component extends="mura.bean.bean" {
	function loadBy( returnFormat = "self" ) {
		if ( arguments.returnFormat == "query" ) { return queryNew( "" ); }
		return this;
	}
}`,
		"app/Thing.cfc": `component { function loadBy() { return variables.other.read(); } function go() {} }`,
		"app/views/edit.cfc": `component {
	function show() {
		var content = getBean( "content" ).loadBy( contentID = 1 );
		content.hasDrafts();
		content.nowhere();
		content.loadBy( contentID = 2 ).hasDrafts();
		var entity = getBean( "entity" ).loadBy( id = 1 );
		entity.save();
		entity.nowhere();
		var self = getBean( "entity" ).loadBy( id = 1, returnFormat = "self" );
		self.save();
	}
}`,
	})

	r := &Resolver{Resolvers: []parser.Resolver{
		{Match: `getBean( "content" )`, Resolve: "mura.content.contentBean", Prefix: "getBean"},
		{Match: `getBean( "entity" )`, Resolve: "mura.bean.beanORM", Prefix: "getBean"},
		{Match: `getBean( "thing" )`, Resolve: "app.Thing", Prefix: "getBean"},
	}}

	expectReasons(t, reasonsWith(t, r, dir, "app/views/edit.cfc"), map[string]string{
		"content.hasDrafts":        "",
		"content.nowhere":          "method 'nowhere' not found in contentBean",
		"content.loadBy.hasDrafts": "",
		"entity.save":              "",
		"entity.nowhere":           "method 'nowhere' not found in beanORM",
		"self.save":                "",
	})

	// What the rule declines. Each of these is $any to the parse here, which
	// accepts every call on it, so the rule is asked directly.
	orm := filepath.Join(dir, "mura", "bean", "beanORM.cfc")
	for call, want := range map[string]string{
		`loadBy( id = 1 )`:                         orm,
		`loadBy( id = 1, returnFormat = "self" )`:  orm,
		`loadBy( id = 1, returnFormat = "query" )`: "",
		`loadBy( id = 1, returnFormat = fmt )`:     "",
		`loadBy( argumentCollection = args )`:      "",
		`loadBy( "query" )`:                        "",
		`save()`:                                   "",
	} {
		if got := r.receiverReturn(orm, parser.CallHop(call)); got != want {
			t.Errorf("beanORM %s: %q, want %q", call, got, want)
		}
	}

	if got := r.receiverReturn(orm, "loadBy"); got != "" {
		t.Errorf("beanORM loadBy without its arguments: %q, want none", got)
	}

	if got := r.receiverReturn(filepath.Join(dir, "app", "Thing.cfc"), "loadBy"); got != "" {
		t.Errorf("a loadBy outside Mura: %q, want none", got)
	}
}
