package parser

import (
	"slices"
	"strings"
)

// functionSpans is a parse's FunctionSpans, for the content they were found
// in: an edit inside a function moves Content and leaves the parse's other
// answers where they were, and offsets cannot be shifted by a line delta.
type functionSpans struct {
	of   string
	list []FunctionSpan
}

// keepSpan files the span the walk left on sc, and clears it, so the scope
// kept is its lines alone.
func (pr *ParseResult) keepSpan(sc *FuncScope) {
	if sc.span == nil {
		return
	}

	if pr.spans != nil {
		pr.spans.list = append(pr.spans.list, *sc.span)
	}

	sc.span = nil
}

// sort orders the spans as they are written.
func (s *functionSpans) sort() {
	slices.SortStableFunc(s.list, func(a, b FunctionSpan) int { return a.Start - b.Start })
}

// FunctionSpans is every function the parse found, where it is written in
// Content, in the order written: the declared methods and the named
// functions nested in them that the script walk reads, the functions
// assigned to a name outside any function, and every <cffunction>. A reader
// that wants a component's functions — their names, access, parameters,
// literal defaults or body text — takes them from here rather than finding
// them again, so it reads them as the parse does: strings with the parse's
// string rule, comments as comments, a <cfscript> island as script.
//
// After an incremental edit, which shifts lines but not offsets, the spans
// are read again from the current Content. What is returned is shared and
// must not be changed.
func (pr *ParseResult) FunctionSpans() []FunctionSpan {
	if pr == nil {
		return nil
	}

	if s := pr.spans; s != nil && s.of == pr.Content {
		return s.list
	}

	return ParseWithOptions(pr.URI, pr.Content, &ParseOptions{Shallow: true}).FunctionSpans()
}

// FunctionSpansOf is FunctionSpans for text there is no parse of: the
// functions-only reading of the one walk, each region read by the script
// walk or the tag function finder the parse uses (varsAndScopes), with
// nothing settled after. It answers as a parse of content would. file is
// the URI or path the text came from, or "": it is read only for its
// extension, as the parse reads it, so a tag-free .cfm page of markup is
// not read as script.
func FunctionSpansOf(file, content string) []FunctionSpan {
	regions, idx := ClassifyRegionsIdx(content)
	if isMarkupTemplate(file, content) {
		regions, idx = splitCFScriptBlocks(content)
	}

	_, _, spans := walkRegions(content, regions, idx, false)

	return spans
}

// FunctionSpanNamed is the one span in spans called name (any case), and
// false when there is none or more than one.
func FunctionSpanNamed(spans []FunctionSpan, name string) (*FunctionSpan, bool) {
	found := -1

	for i := range spans {
		if !strings.EqualFold(spans[i].Name, name) {
			continue
		}

		if found >= 0 {
			return nil, false
		}

		found = i
	}

	if found < 0 {
		return nil, false
	}

	return &spans[found], true
}
