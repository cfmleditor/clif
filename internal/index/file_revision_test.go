package index

import (
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

func TestDiskRevisionFollowsOnlyItsSource(t *testing.T) {
	idx := New()

	a, b := uri.File("/workspace/A.cfc"), uri.File("/workspace/B.cfc")
	if idx.DiskRevision(a) != 0 {
		t.Fatal("missing file has a revision")
	}

	idx.IndexFileFromResult(a, nil, nil)

	first := idx.DiskRevision(a)
	if first == 0 {
		t.Fatal("function-free file has no revision")
	}

	idx.IndexFile(b, `component {function work(){}}`)

	if idx.DiskRevision(a) != first {
		t.Fatal("unrelated file changed the revision")
	}

	idx.SetExtends(b, "Base")

	if idx.DiskRevision(a) != first {
		t.Fatal("unrelated metadata changed the revision")
	}

	idx.IndexParsed(a, "component {}", parser.Parse(a, "component {}"))

	indexed := idx.DiskRevision(a)
	if indexed <= first {
		t.Fatal("parsed source update did not advance the revision")
	}

	idx.IndexBufferFromResult(a, nil, nil)
	idx.ShiftLines(a, 0, 1)

	if idx.DiskRevision(a) != indexed {
		t.Fatal("a buffer edit changed the disk revision")
	}

	idx.RemoveFile(a)

	if idx.DiskRevision(a) != 0 {
		t.Fatal("deleted file kept its revision")
	}

	idx.IndexFile(a, "component {}")

	if idx.DiskRevision(a) <= indexed {
		t.Fatal("recreated file reused its old revision")
	}

	idx.RemoveFilesUnder("file:///workspace")

	if idx.DiskRevision(a) != 0 || idx.DiskRevision(b) != 0 {
		t.Fatal("removed workspace kept file revisions")
	}
}
