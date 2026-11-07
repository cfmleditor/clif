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
