package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/parser"
)

func cfmlFilesIn(t *testing.T, dir string) []string {
	t.Helper()

	var out []string

	err := filepath.WalkDir(dir, func(p string, _ os.DirEntry, err error) error {
		if err == nil && (strings.HasSuffix(p, ".cfc") || strings.HasSuffix(p, ".cfm")) {
			out = append(out, p)
		}

		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	return out
}

// TestAnUntypedArgumentHoldsWhatEveryCallerPasses: an argument with no type is
// the component every call of its function passes, as alternatives when they
// differ. A caller that omits it says nothing; one whose receiver or expression
// cannot be typed leaves it untyped. Without the files to search it is off.
func TestAnUntypedArgumentHoldsWhatEveryCallerPasses(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"models/Item.cfc":  `component { function useItM(){} function mixedItem(){} function openM(){} function skipM(){} function inheritedM(){} function forwardedM(){} function conditionalM(){} function siblingM(){} function afterM(){} }`,
		"models/Other.cfc": `component { function mixedOther(){} }`,
		"svc/Base.cfc":     `component { function inherited( required thing ){ arguments.thing.inheritedM(); } }`,
		"svc/Svc.cfc": `component extends="Base" {
function useIt( required thing ){ arguments.thing.useItM(); arguments.thing.nope(); }
function useMixed( required thing ){ arguments.thing.mixedItem(); arguments.thing.mixedOther(); }
function useOpen( required thing ){ arguments.thing.openM(); }
function useSkip( required thing, extra ){ arguments.thing.skipM(); }
function useForwarded( required thing ){ arguments.thing.forwardedM(); }
function useConditional( required thing ){ arguments.thing.conditionalM(); }
function useSibling( required thing ){ arguments.thing.siblingM(); }
function useAfter( required thing ){ arguments.thing.afterM(); }
function callers(){
	var a = new models.Item();
	var forwarded = a;
	useIt( a );
	useForwarded( forwarded );
	if (runtime()) {
		var conditional = a;
	}
	useConditional( conditional );
	if (first()) {
		var sibling = a;
	}
	if (second()) {
		useSibling( sibling );
	}
	useAfter( after );
	var after = a;
	useIt(
		thing = new models.Item()
	);
	useMixed( new models.Item() );
	useMixed( thing = new models.Other() );
	useSkip( a );
	useSkip();
	useOpen( a );
	super.inherited( a );
}
}`,
		"callers/Elsewhere.cfc": `component { function go( x ){ x.useOpen( new models.Other() ); } }`,
	})

	got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "svc/Svc.cfc")

	for _, k := range []string{"arguments.thing.useItM", "arguments.thing.mixedItem", "arguments.thing.mixedOther", "arguments.thing.skipM", "arguments.thing.forwardedM"} {
		if got[k] != "" {
			t.Errorf("%s: %q, want it resolved", k, got[k])
		}
	}

	for _, k := range []string{"arguments.thing.conditionalM", "arguments.thing.siblingM", "arguments.thing.afterM"} {
		if !strings.Contains(got[k], "no component ref") {
			t.Errorf("%s was typed without a reaching straight-line assignment: %q", k, got[k])
		}
	}

	if !strings.Contains(got["arguments.thing.nope"], "method 'nope' not found in") {
		t.Errorf("useIt's argument is not an Item: %q", got["arguments.thing.nope"])
	}

	if !strings.Contains(got["arguments.thing.openM"], "no component ref") {
		t.Errorf("useOpen was typed despite a caller on an untyped receiver: %q", got["arguments.thing.openM"])
	}

	inherited := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "svc/Base.cfc")
	if inherited["arguments.thing.inheritedM"] != "" {
		t.Errorf("a super. call is not a caller: %q", inherited["arguments.thing.inheritedM"])
	}

	off := reasonsWith(t, &Resolver{}, dir, "svc/Svc.cfc")
	if !strings.Contains(off["arguments.thing.useItM"], "no component ref") {
		t.Errorf("inference ran with no files to search: %q", off["arguments.thing.useItM"])
	}
}

