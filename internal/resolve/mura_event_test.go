package resolve

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestMuraHandsItsOwnCodeAnEvent: Mura calls its handlers, renderer and API
// with an event, a servletEvent or a plain one, and says so nowhere in the
// source. Inside Mura's own directory, in a display object and in a plugin's
// event handler, `event` is both classes; a local the function assigns
// something else, and a project file of its own, are left as they were.
func TestMuraHandsItsOwnCodeAnEvent(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"mura/event.cfc":        `component { function getValue( k ) {} }`,
		"mura/servletEvent.cfc": `component { function getValue( k ) {} function getContentRenderer() {} }`,
		"mura/Handler/standardEventsHandler.cfc": `component {
	function onRender( event ) {
		arguments.event.getValue( "x" );
		event.getContentRenderer();
		event.nowhere();
	}
	function loop( events ) {
		var event = events.next();
		event.getDisplayStop();
	}
	function render( renderer ) {
		var event = renderer.getEvent();
		event.getValue( "y" );
	}
}`,
		"mura/plugin/pluginGenericEventHandler.cfc": `component {}`,
		"mura/content/contentRenderer.cfc":          `component { function dspObject() {} }`,
		"plugins/p/eventHandler.cfc": `component extends="mura.plugin.pluginGenericEventHandler" {
	function onSiteRequestStart( event ) { event.getValue( "z" ); }
}`,
		"app/Other.cfc":               `component { function f( event ) { event.getValue( "w" ); } }`,
		"display_objects/gallery.cfm": `<cfoutput>#dspObject()##event.getValue( "v" )#</cfoutput>`,
	})

	displayObject := &Resolver{ImplicitExtends: func(p string) string {
		if strings.Contains(filepath.ToSlash(p), "/display_objects/") && strings.HasSuffix(p, ".cfm") {
			return "mura.content.contentRenderer"
		}

		return ""
	}}

	expectReasons(t, reasonsIn(t, dir, "mura/Handler/standardEventsHandler.cfc"), map[string]string{
		"arguments.event.getValue": "",
		"event.getContentRenderer": "",
		"event.nowhere":            "method 'nowhere' not found in mura.servletEvent|mura.event",
		"event.getDisplayStop":     "variable 'event' has no component ref",
		"event.getValue":           "",
	})
	expectReasons(t, reasonsIn(t, dir, "plugins/p/eventHandler.cfc"), map[string]string{
		"event.getValue": "",
	})
	expectReasons(t, reasonsIn(t, dir, "app/Other.cfc"), map[string]string{
		"event.getValue": "variable 'event' has no component ref",
	})
	expectReasons(t, reasonsWith(t, displayObject, dir, "display_objects/gallery.cfm"), map[string]string{
		"event.getValue": "",
	})
}

// TestAHopWithArgumentsIsNotTheCallWithNone: `$.event()` is the event object
// and `$.event( "requester" )` is a value the event holds, so a resolver for
// the first must not type the second. The hop was matched as `event()`
// whatever it passed, and `$.event( "requester" ).getEmail()` was reported as
// a method missing from the event.
func TestAHopWithArgumentsIsNotTheCallWithNone(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Scope.cfc": `component { function event() {} }`,
		"Evt.cfc":   `component { function getValue( k ) {} }`,
		"Use.cfc": `component {
	function go() {
		var s = new Scope();
		s.event().getValue( "a" );
		s.event().nowhere();
		s.event( "requester" ).getEmail();
	}
}`,
	})

	r := &Resolver{Resolvers: []parser.Resolver{
		{Match: `(?i)(?:^|\.)\s*event\s*\(\s*\)$`, Resolve: "Evt", Prefix: "event"},
	}}

	got := reasonsWith(t, r, dir, "Use.cfc")
	if g := got["s.event.getValue"]; g != "" {
		t.Errorf("event() typed as the event object: %q", g)
	}

	if g := got["s.event.nowhere"]; !strings.Contains(g, "not found in Evt") {
		t.Errorf("a method the event lacks: %q, want it reported against Evt", g)
	}

	if g := got["s.event.getEmail"]; strings.Contains(g, "Evt") {
		t.Errorf("event( \"requester\" ) typed as the event object: %q", g)
	}
}
