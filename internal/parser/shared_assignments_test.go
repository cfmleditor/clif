package parser

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/testutil"
)

// A parse given SharedAssignments answers Assignments with what the caller
// holds, and what AssignmentsOf reads is what the parse's own regions give:
// over every CFML file the repository's fixtures hold, the two agree.
func TestSharedAssignmentsAreTheParsesOwn(t *testing.T) {
	files, shared := 0, 0

	err := filepath.WalkDir(testutil.Testdata(), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		switch strings.ToLower(filepath.Ext(path)) {
		case ".cfc", ".cfm", ".cfml":
		default:
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(b)
		files++

		own := ParseWithOptions(testURI, content, &ParseOptions{}).Assignments()

		calls := 0
		got := ParseWithOptions(testURI, content, &ParseOptions{SharedAssignments: func(text string) []Assignment {
			calls++

			return AssignmentsOf(path, text)
		}}).Assignments()

		shared += calls

		if !reflect.DeepEqual(own, got) {
			t.Errorf("%s: the shared assignments differ from the parse's own: %d, want %d", path, len(got), len(own))
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if files < 20 || shared != files {
		t.Errorf("%d files and %d reads through SharedAssignments: it was not the reader", files, shared)
	}
}