func TestCallerAliasesRequireCompleteStraightLineProof(t *testing.T) {
	for _, tc := range []struct {
		name, body, padding string
	}{
		{"unbraced if", "if (runtime())\nvar forwarded = a;\nuseForwarded(forwarded);", ""},
		{"unbraced loop", "while (runtime())\nvar forwarded = a;\nuseForwarded(forwarded);", ""},
		{"unbraced else", "if (runtime()) {} else\nvar forwarded = a;\nuseForwarded(forwarded);", ""},
		{"exhausted sibling paths", "if (first()) {\nvar forwarded = a;\n}\nif (second()) {\nuseForwarded(forwarded);\n}", strings.Repeat("work();\n", 1100)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"Item.cfc": `component { function ready(){} }`,
				"Svc.cfc": `component {
function useForwarded(thing) { arguments.thing.ready(); }
function callers() {
var a = new Item();
` + tc.body + "\n" + tc.padding + `
}
}`,
			})

			got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Svc.cfc")
			if reason, exists := got["arguments.thing.ready"]; !exists || !strings.Contains(reason, "no component ref") {
				t.Fatalf("alias resolved without complete straight-line proof: %v", got)
			}
		})
	}
}

func TestCallerAliasesAreNotLimitedByTheRestOfTheFile(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Item.cfc": `component { function ready(){} }`,
		"Svc.cfc": `component {
function unrelated() {
` + strings.Repeat("work();\n", 1100) + `}
function useForwarded(thing) { arguments.thing.ready(); }
function callers() {
var a = new Item();
var forwarded = a;
useForwarded(forwarded);
}
}`,
	})

	got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Svc.cfc")
	if reason := got["arguments.thing.ready"]; reason != "" {
		t.Fatalf("a large function elsewhere in the file withheld the alias: %q", reason)
	}
}

// TestACallerPlacedByTheParseIsACaller: Masa's
// `$.getBean( "userManager" ).update( … )` has an untyped variable and a
// receiver the parse already resolved through the getBean resolver, as a
// chained `new Svc()` has none at all. Reading only the variable left such a
// caller unplaced, which withheld the argument for every function sharing
// its name; a caller placed on another component was dismissed although it
// is a call of this one.
func TestACallerPlacedByTheParseIsACaller(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Item.cfc": `component { function ready(){} }`,
		"Svc.cfc":  `component { function use( thing ){ arguments.thing.ready(); } }`,
		"A.cfc":    `component { function run(){ $.getBean( "Svc" ).use( new Item() ); } }`,
		"B.cfc":    `component { function run(){ new Svc().use( new Item() ); } }`,
	})

	getBean := func() []parser.Resolver {
		return []parser.Resolver{{Match: `(?i)(?:^|\.)getBean\(\s*["']([A-Za-z_][\w.]*)["']\s*\)$`, Resolve: "$1", Prefix: "getBean"}}
	}

	got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir), Resolvers: getBean()}, dir, "Svc.cfc")
	expectReasons(t, got, map[string]string{"arguments.thing.ready": ""})

	// The same caller passing something untyped still withholds it.
	writeFiles(t, dir, map[string]string{"C.cfc": `component { function run( x ){ $.getBean( "Svc" ).use( x ); } }`})

	got = reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir), Resolvers: getBean()}, dir, "Svc.cfc")
	if !strings.Contains(got["arguments.thing.ready"], "no component ref") {
		t.Errorf("a placed caller passing an untyped value was ignored: %q", got["arguments.thing.ready"])
	}
}

