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

// localNames tracks variable names declared local in a script function body.
// Up to maxLocalScan names are kept in a small slice and scanned with
// strings.EqualFold — avoiding map buckets and strings.ToLower allocations for
// the common case — and promoted to a map only for functions with many locals.
type localNames struct {
	names []string
	set   map[string]struct{}
}

func (l *localNames) reset() {
	l.names = l.names[:0]
	if l.set != nil {
		clear(l.set)
		l.set = nil
	}
}

func (l *localNames) has(name string) bool {
	if l == nil {
		return false
	}

	if l.set != nil {
		var buf foldScratch

		_, ok := l.set[string(buf.lowerFold(name))]

		return ok
	}

	for _, v := range l.names {
		if strings.EqualFold(v, name) {
			return true
		}
	}

	return false
}

func (l *localNames) add(name string) {
	if l == nil {
		return
	}

	if l.set != nil {
		var buf foldScratch
		if _, ok := l.set[string(buf.lowerFold(name))]; ok {
			return
		}

		l.set[strings.ToLower(name)] = struct{}{}
		l.names = append(l.names, name)

		return
	}

	for _, v := range l.names {
		if strings.EqualFold(v, name) {
			return
		}
	}

	l.names = append(l.names, name)
	if len(l.names) > maxLocalScan {
		l.set = make(map[string]struct{}, 2*len(l.names))
		for _, v := range l.names {
			l.set[strings.ToLower(v)] = struct{}{}
		}
	}
}

func (l *localNames) clone() *localNames {
	out := &localNames{}
	if l == nil || len(l.names) == 0 {
		return out
	}

	out.names = make([]string, len(l.names))
	copy(out.names, l.names)

	if l.set != nil {
		out.set = make(map[string]struct{}, len(l.set))
		for k := range l.set {
			out.set[k] = struct{}{}
		}
	}

	return out
}

// scriptExtras is what a script parser records in some parses only, behind
// one pointer: a slice here moved the struct up a size class
// (TestParserStructsKeepTheirSize).
type scriptExtras struct {
	// cols turns the offsets of declarations into columns in order, so many
	// on one line do not each measure it (columnCursor).
	cols columnCursor
	// implements is the implements attribute of the component header.
	implements     string
	instantiations []Instantiation
	// callNames, lowercased, are the only names whose calls are recorded;
	// nil records every call.
	callNames []string
	// rootLocals is reused across top-level functions in parseBraceBody so
	// each function does not allocate a fresh map or slice.
	rootLocals localNames
	// spanSlab and argSlab batch FunctionSpan and []Argument allocations
	// across functions in a script file.
	spanSlab []FunctionSpan
	argSlab  []Argument
}

func (p *scriptParser) ensureExtras() *scriptExtras {
	if p.extras == nil {
		p.extras = &scriptExtras{}
	}

	return p.extras
}

func (p *scriptParser) allocSpan() *FunctionSpan {
	ex := p.ensureExtras()
	if len(ex.spanSlab) == 0 {
		ex.spanSlab = make([]FunctionSpan, 16)
	}

	s := &ex.spanSlab[0]
	ex.spanSlab = ex.spanSlab[1:]

	return s
}

func (p *scriptParser) allocArgs(args []Argument) []Argument {
	if len(args) == 0 {
		return nil
	}

	ex := p.ensureExtras()
	start := len(ex.argSlab)
	ex.argSlab = append(ex.argSlab, args...)
	end := len(ex.argSlab)

	return ex.argSlab[start:end:end]
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
