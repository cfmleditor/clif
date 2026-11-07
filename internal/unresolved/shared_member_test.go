package unresolved

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

// A method many components declare, called on a receiver the source does not
// type, is a duck-typed call on one of them. A method few declare, or none, is
// still reported, and the open-world pass keeps all of them.
func TestAMethodManyComponentsDeclareIsAcceptedOnAnUntypedReceiver(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) string {
		f := filepath.Join(dir, name)
		if err := os.WriteFile(f, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		return f
	}

	var files []string

	for i := range 6 {
		files = append(files, write(fmt.Sprintf("Bean%d.cfc", i), "component { function getValue() { return 1; } }"))
	}

	files = append(files,
		write("Rare.cfc", "component { function seldom() { return 1; } }"),
		write("Use.cfc", "component {\n function go( item ) {\n item.getValue();\n item.seldom();\n item.nobodyHas();\n }\n}"),
	)

	got := Scan(vfs.OS{}, files, nil, &Options{WorkspaceFolders: []string{dir}})
	if len(got.Calls) != 2 {
		t.Fatalf("want seldom and nobodyHas reported and getValue accepted, got %+v", got.Calls)
	}

	for _, c := range got.Calls {
		if c.Function == "getValue" {
			t.Fatal(c)
		}
	}

	strict := Scan(vfs.OS{}, files, nil, &Options{WorkspaceFolders: []string{dir}, keepUntypedHops: true})
	if len(strict.Calls) != 3 {
		t.Fatalf("the strict view keeps all three, got %+v", strict.Calls)
	}
}