// TestADollarNamedFunctionsCallersAreFound: Wheels names its internals with a
// leading $, and the caller index read `$build(` as a call to build, so such a
// function's untyped argument was never typed by what its callers pass.
func TestADollarNamedFunctionsCallersAreFound(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Item.cfc": `component { function go(){} }`,
		"Svc.cfc": `component {
function $use( required thing ){ arguments.thing.go(); arguments.thing.nope(); }
function callers(){ $use( new Item() ); }
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Svc.cfc"), map[string]string{
		"arguments.thing.go":   "",
		"arguments.thing.nope": "method 'nope' not found in Item",
		"$use":                 "",
	})
}

// TestACallOnWhatCannotBeTheComponentIsNotACaller: Mura's settingsDAO.update(bean)
// shares its name with MessageDigest's md.update() and with a plugin's
// pluginCFC.update(), and neither receiver could be typed, so the DAO's
// argument was never inferred. A Java object is never a component, and a
// component built from a computed path ending in a literal file name is that
// file — unless a file of that name extends the declaring component.
func TestACallOnWhatCannotBeTheComponentIsNotACaller(t *testing.T) {
	files := map[string]string{
		"Item.cfc": `component { function go(){} }`,
		"DAO.cfc":  `component { function update( required bean ){ arguments.bean.go(); } }`,
		"Mgr.cfc": `component {
function save(){ var d = new DAO(); d.update( new Item() ); }
function sha(){
	var md = createObject("java", "java.security.MessageDigest").getInstance("SHA-1");
	md.update( 1 );
}
function plugins( dir ){
	var p = createObject("component", "plugins.#dir#.plugin");
	p.update( 1 );
}
}`,
	}

	dir := t.TempDir()
	writeFiles(t, dir, files)
	expectReasons(t, reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "DAO.cfc"), map[string]string{
		"arguments.bean.go": "",
	})

	// A plugin.cfc that extends the DAO could be the receiver.
	files["plugins/x/plugin.cfc"] = `component extends="DAO" {}`

	dir = t.TempDir()
	writeFiles(t, dir, files)

	if got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "DAO.cfc")["arguments.bean.go"]; got == "" {
		t.Errorf("typed although a plugin.cfc extending the DAO may be the receiver")
	}
}

// TestAnArgumentPassedAsThisIsTheCallersComponent: Mura's contentRenderer
// hands itself to its utility, `utility.f( this )`, and the utility's
// `arguments.renderer` was untyped. `this` is the calling component, or a
// component extending it, as a theme's renderer does.
func TestAnArgumentPassedAsThisIsTheCallersComponent(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Renderer.cfc":      `component { function own(){} function go(){ new Utility().f( this ); } }`,
		"ThemeRenderer.cfc": `component extends="Renderer" { function themed(){} }`,
		"Utility.cfc":       `component { function f( required renderer ){ arguments.renderer.own(); arguments.renderer.themed(); arguments.renderer.nope(); } }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Utility.cfc"), map[string]string{
		"arguments.renderer.own":    "",
		"arguments.renderer.themed": "",
	})

	if got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Utility.cfc")["arguments.renderer.nope"]; !strings.Contains(got, "not found") {
		t.Errorf("arguments.renderer.nope: %q, want a method not found", got)
	}
}

// TestACallerTypedByAResolverIsPlaced: Mura's admin pages call
// `$.dspObjects( … )`, and `$` is a MuraScope only by the mura preset's
// resolver on the variable name. The MuraScope has no dspObjects of its own —
// its onMissingMethod hands the call to the renderer — so the page does not
// call the utility's dspObjects, whose renderer argument is then what its one
// real caller passes. Placing the receiver by the receiver lookup alone left
// the page's call unplaceable, and the argument untyped.
func TestACallerTypedByAResolverIsPlaced(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"mura/MuraScope.cfc": `component { function onMissingMethod( name, args ) {} }`,
		"mura/Renderer.cfc": `component {
	variables.util = new Util();
	function dspObjects( columnid ) {
		arguments.renderer = this;
		return variables.util.dspObjects( argumentCollection = arguments );
	}
	function own() {}
}`,
		"mura/Util.cfc": `component {
	function dspObjects( renderer ) { arguments.renderer.own(); }
}`,
		"admin/page.cfm": `<cfoutput>#$.dspObjects( columnid = 1 )#</cfoutput>`,
	})

	r := &Resolver{
		InferArgsFiles: cfmlFilesIn(t, dir),
		Resolvers:      []parser.Resolver{{Match: "$", Resolve: "mura.MuraScope", Prefix: "$"}},
	}

	expectReasons(t, reasonsWith(t, r, dir, "mura/Util.cfc"), map[string]string{
		"arguments.renderer.own": "",
	})
}

