package server

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// TestAReaderNeverSeesAFileHalfIndexed: indexing a file wrote its functions
// first and its this-scoped variables, extends and includes in later calls,
// so a completion that arrived between them listed a method but not the
// property beside it (TestIntelliLuceeCompletion, once in a while under load).
// Both doors, the workspace scan and the editor's buffer, write a file whole.
func TestAReaderNeverSeesAFileHalfIndexed(t *testing.T) {
	const src = "component extends=\"Base\" {\n  this.price = 0;\n  function calculateTax() {}\n}"

	doors := map[string]func(s *Server, pr *parser.ParseResult){
		"scan":   func(s *Server, pr *parser.ParseResult) { s.applyIndexResult(cfpath.ToURI("/w/Product.cfc"), pr) },
		"buffer": func(s *Server, pr *parser.ParseResult) { s.reindexFromParseResult(cfpath.ToURI("/w/Product.cfc"), pr) },
	}

	for name, write := range doors {
		t.Run(name, func(t *testing.T) {
			s := newTestServer()
			docURI := cfpath.ToURI("/w/Product.cfc")
			pr := parser.Parse(docURI, src)

			var (
				stop    atomic.Bool
				seen    atomic.Int64
				torn    atomic.Int64
				readers sync.WaitGroup
			)

			readers.Add(1)

			go func() {
				defer readers.Done()

				for !stop.Load() {
					if len(s.index.FunctionsForFile(docURI)) == 0 {
						continue
					}

					seen.Add(1)

					ext, _ := s.index.ExtendsForFile(docURI)
					if len(s.index.ThisVarsForFile(docURI)) == 0 || ext == "" {
						torn.Add(1)
					}
				}
			}()

			for range 20000 {
				write(s, pr)
			}

			stop.Store(true)
			readers.Wait()

			if seen.Load() == 0 {
				t.Fatal("the reader never saw the file indexed")
			}

			if torn.Load() > 0 {
				t.Errorf("the reader found the file's functions without its this-scoped variables or extends %d times in %d reads", torn.Load(), seen.Load())
			}
		})
	}
}
