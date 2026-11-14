package unresolved

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

// A receiver nothing types is the one component of its name when that
// component declares the method; when it does not, the call is reported as it
// was, so the convention never makes a finding of its own.
func TestAReceiverNamedLikeTheOnlyComponentIsThatComponent(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) string {
		f := filepath.Join(dir, name)
		if err := os.WriteFile(f, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		return f
	}

	files := []string{
		write("AddressBean.cfc", "component { function findCity() {} }"),
		write("Use.cfc", "component {\n function go( addressBean, oAddressBean, c ) {\n addressBean.findCity();\n oAddressBean.findCity();\n addressBean.findNothing();\n c.findCity();\n }\n}"),
	}

	got := Scan(vfs.OS{}, files, nil, &Options{WorkspaceFolders: []string{dir}})
	if len(got.Calls) != 2 {
		t.Fatalf("want findNothing and the one-letter c reported, got %+v", got.Calls)
	}

	for _, c := range got.Calls {
		if c.Function == "findCity" && c.Variable != "c" {
			t.Fatal(c)
		}
	}
}

// A method few components declare says which one when the receiver's name
// shares a word with it. Two candidates, a generic word and a method nobody
// declares are not answers.
func TestANameSharingAWordWithTheOnlyDeclarerIsThatComponent(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) string {
		f := filepath.Join(dir, name)
		if err := os.WriteFile(f, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		return f
	}

	files := []string{
		write("Author.cfc", "component { function getEmail() {} }"),
		write("BaseContent.cfc", "component { function fetchAll() {} }"),
		write("PageThing.cfc", "component { function twin() {} }"),
		write("PageOther.cfc", "component { function twin() {} }"),
		write("Use.cfc", "component {\n function go( thisAuthor, content, pageHolder, bean ) {\n thisAuthor.getEmail();\n content.fetchAll();\n pageHolder.twin();\n bean.fetchAll();\n content.nothingDeclaresThis();\n }\n}"),
	}

	got := Scan(vfs.OS{}, files, nil, &Options{WorkspaceFolders: []string{dir}})

	want := map[string]bool{"twin": true, "nothingDeclaresThis": true, "fetchAll": true}

	if len(got.Calls) != 3 {
		t.Fatalf("want the ambiguous, the generic-named and the undeclared call reported, got %+v", got.Calls)
	}

	for _, c := range got.Calls {
		if !want[c.Function] || c.Function == "fetchAll" && c.Variable != "bean" {
			t.Fatal(c)
		}
	}
}

// A native member name a component also declares is the native member on a
// receiver nothing typed, and a distinctive method exactly one component
// declares is that component's. A short or generic name, a distinctive name two
// components declare, and a typed receiver's missing method are still reported.
func TestNativeAndDistinctiveMembersOfAnUntypedReceiver(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) string {
		f := filepath.Join(dir, name)
		if err := os.WriteFile(f, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		return f
	}

	files := []string{
		write("Buffer.cfc", "component { function keyExists() {} function loadSpecialSettings() {} function process() {} function sharedDistinctiveName() {} }"),
		write("Other.cfc", "component { function sharedDistinctiveName() {} }"),
		write("Use.cfc", "component {\n function go( thing ) {\n thing.items.keyExists( 'a' );\n thing.loadSpecialSettings();\n thing.process();\n thing.sharedDistinctiveName();\n }\n}"),
	}

	got := Scan(vfs.OS{}, files, nil, &Options{WorkspaceFolders: []string{dir}})
	if len(got.Calls) != 2 {
		t.Fatalf("want process and sharedDistinctiveName reported, got %+v", got.Calls)
	}

	for _, c := range got.Calls {
		if c.Function != "process" && c.Function != "sharedDistinctiveName" {
			t.Fatal(c)
		}
	}
}