// TestADeclarationIsNotTheCallOnItsLine: `function dspObjects() {
// variables.util.dspObjects( renderer = this ); }` written on one line holds
// the function's name twice, and reading the first as the call found no
// argument there, so the caller passed nothing.
func TestADeclarationIsNotTheCallOnItsLine(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Renderer.cfc": `component {
	variables.util = new Util();
	function dspObjects() { variables.util.dspObjects( renderer = this ); }
	function own() {}
}`,
		"Util.cfc": `component { function dspObjects( renderer ) { arguments.renderer.own(); } }`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Util.cfc"), map[string]string{
		"arguments.renderer.own": "",
	})
}

// Common names must spend the caller budget on their resolved owner, not on
// unrelated methods or lexical matches. Unknown receivers still veto inference.
func TestArgumentCallerBudgetUsesOwnership(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprint("unknown=", unknown), func(t *testing.T) {
			dir := t.TempDir()

			files := map[string]string{
				"Item.cfc":   `component { function ready(){} }`,
				"DAO.cfc":    `component { function update(bean){ arguments.bean.ready(); arguments.bean.missing(); } }`,
				"Caller.cfc": `component { function run(){ new DAO().update(new Item()); } }`,
				"Other.cfc":  `component { function update(bean){} }`,
			}
			for i := range 61 {
				files[fmt.Sprintf("noise/N%d.cfc", i)] = `component { function run(){ new Other().update("not a bean"); } }`
			}

			if unknown {
				files["Unknown.cfc"] = `component { function run(x){ x.update(new Item()); } }`
			}

			writeFiles(t, dir, files)

			got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir), CallerInferenceLimits: config.CallerInferenceLimits{MaxCallerFiles: 1}}, dir, "DAO.cfc")
			if unknown {
				if !strings.Contains(got["arguments.bean.ready"], "no component ref") {
					t.Fatalf("unknown caller was ignored: %v", got)
				}
			} else {
				expectReasons(t, got, map[string]string{"arguments.bean.ready": "", "arguments.bean.missing": "method 'missing' not found in Item"})
			}
		})
	}
}

func TestArgumentInferenceBeyondFormerLimits(t *testing.T) {
	for _, kind := range []string{"candidates", "owners", "sites", "constructors", "late unknown"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			files := map[string]string{
				"Item.cfc":   `component { function ready(){} }`,
				"DAO.cfc":    `component { function update(bean){ arguments.bean.ready(); } }`,
				"Caller.cfc": `component { function run(){ new DAO().update(new Item()); } }`,
			}

			switch kind {
			case "candidates":
				for i := range 513 {
					files[fmt.Sprintf("noise/N%d.cfc", i)] = `component { /* update( */ }`
				}
			case "owners":
				for i := range 61 {
					files[fmt.Sprintf("noise/N%d.cfc", i)] = `component { function run(){ new DAO().update(); } }`
				}
			case "sites":
				files["Caller.cfc"] = `component { function run(){` + strings.Repeat(`new DAO().update(new Item());`, 41) + `} }`
			case "constructors":
				files["DAO.cfc"] = `component { function init(bean){ arguments.bean.ready(); arguments.bean.missing(); } }`

				files["Caller.cfc"] = `component { function run(){ new DAO(new Item()); } }`
				for i := range 65 {
					files[fmt.Sprintf("noise/N%d.cfc", i)] = `component { function run(){ new DAO(new Item()); } }`
				}
			case "late unknown":
				files["Caller.cfc"] = `component { function run(x){` + strings.Repeat(`new DAO().update(new Item());`, 41) + `new DAO().update(x); } }`
			}

			writeFiles(t, dir, files)

			got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "DAO.cfc")
			if kind == "late unknown" {
				if !strings.Contains(got["arguments.bean.ready"], "no component ref") {
					t.Fatalf("late unknown value was ignored: %v", got)
				}
			} else {
				if got["arguments.bean.ready"] != "" {
					t.Fatalf("%s still prevents inference: %v", kind, got)
				}

				if kind == "constructors" && !strings.Contains(got["arguments.bean.missing"], "not found") {
					t.Fatalf("constructor inference did not check missing method: %v", got)
				}
			}
		})
	}
}

func TestOptionalCallerInferenceLimits(t *testing.T) {
	for _, kind := range []string{"candidates", "owners", "sites", "constructors", "constructor sites", "unlimited"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			files := map[string]string{
				"Item.cfc":   `component { function ready(){} }`,
				"DAO.cfc":    `component { function update(bean){ arguments.bean.ready(); } }`,
				"Caller.cfc": `component { function run(){ new DAO().update(new Item()); new DAO().update(new Item()); } }`,
				"Other.cfc":  `component { /* update( */ }`,
			}
			limits := config.CallerInferenceLimits{}

			switch kind {
			case "candidates":
				limits.MaxCandidateFiles = 1
			case "owners":
				limits.MaxCallerFiles = 1
				files["Second.cfc"] = `component { function run(){ new DAO().update(new Item()); } }`
			case "sites":
				limits.MaxCallSites = 1
			case "constructors", "constructor sites":
				files["DAO.cfc"] = `component { function init(bean){ arguments.bean.ready(); } }`
				files["Caller.cfc"] = `component { function run(){ new DAO(new Item()); new DAO(new Item()); } }`
				files["Second.cfc"] = `component { function run(){ new DAO(new Item()); } }`

				if kind == "constructors" {
					limits.MaxConstructorFiles = 1
				} else {
					limits.MaxCallSites = 1
				}
			case "unlimited":
				limits = config.CallerInferenceLimits{MaxCandidateFiles: -1, MaxCallerFiles: -1, MaxCallSites: -1, MaxConstructorFiles: -1}
			}

			writeFiles(t, dir, files)

			got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir), CallerInferenceLimits: limits}, dir, "DAO.cfc")["arguments.bean.ready"]
			if kind == "unlimited" {
				if got != "" {
					t.Fatalf("nonpositive limits prevented inference: %s", got)
				}
			} else if !strings.Contains(got, "no component ref") {
				t.Fatalf("%s limit did not withhold inference: %s", kind, got)
			}
		})
	}
}

func TestUnknownReceiverWithoutTheArgumentDoesNotBlockInference(t *testing.T) {
	for _, passed := range []bool{false, true} {
		t.Run(fmt.Sprint("passed=", passed), func(t *testing.T) {
			dir := t.TempDir()

			unknown := `x.use();`
			if passed {
				unknown = `x.use(argumentCollection=runtimeValues);`
			}

			writeFiles(t, dir, map[string]string{
				"Item.cfc":    `component {function ready(){}}`,
				"Service.cfc": `component {function use(bean){arguments.bean.ready();arguments.bean.missing();}}`,
				"Caller.cfc":  `component {function run(x){new Service().use(new Item());` + unknown + `}}`,
			})

			got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Service.cfc")
			if passed {
				if !strings.Contains(got["arguments.bean.ready"], "no component ref") {
					t.Fatalf("unknown collection did not block: %v", got)
				}
			} else {
				expectReasons(t, got, map[string]string{"arguments.bean.ready": "", "arguments.bean.missing": "method 'missing' not found in Item"})
			}
		})
	}
}

func TestCallerInferenceDoesNotTreatRuntimeComponentNameAsLiteral(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Service.cfc": `component {function use(target){arguments.target.ready();}}`,
		"Caller.cfc": `component {function make(name){
   try {var bundle=createObject("component","#arguments.name#");} catch(any e){rethrow;}
   return bundle;
  } function run(name,x){
   var target=make(name);
   var service=new Service();
   service.use(target);
   x.use();
  }}`,
	})

	got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Service.cfc")["arguments.target.ready"]
	if !strings.Contains(got, "no component ref") {
		t.Fatalf("runtime name was inferred as a literal component: %s", got)
	}
}

func TestChainedUnknownCallOmittingArgumentDoesNotVeto(t *testing.T) {
	for _, supplied := range []bool{false, true} {
		t.Run(strconv.FormatBool(supplied), func(t *testing.T) {
			dir := t.TempDir()

			args := ""
			if supplied {
				args = "bean=unknownValue"
			}

			writeFiles(t, dir, map[string]string{
				"Item.cfc":    `component {function ready(){}}`,
				"Service.cfc": `component {function use(bean){arguments.bean.ready();arguments.bean.missing();}}`,
				"Caller.cfc":  `component {function run(x){new Service().use(new Item());x.getStats().setLockID(1).use(` + args + `);}}`,
			})

			got := reasonsWith(t, &Resolver{InferArgsFiles: cfmlFilesIn(t, dir)}, dir, "Service.cfc")
			if supplied {
				if !strings.Contains(got["arguments.bean.ready"], "no component ref") {
					t.Fatalf("unknown supplied chain was ignored: %v", got)
				}
			} else {
				expectReasons(t, got, map[string]string{"arguments.bean.ready": "", "arguments.bean.missing": "method 'missing' not found in Item"})
			}
		})
	}
}
