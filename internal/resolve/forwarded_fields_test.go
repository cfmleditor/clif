package resolve

import (
	"path/filepath"
	"testing"
)

// TestAFieldSetThroughAPrivateSetterIsWhatInitIsHanded: Preside's services
// take each dependency as an init argument typed by an `@arg.inject id` doc,
// store it through a private _setX( arguments.x ) and read it back through
// _getX(), which returns the field. The setter's argument is untyped, so the
// getter returned nothing and every _getX().m() was a finding: about a
// thousand over Preside. A second write of another kind, a public setter
// (WireBox may call it with anything) or a call the plans cannot read as a
// statement of its own leaves the getter untyped.
func TestAFieldSetThroughAPrivateSetterIsWhatInitIsHanded(t *testing.T) {
	dir := t.TempDir()

	pattern := func(initArg, doc, extra, setterAccess string) string {
		return "component {\n" +
			"\t/**\n\t * " + doc + "\n\t */\n" +
			"\tpublic any function init( required " + initArg + " dep ) {\n\t\t_setDep( arguments.dep );\n\t\treturn this;\n\t}\n" +
			"\tfunction f() {\n\t\t_getDep().work();\n\t}\n" +
			"\tprivate any function _getDep() {\n\t\treturn _dep;\n\t}\n" +
			"\t" + setterAccess + " void function _setDep( required any dep ) {\n\t\t_dep = arguments.dep;\n\t}\n" +
			extra +
			"}\n"
	}

	writeFiles(t, dir, map[string]string{
		"svc/Dep.cfc":          `component { function work() {} }`,
		"svc/Injected.cfc":     pattern("any", "@dep.inject Dep", "", "private"),
		"svc/Declared.cfc":     pattern("svc.Dep", "x", "", "private"),
		"svc/Untyped.cfc":      pattern("any", "x", "", "private"),
		"svc/TwoWrites.cfc":    pattern("any", "@dep.inject Dep", "\tfunction reset() {\n\t\t_dep = \"\";\n\t}\n", "private"),
		"svc/PublicSetter.cfc": pattern("any", "@dep.inject Dep", "", "public"),
		"svc/OtherCall.cfc":    pattern("any", "@dep.inject Dep", "\tfunction g() {\n\t\tvar ok = _setDep( \"x\" );\n\t}\n", "private"),
	})

	untyped := "method '_getDep' has no component return type (chain to 'work')"

	for file, want := range map[string]string{
		"svc/Injected.cfc":     "",
		"svc/Declared.cfc":     "",
		"svc/Untyped.cfc":      untyped,
		"svc/TwoWrites.cfc":    untyped,
		"svc/PublicSetter.cfc": untyped,
		"svc/OtherCall.cfc":    untyped,
	} {
		r := &Resolver{Mappings: map[string]string{"svc": filepath.Join(dir, "svc")}}
		got := reasonsWith(t, r, dir, file)

		if got["_getDep.work"] != want {
			t.Errorf("%s: work: %q, want %q (got %v)", file, got["_getDep.work"], want, got)
		}
	}
}
