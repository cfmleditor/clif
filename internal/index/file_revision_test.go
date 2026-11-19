package index

import (
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

func TestFileRevisionFollowsOnlyItsSource(t *testing.T) {
	idx := New()

	a, b := uri.File("/workspace/A.cfc"), uri.File("/workspace/B.cfc")
	if idx.FileRevision(a) != 0 {
		t.Fatal("missing file has a revision")
	}

	idx.IndexFileFromResult(a, nil, nil)

	first := idx.FileRevision(a)
	if first == 0 {
		t.Fatal("function-free file has no revision")
	}

	idx.IndexFile(b, `component {function work(){}}`)

	if idx.FileRevision(a) != first {
		t.Fatal("unrelated file changed the revision")
	}

	idx.SetExtends(b, "Base")

	if idx.FileRevision(a) != first {
		t.Fatal("unrelated metadata changed the revision")
	}

	idx.IndexParsed(a, "component {}", parser.Parse(a, "component {}"))

	indexed := idx.FileRevision(a)
	if indexed <= first {
		t.Fatal("parsed source update did not advance the revision")
	}

	idx.ShiftLines(a, 0, 1)

	shifted := idx.FileRevision(a)
	if shifted <= indexed {
		t.Fatal("line edit did not advance the revision")
	}

	idx.RemoveFile(a)

	if idx.FileRevision(a) != 0 {
		t.Fatal("deleted file kept its revision")
	}

	idx.IndexFile(a, "component {}")

	if idx.FileRevision(a) <= shifted {
		t.Fatal("recreated file reused its old revision")
	}

	idx.RemoveFilesUnder("file:///workspace")

	if idx.FileRevision(a) != 0 || idx.FileRevision(b) != 0 {
		t.Fatal("removed workspace kept file revisions")
	}
}
