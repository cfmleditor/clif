package server

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/cfmleditor/clif/internal/docs"
	"go.lsp.dev/protocol"
)

// defaultDocResults is how many entries clif.searchDocs answers without a limit.
const defaultDocResults = 50

// docSummary is one entry of a clif.searchDocs answer.
type docSummary struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "tag" or "function"
	Description string `json:"description"`
	Syntax      string `json:"syntax,omitempty"`
}

// docDetail is a clif.docEntry answer: the entry as the server's own hover
// documents it, with its parameters.
type docDetail struct {
	docSummary
	Member   string       `json:"member,omitempty"`
	Script   string       `json:"script,omitempty"`
	Returns  string       `json:"returns,omitempty"`
	Markdown string       `json:"markdown"`
	Params   []docs.Param `json:"params,omitempty"`
}

// cmdSearchDocs runs clif.searchDocs: the CFML tags and functions whose names
// or descriptions hold the query, so an editor can offer a documentation
// browser without carrying a copy of the docs. Arguments, each optional:
// the query (empty lists everything, by name), "tag", "function" or "all",
// and the most entries to answer.
func (s *Server) cmdSearchDocs(_ context.Context, args []protocol.LSPAny) (any, error) {
	query, _ := argString(args, 0)
	kind, _ := argString(args, 1)

	limit := defaultDocResults
	if n, ok := argFloat(args, 2); ok && n > 0 {
		limit = int(n)
	}

	switch kind {
	case "", "all", "tag", "function":
	default:
		return nil, errors.New("clif.searchDocs: the kind is tag, function or all")
	}

	return searchDocs(query, kind, limit), nil
}

// searchDocs ranks the entries of kind ("" or "all" for both) against query:
// an exact name first, then a name that starts with it, then one that holds
// it, then a description that does; ties by name.
func searchDocs(query, kind string, limit int) []docSummary {
	query = strings.ToLower(strings.TrimSpace(query))

	var pool []*docs.Entry

	if kind != "function" {
		pool = append(pool, docs.AllTags()...)
	}

	if kind != "tag" {
		pool = append(pool, docs.AllFunctions()...)
	}

	type ranked struct {
		e    *docs.Entry
		rank int
	}

	var hits []ranked

	for _, e := range pool {
		name := strings.ToLower(e.Name)

		rank := -1

		switch {
		case query == "":
			rank = 0
		case name == query:
			rank = 0
		case strings.HasPrefix(name, query):
			rank = 1
		case strings.Contains(name, query):
			rank = 2
		case strings.Contains(strings.ToLower(e.Description), query):
			rank = 3
		}

		if rank >= 0 {
			hits = append(hits, ranked{e, rank})
		}
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].rank != hits[j].rank {
			return hits[i].rank < hits[j].rank
		}

		return strings.ToLower(hits[i].e.Name) < strings.ToLower(hits[j].e.Name)
	})

	out := make([]docSummary, 0, min(limit, len(hits)))

	for _, h := range hits[:min(limit, len(hits))] {
		out = append(out, summaryOf(h.e))
	}

	return out
}

func summaryOf(e *docs.Entry) docSummary {
	first, _, _ := strings.Cut(strings.TrimSpace(e.Description), "\n")

	return docSummary{Name: e.Name, Type: e.Type, Description: first, Syntax: e.Syntax}
}

// cmdDocEntry runs clif.docEntry: the whole documentation of one tag or
// function. Arguments: the name, and "tag" or "function" where a name is both
// (a tag is looked up first otherwise). Null when there is no such entry.
func (s *Server) cmdDocEntry(_ context.Context, args []protocol.LSPAny) (any, error) {
	name, _ := argString(args, 0)
	if name == "" {
		return nil, errors.New("clif.docEntry requires a tag or function name")
	}

	kind, _ := argString(args, 1)

	var (
		e  *docs.Entry
		ok bool
	)

	switch kind {
	case "function":
		e, ok = docs.LookupFunction(name)
	case "tag":
		e, ok = docs.LookupTag(name)
	default:
		if e, ok = docs.LookupTag(name); !ok {
			e, ok = docs.LookupFunction(name)
		}
	}

	if !ok {
		return nil, nil
	}

	return docDetail{
		docSummary: summaryOf(e),
		Member:     e.Member, Script: e.Script, Returns: e.Returns,
		Markdown: e.Doc(), Params: e.Params,
	}, nil
}
