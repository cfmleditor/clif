package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// The two long-lived processes bound their text caches at startup; without
// it each keeps up to a batch scan's budget of superseded document versions.
func TestLongLivedProcessesLimitTextCaches(t *testing.T) {
	for file, fn := range map[string]string{"main.go": "runServer", "mcp.go": "cmdMCP"} {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}

		calls := false

		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Name.Name != fn {
				continue
			}

			ast.Inspect(fd.Body, func(n ast.Node) bool {
				if c, ok := n.(*ast.CallExpr); ok {
					if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "limitLongLivedCaches" {
						calls = true
					}
				}

				return true
			})
		}

		if !calls {
			t.Errorf("%s in %s does not call limitLongLivedCaches", fn, file)
		}
	}
}
