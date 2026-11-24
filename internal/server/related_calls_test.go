package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
)

// relatedCallsWorkspace writes an extends family (Animal, Dog, Puppy, Cat) and
// an unrelated component that all declare or inherit speak(), and a consumer
// calling it on a script local of each type. It returns the server, and the
// directory.
func relatedCallsWorkspace(t *testing.T) (*Server, string, map[string]string) {
	t.Helper()

	dir := t.TempDir()
	speak := "\tpublic string function speak() { return \"\"; }\n"
	files := map[string]string{
		"Animal.cfc":    "component {\n" + speak + "}\n",
		"Dog.cfc":       "component extends=\"Animal\" {\n" + speak + "}\n",
		"Puppy.cfc":     "component extends=\"Dog\" {\n}\n",
		"Cat.cfc":       "component extends=\"Animal\" {\n" + speak + "}\n",
		"Unrelated.cfc": "component {\n" + speak + "}\n",
	}

	var consumer strings.Builder

	consumer.WriteString("component {\n")

	for _, typ := range []string{"Animal", "Dog", "Puppy", "Cat", "Unrelated"} {
		consumer.WriteString("\tpublic void function via" + typ + "() {\n\t\tvar x = new " + typ + "();\n\t\tx.speak();\n\t}\n")
	}

	files["Consumer.cfc"] = consumer.String() + "}\n"

	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	srv := newTestServer()
	srv.References = true
	srv.WorkspaceFolders = []string{dir}
	srv.indexWorkspace()

	return srv, dir, files
}

// Call hierarchy and the code lens answer the question references answer:
// a call typed as a subtype of the method's component reaches it, one typed as
// an unrelated component with a method of the same name does not.
func TestIncomingCallsFollowExtendsAndTypeScriptReceivers(t *testing.T) {
	srv, dir, files := relatedCallsWorkspace(t)

	animalURI := cfpath.ToURI(filepath.Join(dir, "Animal.cfc"))
	openDoc(t, srv, animalURI, files["Animal.cfc"])

	items := prepareCallHierarchy(t, srv, animalURI, protocol.Position{Line: 1, Character: 26})
	if len(items) != 1 {
		t.Fatalf("prepareCallHierarchy = %+v, want 1 item", items)
	}

	var got []string
	for _, in := range incomingCalls(t, srv, &items[0]) {
		got = append(got, in.From.Name)
	}

	want := []string{"viaAnimal", "viaCat", "viaDog", "viaPuppy"}
	if len(got) != len(want) || !hasAll(got, want) {
		t.Errorf("incoming callers = %v, want %v (and not viaUnrelated)", got, want)
	}
}

func TestCodeLensCountsRelatedCalls(t *testing.T) {
	srv, dir, files := relatedCallsWorkspace(t)

	dogURI := cfpath.ToURI(filepath.Join(dir, "Dog.cfc"))
	openDoc(t, srv, dogURI, files["Dog.cfc"])

	res, err := srv.handleCodeLens(context.Background(), makeCall(t, protocol.MethodTextDocumentCodeLens, protocol.CodeLensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: dogURI},
	}))
	if err != nil {
		t.Fatalf("handleCodeLens: %v", err)
	}

	lenses, _ := res.([]protocol.CodeLens)
	if len(lenses) != 1 {
		t.Fatalf("lenses = %+v, want one for speak", lenses)
	}

	// Dog.speak is reached by calls typed Dog, Puppy (below) and Animal (above):
	// not Cat (a sibling) nor Unrelated.
	if got := lenses[0].Command.Title; got != "3 usages" {
		t.Errorf("lens = %q, want \"3 usages\"", got)
	}
}
