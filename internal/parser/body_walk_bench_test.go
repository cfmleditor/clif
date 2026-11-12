package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.lsp.dev/uri"
)

// bodyBenchFiles are the corpus files TestLazyBodyReadCost times: the
// largest script function there is (Lucee's LDEV4901, one 578KB function),
// Mura's largest tag and script components and ContentBox's largest.
var bodyBenchFiles = []string{
	"lucee_Lucee/test/tickets/LDEV4901.cfc",
	"MSU-NatSci_MuraCMS/core/mura/publisher.cfc",
	"MSU-NatSci_MuraCMS/core/mura/content/contentRenderer.cfc",
	"MSU-NatSci_MuraCMS/core/mura/client/api/json/v1/apiUtility.cfc",
	"Ortus-Solutions_ContentBox/modules/contentbox/models/system/CBHelper.cfc",
	"Ortus-Solutions_ContentBox/modules/contentbox/models/content/BaseContent.cfc",
}

// TestLazyBodyReadCost prints what one function's lazy reads cost on the
// server's path: a parse as the editor holds it, then, for the largest and
// the median function, a keystroke inside it (ApplyEdit) after another,
// followed by its variables, its refs and its calls, and the file's
// variables-scope names.
// It reads the public API alone, so the same file times any build. Set
// CLIF_BODY_BENCH to the corpus directory to run it.
func TestLazyBodyReadCost(t *testing.T) {
	root := os.Getenv("CLIF_BODY_BENCH")
	if root == "" {
		t.Skip("set CLIF_BODY_BENCH to the corpus directory")
	}

	for _, rel := range bodyBenchFiles {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}

		u := uri.File(filepath.Join(root, rel))
		src := string(data)
		pr := ParseWithOptions(u, src, &ParseOptions{})

		if len(pr.Scopes) == 0 {
			continue
		}

		largest, sizes := pr.Scopes[0], make([]FuncScope, 0, len(pr.Scopes))
		for _, s := range pr.Scopes {
			if s.End-s.Start > largest.End-largest.Start {
				largest = s
			}

			sizes = append(sizes, s)
		}

		median := sizes[len(sizes)/2]

		for _, pick := range []struct {
			name  string
			scope FuncScope
		}{{"largest", largest}, {"median", median}} {
			s := pick.scope
			line := s.Start + 1

			// A keystroke after another: the first, and every read it
			// prompts, are not timed, so what is timed is a read after a
			// keystroke in a function being typed in.
			edit := func() *ParseResult {
				p := ParseWithOptions(u, src, &ParseOptions{})
				p.ApplyEdit(line, 0, line, 0, " ")
				_ = p.FuncVars(s.Start, s.End)
				_, _ = p.FuncRefs(s.Start, s.End)
				_ = p.VariablesVars()
				_ = p.HasScopedAssignment(ScopeVariables, "nosuchname")
				p.ApplyEdit(line, 0, line, 0, " ")

				return p
			}

			var p *ParseResult

			timed := func(read func()) time.Duration {
				best := time.Duration(0)

				for range 7 {
					p = edit()
					start := time.Now()

					read()

					if d := time.Since(start); best == 0 || d < best {
						best = d
					}
				}

				return best
			}

			vars := timed(func() { _ = p.FuncVars(s.Start, s.End) })
			refs := timed(func() { _, _ = p.FuncRefs(s.Start, s.End) })
			calls := timed(func() { _ = p.FuncCalls(s.Start, s.End) })
			scoped := timed(func() { _ = p.VariablesVars() })
			anyScoped := timed(func() { _ = p.HasScopedAssignment(ScopeVariables, "nosuchname") })

			fmt.Printf("%-40s %-8s lines=%-6d vars=%-10v refs=%-10v calls=%-10v variablesVars=%-10v anyScoped=%v\n",
				filepath.Base(rel), pick.name, s.End-s.Start+1, vars, refs, calls, scoped, anyScoped)
		}
	}
}
