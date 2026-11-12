// Recording only the calls of some names: ParseOptions.FindCalls without ExtractCalls.

package parser

import "strings"

// A parse asked for the calls of a few names (ParseOptions.FindCalls) and not
// for every call walks the code as ExtractCalls does and records only those:
// the resolver's framework readers (setView, hasMany, map().to(), …) take
// their calls from the parse this way rather than walking the file's tokens
// themselves. It used to find them by searching each line's text for
// `name(`, which matched a commented-out call, a call in a string and a
// function's own declaration.
//
// What it skips is work, not a rule: the walk, the receivers, the chains and
// the spans are ExtractCalls', so a name's calls are the ones an ExtractCalls
// parse records. Nothing else is recorded: no instantiation, and none of the
// member writes a tag file's <cfset> sub-parse keeps when it extracts calls,
// so every other field of the result is the plain parse's.

// scriptExtras is what a script parser records in some parses only, behind
// one pointer: a slice here moved the struct up a size class
// (TestParserStructsKeepTheirSize).
type scriptExtras struct {
	instantiations []Instantiation
	// callNames, lowercased, are the only names whose calls are recorded;
	// nil records every call.
	callNames []string
}

// lowerNames is names lowercased, or nil for none.
func lowerNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}

	out := make([]string, len(names))
	for i, n := range names {
		out[i] = strings.ToLower(n)
	}

	return out
}

// namedIn reports whether name is one of names (lowercased), ignoring case.
func namedIn(names []string, name string) bool {
	for _, n := range names {
		if strings.EqualFold(n, name) {
			return true
		}
	}

	return false
}

// mentionsAnyName reports whether text holds one of names, ignoring case: an
// expression that holds none cannot call one, and is not parsed for calls.
func mentionsAnyName(text string, names []string) bool {
	for _, n := range names {
		if IndexFold(text, n) >= 0 {
			return true
		}
	}

	return false
}

// setCallNames limits the calls p records to names, lowercased; nil records
// every call.
func (p *scriptParser) setCallNames(names []string) {
	if names == nil && p.extras == nil {
		return
	}

	if p.extras == nil {
		p.extras = &scriptExtras{}
	}

	p.extras.callNames = names
}

// callNames is what setCallNames set.
func (p *scriptParser) callNames() []string {
	if p.extras == nil {
		return nil
	}

	return p.extras.callNames
}

// recordsCall reports whether a call of name is recorded: the parse extracts
// calls, and of every name or of this one.
func (p *scriptParser) recordsCall(name string) bool {
	if !p.extractCalls {
		return false
	}

	names := p.callNames()

	return names == nil || namedIn(names, name)
}

// mayCall reports whether expr can hold a call p records: one of its names
// is written in it, when it records only some.
func (p *tagParser) mayCall(expr string) bool {
	return p.callNames == nil || mentionsAnyName(expr, p.callNames)
}

// callNames is the names the parse records the calls of when it does not
// extract every call (FindCalls, lowercased); nil when it extracts them all,
// or none.
func (pr *ParseResult) callNames() []string {
	if pr.extractCalls {
		return nil
	}

	return pr.findCalls
}

// walksCalls reports whether the parse's walk records calls: of every name,
// or of the names FindCalls gave.
func (pr *ParseResult) walksCalls() bool {
	return pr.extractCalls || len(pr.findCalls) > 0
}

// RecordsCallsOf reports whether the parse recorded the calls of every one of
// names, so that CallsWritten answers for them: it extracts every call, or
// it was asked for these names' calls (FindCalls).
func (pr *ParseResult) RecordsCallsOf(names ...string) bool {
	if !pr.walksCalls() {
		return false
	}

	only := pr.callNames()
	if only == nil {
		return true
	}

	for _, n := range names {
		if !namedIn(only, n) {
			return false
		}
	}

	return true
}
