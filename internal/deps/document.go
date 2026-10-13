package deps

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

// Document is one file to draw the dependencies of.
//
// The command line (`clif deps`) and the editor (`clif.exportDeps`) each
// assembled this by hand — which calls to start from, where the refs fall back
// to, how the walk loads another file — and the two copies of the loader were
// identical down to a comment. They differ in one thing only, how a file is
// parsed, and that is the Parse callback.
type Document struct {
	// URI names the document; FuncName, when set, scopes the graph to one
	// function.
	URI      string
	FuncName string

	// PR is the document's own parse, with calls extracted. Nil when it could not
	// be read, in which case the graph falls back to IndexRefs.
	PR *parser.ParseResult

	// Parse parses any other file the walk reaches, privately. A nil result is a
	// file that could not be read.
	Parse func(fileURI uri.URI) *parser.ParseResult

	// IndexRefs are the component refs the index holds for the document, used
	// only when PR is nil.
	IndexRefs func() []parser.ComponentRef

	Index    Index
	Resolver Resolver
}

// maxDocumentDepth bounds how far a document's graph follows its dependencies.
const maxDocumentDepth = 10

// BuildDocument generates the transitive dependency graph of a document.
func BuildDocument(d *Document) Result {
	var (
		calls []parser.CallSite
		refs  []parser.ComponentRef
	)

	if d.PR != nil {
		calls = DocumentCalls(d.PR, d.FuncName)
		refs = d.PR.ComponentRefs
	} else if d.IndexRefs != nil {
		refs = d.IndexRefs()
	}

	return Build(&Options{
		DocURI:    d.URI,
		FuncName:  d.FuncName,
		Calls:     calls,
		Refs:      refs,
		Index:     d.Index,
		Resolver:  d.Resolver,
		LoadCalls: CallLoader(d.Parse),
		MaxDepth:  maxDocumentDepth,
	})
}

// DocumentCalls are the calls a document's graph starts from: the calls of the
// function named, or of every function when none is.
func DocumentCalls(pr *parser.ParseResult, funcName string) []parser.CallSite {
	var calls []parser.CallSite

	for _, sc := range pr.Scopes {
		if funcName == "" {
			calls = append(calls, pr.FuncCalls(sc.Start, sc.End)...)

			continue
		}

		if fn := functionAt(pr, funcName, sc.Start); fn != nil {
			return pr.FuncCalls(sc.Start, sc.End)
		}
	}

	return calls
}

// CallLoader is the Options.LoadCalls hook over a parse: it parses a target file
// on demand so the traversal can continue past the first hop, memoised per file
// since one traversal asks about several functions in the same component and the
// parse is the expensive part. It is short-lived by design — one per graph, so
// it never becomes a second cache to invalidate.
func CallLoader(parse func(fileURI uri.URI) *parser.ParseResult) func(fileURI uri.URI, funcName string) ([]parser.CallSite, []parser.ComponentRef) {
	parsed := make(map[uri.URI]*parser.ParseResult)

	return func(fileURI uri.URI, funcName string) ([]parser.CallSite, []parser.ComponentRef) {
		pr, ok := parsed[fileURI]
		if !ok {
			pr = parse(fileURI)
			parsed[fileURI] = pr
		}

		if pr == nil {
			return nil, nil
		}

		for _, sc := range pr.Scopes {
			if functionAt(pr, funcName, sc.Start) == nil {
				continue
			}

			// This function's own locals first, then the file-level refs. A
			// `var svc = new Foo()` shadows a `variables.svc` of the same name,
			// and the consumer takes the first match — the other order made the
			// graph follow the shadowed component. resolve.CanResolveCall
			// resolves in this order too.
			return pr.FuncCalls(sc.Start, sc.End), slices.Concat(pr.FuncComponentRefs(sc.Start, sc.End), pr.ComponentRefs)
		}

		return nil, nil
	}
}

// functionAt is the function called name declared at line, or nil.
func functionAt(pr *parser.ParseResult, name string, line int) *parser.FunctionDef {
	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		if strings.EqualFold(f.Name, name) && int(f.Line) == line {
			return f
		}
	}

	return nil
}
