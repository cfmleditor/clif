package config

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestEveryConfigLookupChecksOwnership: the config file's names reach a path
// only through CandidatesIn or TrustedCandidates, which leave out a file
// another local user could have written. A walk that joined a directory with
// FileName, LegacyFileName or FileNames itself, or spelled the name out,
// would read a planted /tmp/.clif.json again. IsFileName stays allowed: it
// tests a name and opens nothing.
func TestEveryConfigLookupChecksOwnership(t *testing.T) {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}

	root := filepath.Join(filepath.Dir(self), "..", "..")
	here := filepath.Dir(self)

	banned := map[string]bool{"FileName": true, "LegacyFileName": true, "FileNames": true}
	allowed := map[string]bool{
		filepath.Join(here, "names.go"): true, // declares them
		filepath.Join(here, "trust.go"): true, // TrustedCandidates itself
	}

	checked := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || d.Name() == "target") {
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || allowed[path] {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}

		checked++

		inConfig := filepath.Dir(path) == here

		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SelectorExpr:
				if x, ok := n.X.(*ast.Ident); ok && x.Name == "config" && banned[n.Sel.Name] {
					t.Errorf("%s uses config.%s; look a config file up through config.CandidatesIn or config.TrustedCandidates", path, n.Sel.Name)
				}

				return true
			case *ast.Ident:
				if inConfig && banned[n.Name] {
					t.Errorf("%s uses %s; look a config file up through CandidatesIn or TrustedCandidates", path, n.Name)
				}
			case *ast.BasicLit:
				if s, err := strconv.Unquote(n.Value); err == nil && (s == FileName || s == LegacyFileName) {
					t.Errorf("%s spells out %q; look a config file up through config.CandidatesIn or config.TrustedCandidates", path, s)
				}
			}

			return true
		})

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if checked < 100 {
		t.Fatalf("checked %d files; the walk is not reaching the module", checked)
	}
}
