package resolve

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestACriteriaBuildersClosureAndRestrictionsAreTyped: cborm's when() hands its
// closure the current builder, and a builder's restrictions is a Restrictions.
// The closure rule needs the closure to be when()'s argument on a builder chain.
func TestACriteriaBuildersClosureAndRestrictionsAreTyped(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"svc/Other.cfc": `component { function run(){} }`,
		"svc/Svc.cfc": `component extends="cborm.models.VirtualEntityService" {
function init(){ super.init( entityName = "x" ); return this; }
function onChain(){
	return newCriteria().isTrue( "p" ).when( true, function( c ){
		c.isEq( "a", 1 );
		c.restrictions.like( "a", "b" );
	} );
}
function onLocal(){
	var q = newCriteria();
	q.when( true, function( e ){ e.isEq( "a", 1 ); } );
}
function notABuilder(){
	return thing().when( true, function( d ){ d.isEq( "a", 1 ); } );
}
function notWhen(){
	return newCriteria().list( function( f ){ f.isEq( "a", 1 ); } );
}
function brokenChain(){
	return newCriteria().list().when( true, function( g ){ g.isEq( "a", 1 ); } );
}
function plainRestrictions( x ){
	x.restrictions.like( "a", "b" );
}
function typedButNotABuilder(){
	var o = new svc.Other();
	o.restrictions.like( "a", "b" );
}
}`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "svc/Svc.cfc")

	for _, k := range []string{"c.isEq", "c.restrictions.like", "e.isEq"} {
		if got[k] != "" {
			t.Errorf("%s: %q, want it resolved", k, got[k])
		}
	}

	for k, want := range map[string]string{
		"d.isEq":              "variable 'd' has no component ref",
		"f.isEq":              "variable 'f' has no component ref",
		"g.isEq":              "variable 'g' has no component ref",
		"x.restrictions.like": "variable 'x.restrictions' has no component ref",
		"o.restrictions.like": "variable 'o.restrictions' has no component ref",
	} {
		if got[k] != want {
			t.Errorf("%s: %q, want %q", k, got[k], want)
		}
	}
}

func TestSameLineCallbackContextsStaySeparate(t *testing.T) {
	dir := t.TempDir()
	src := `component extends="cborm.models.VirtualEntityService" {function run(){newCriteria().when(true,function(c){c.isEq("a",1);other(function(c){c.isEq("b",2);});});other(function(c){c.isEq("d",3);});}}`
	writeFiles(t, dir, map[string]string{"Svc.cfc": src})
	r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
	pr := parser.ParseWithOptions(cfpath.ToURI(filepath.Join(dir, "Svc.cfc")), src, &parser.ParseOptions{ExtractCalls: true})
	count := 0

	for _, call := range pr.AllCalls() {
		if call.FuncName != "isEq" {
			continue
		}

		reason := r.CanResolveCall(&call, pr, dir)

		want := strings.HasPrefix(src[call.StartByte:], `isEq("a"`)
		if (reason == "") != want {
			t.Fatalf("%+v: %q, want resolved %v", call, reason, want)
		}

		count++
	}

	if count != 3 {
		t.Fatalf("checked %d", count)
	}
}
