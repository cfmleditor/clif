package unresolved

import (
	"os"
	"path/filepath"
	"testing"

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
		write("AddressBean.cfc", "component { function getCity() {} }"),
		write("Use.cfc", "component {\n function go( addressBean, oAddressBean, c ) {\n addressBean.getCity();\n oAddressBean.getCity();\n addressBean.getNothing();\n c.getCity();\n }\n}"),
	}

	got := Scan(vfs.OS{}, files, nil, &Options{WorkspaceFolders: []string{dir}})
	if len(got.Calls) != 2 {
		t.Fatalf("want getNothing and the one-letter c reported, got %+v", got.Calls)
	}

	for _, c := range got.Calls {
		if c.Function == "getCity" && c.Variable != "c" {
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