// An accessor-shaped call (verb and a capitalised property) or a Java iterator
// method on a receiver nothing types is a bean or Java call. A name that is no
// accessor (`get` alone, `getter`, `settle`) and a method missing from a typed
// receiver are still reported.
func TestAnAccessorOnAnUntypedReceiverIsABeanCall(t *testing.T) {
	dir := t.TempDir()

	src := "component {\n function go( thing ) {\n var typed = new Known();\n thing.getTitle();\n thing.setType( 1 );\n thing.isLoaded();\n thing.hasNext();\n thing.getter();\n thing.settle();\n typed.getMissing();\n }\n}"
	use := filepath.Join(dir, "Use.cfc")
	known := filepath.Join(dir, "Known.cfc")

	for f, s := range map[string]string{use: src, known: "component { function other() {} }"} {
		if err := os.WriteFile(f, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got := Scan(vfs.OS{}, []string{use, known}, nil, &Options{WorkspaceFolders: []string{dir}})

	want := map[string]bool{"getter": true, "settle": true, "getMissing": true}
	if len(got.Calls) != len(want) {
		t.Fatalf("want %v reported, got %+v", want, got.Calls)
	}

	for _, c := range got.Calls {
		if !want[c.Function] {
			t.Fatal(c)
		}
	}
}

// A method no component declares is accepted on an untyped receiver when a
// native CFML type's contract or a JDK class has it (`entrySet`, `getCause`,
// `toURI`); a name neither knows stays reported.
func TestAJavaOrNativeMethodNoComponentDeclaresIsAcceptedOnAnUntypedReceiver(t *testing.T) {
	dir := t.TempDir()
	use := filepath.Join(dir, "Use.cfc")

	src := "component {\n function go( thing ) {\n thing.entrySet();\n thing.getCause();\n thing.toURI();\n thing.frobnicateWidget();\n }\n}"
	if err := os.WriteFile(use, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	got := Scan(vfs.OS{}, []string{use}, nil, &Options{WorkspaceFolders: []string{dir}})
	if len(got.Calls) != 1 || got.Calls[0].Function != "frobnicateWidget" {
		t.Fatalf("want only frobnicateWidget reported, got %+v", got.Calls)
	}
}

// A function an included template declares is callable from a component the
// include graph never reaches: Masa defines parseXML in a backport template its
// application includes by a computed directory (`#backportdir#lucee.cfm`) and
// its components call it bare. A function declared nowhere is still reported.
func TestABareCallToAFunctionAnIncludedTemplateDeclaresIsAccepted(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) string {
		f := filepath.Join(dir, name)
		if err := os.WriteFile(f, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		return f
	}

	files := []string{
		write("backport.cfm", "<cfscript>\ninclude '#backportdir#lucee.cfm';\n</cfscript>"),
		write("lucee.cfm", "<cfscript>\nfunction parseXML( s ) { return xmlParse( s ); }\n</cfscript>"),
	}
	files = append(files, write("Tool.cfc", "component {\n function go() {\n return parseXML( 'x' ) & nowhereDeclared();\n }\n}"))

	got := Scan(vfs.OS{}, files, files, &Options{WorkspaceFolders: []string{dir}, InferArgs: true})
	if len(got.Calls) != 1 || got.Calls[0].Function != "nowhereDeclared" {
		t.Fatalf("want only nowhereDeclared reported, got %+v", got.Calls)
	}
}

// A UTF-16 component (with its byte-order mark) is analysed as the text it is,
// and a binary file saved as .cfc is skipped, however late its first NUL.
func TestUTF16IsAnalysedAndBinaryIsSkipped(t *testing.T) {
	dir := t.TempDir()

	src := "component {\n function go() {\n return missingFunctionHere();\n }\n}"
	enc := []byte{0xFF, 0xFE}

	for _, r := range src {
		enc = append(enc, byte(r), byte(r>>8))
	}

	utf16 := filepath.Join(dir, "Wide.cfc")
	binary := filepath.Join(dir, "Blob.cfc")

	blob := append(bytes.Repeat([]byte("x"), 2000), 0)
	blob = append(blob, []byte(" function go() { return alsoMissingHere(); }")...)

	for f, b := range map[string][]byte{utf16: enc, binary: blob} {
		if err := os.WriteFile(f, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got := Scan(vfs.OS{}, []string{utf16, binary}, nil, &Options{WorkspaceFolders: []string{dir}})
	if len(got.Calls) != 1 || got.Calls[0].Function != "missingFunctionHere" || got.Calls[0].Line != 2 {
		t.Fatalf("want the UTF-16 file's one call on its third line and nothing from the binary, got %+v", got.Calls)
	}
}

// The same call from a component that extends one is judged against its
// extends chain, which does not hold the name either: Masa's components
// extend mura.cfobject, whose include pulls the backport in.
func TestABareCallInAComponentWithABaseToATemplateFunctionIsAccepted(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) string {
		f := filepath.Join(dir, name)
		if err := os.WriteFile(f, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		return f
	}

	files := []string{
		write("backport.cfm", "<cfscript>\ninclude '#backportdir#lucee.cfm';\n</cfscript>"),
		write("lucee.cfm", "<cfscript>\nfunction parseXML( s ) { return xmlParse( s ); }\n</cfscript>"),
		write("Base.cfc", "component {\n function init() { return this; }\n}"),
	}
	files = append(files, write("Tool.cfc", "component extends=\"Base\" {\n function go() {\n return parseXML( 'x' ) & nowhereDeclared();\n }\n}"))

	got := Scan(vfs.OS{}, files, files, &Options{WorkspaceFolders: []string{dir}, InferArgs: true})
	if len(got.Calls) != 1 || got.Calls[0].Function != "nowhereDeclared" {
		t.Fatalf("want only nowhereDeclared reported, got %+v", got.Calls)
	}
}

// A string that is only part of a call's first argument names nothing:
// `getBean( url.p & "Provider" )` was read as `getBean( "Provider" )`, which
// typed the chain as a component called Provider and reported a method missing
// from it. A whole-string name still types the call, and an unknown one is
// still reported.
func TestAStringThatIsPartOfAnArgumentIsNotTheBeanName(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) string {
		f := filepath.Join(dir, name)
		if err := os.WriteFile(f, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		return f
	}

	files := []string{
		write("Real.cfc", "component {\n function run() {}\n}"),
		write("Use.cfc", "component {\n function go() {\n  var a = getBean( 'Real' ).run();\n  var b = getBean( 'Gone' ).missing();\n  var c = getBean( url.p & 'Frag' ).fragCall();\n  var d = getBean( 'Frag2' & url.p ).frag2Call();\n }\n}"),
	}

	resolvers := []parser.Resolver{{
		Match:   `(?i)(?:^|\.)getBean\(\s*(?:\w+\s*[=:]\s*)?["']([A-Za-z_][\w.]*)["']\s*(?:,[^()]*)?\)$`,
		Resolve: "$1", Prefix: "getBean",
	}}

	got := Scan(vfs.OS{}, files, files, &Options{Settings: resolve.Settings{Resolvers: resolvers, WorkspaceFolders: []string{dir}}, InferArgs: true})
	for _, c := range got.Calls {
		if strings.Contains(c.Reason, "Frag") {
			t.Errorf("a fragment of an argument typed the call: %+v", c)
		}
	}
}

// A `new` inside a call's arguments is the receiver of the hops chained on it,
// as it is standing alone: `expect( new Real().present() )` is a call to
// Real.present, not a bare call to a function of this file, and a method Real
// lacks is reported against it.
func TestAChainOnANewInsideCallArgumentsIsOnTheNewComponent(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) string {
		f := filepath.Join(dir, name)
		if err := os.WriteFile(f, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		return f
	}

	files := []string{
		write("Real.cfc", "component {\n function present() { return this; }\n function again() {}\n}"),
		write("Use.cfc", "component {\n function go() {\n  check( new Real().present().again() );\n  check( new Real().absent() );\n }\n function check( v ) {}\n}"),
	}

	got := Scan(vfs.OS{}, files, files, &Options{Settings: resolve.Settings{WorkspaceFolders: []string{dir}}, InferArgs: true})

	var reported []string
	for _, c := range got.Calls {
		reported = append(reported, c.Function+": "+c.Reason)
	}

	if len(reported) != 1 || !strings.HasPrefix(reported[0], "absent:") || !strings.Contains(reported[0], "Real") {
		t.Fatalf("want only absent reported against Real, got %q", reported)
	}
}

// A static call, `X::m()`, is read as `new X()` reads its path: an import says
// what a bare name is (a quoted one brings in a Java class), `java:` and the JDK's
// own packages name a Java class, and one in an argument list is the same call.
// What a static call returns, or a static field holds, has no type here, so a
// method chained on it is not a bare call. A component that is not there, and a
// method a component lacks, are still reported.
func TestAStaticCallIsReadAsANewPathIs(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) string {
		f := filepath.Join(dir, name)
		if err := os.WriteFile(f, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		return f
	}

	files := []string{
		write("Real.cfc", "component {\n static function present() {}\n}"),
		write("Use.cfc", `component {
	import "com.q.Quoted";
	function go() {
		Quoted::fromImport();
		java.lang.Double::valueOf( "1" );
		check( java:Double::valueOf( "1" ) );
		check( Quoted::make().chained() );
		check( Elsewhere::FIELD.chainedField() );
		Real::present();
		check( Real::present() );
		Real::absent();
		check( Gone::vanished() );
	}
	function check( v ) {}
}`),
	}

	got := Scan(vfs.OS{}, files, files, &Options{Settings: resolve.Settings{WorkspaceFolders: []string{dir}}, InferArgs: true})

	var reported []string
	for _, c := range got.Calls {
		reported = append(reported, c.Function)
	}

	slices.Sort(reported)

	if want := []string{"absent", "vanished"}; !slices.Equal(reported, want) {
		t.Fatalf("reported %v, want %v; %+v", reported, want, got.Calls)
	}
}
