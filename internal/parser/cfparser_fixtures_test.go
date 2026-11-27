package parser_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/language"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// TestCFParserFixtureReplay parses every .cfc/.cfm fixture in a local checkout
// of cfmleditor/cfparser when CFPARSER_TESTS_DIR is set, reporting how many
// files parse cleanly with tree-sitter-cfml. Skipped when unset.
//
// Usage:
//
//	CFPARSER_TESTS_DIR=/path/to/cfparser/cfml.parsing/src/test/resources/cfml \
//	  go test -v -run TestCFParserFixtureReplay ./internal/parser/...
func TestCFParserFixtureReplay(t *testing.T) {
	root := strings.TrimSpace(os.Getenv("CFPARSER_TESTS_DIR"))
	if root == "" {
		t.Skip("set CFPARSER_TESTS_DIR to replay cfmleditor/cfparser test fixtures")
	}

	var total, clean int

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !cfpath.IsCFMLFile(path) {
			return walkErr
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		total++

		if !hasSyntaxError(data) {
			clean++
		} else {
			rel, _ := filepath.Rel(root, path)
			t.Logf("PARSE_ERROR %s", rel)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s): %v", root, err)
	}

	t.Logf("cfmleditor/cfparser fixtures: %d/%d parsed cleanly", clean, total)
}

func hasSyntaxError(src []byte) bool {
	regions := parser.ClassifyRegions(string(src))
	if len(regions) == 1 && regions[0].Kind == parser.RegionScript {
		return treeHasError(language.CFScript, src)
	}

	if treeHasError(language.CFML, src) {
		return true
	}

	for _, r := range regions {
		if r.Kind == parser.RegionScript && treeHasError(language.CFScript, []byte(r.Text)) {
			return true
		}
	}

	return false
}

func treeHasError(g language.Grammar, src []byte) bool {
	tree := language.Parse(g, src, nil)
	if tree == nil {
		return true
	}
	defer tree.Close()

	return tree.RootNode() != nil && hasErrorNode(tree.RootNode())
}

func hasErrorNode(n *sitter.Node) bool {
	if n == nil {
		return false
	}

	return n.HasError()
}
