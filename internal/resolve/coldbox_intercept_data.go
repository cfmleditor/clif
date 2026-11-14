package resolve

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// A ColdBox interceptor is handed the data struct the code announcing its
// interception point passed:
//
//	announce( "cbadmin_post#variables.entity#Save", { content : oContent, isNew : isNew } )
//
// reaches `function cbadmin_postEntrySave( event, data )`, whose
// arguments.data.content is that oContent. Nothing in the interceptor says so,
// and on ContentBox its notification and cache-cleanup listeners read every
// entry, page and content store they are told about that way.
//
// interceptData reads every announce(), announceInterception() and
// processState() call whose state could be the function's name and types the
// value each passes under the key where the call is made, as viewArgs does a
// ColdBox view's args. A computed state (`"cbadmin_post#variables.entity#Save"`)
// names every interception point its literal parts allow. Every call passing
// the key must type it, and they are the alternatives; one that passes no data
// or omits the key says nothing about it. A call whose data is not a struct
// literal could pass anything, and leaves the key untyped; so does a call
// passing the key whose state is not spelled at all. The calls are found
// through the batch caller index, so this runs in a batch scan only.

// interceptSite is one announce of an interception point: where it is, the
// points it can announce (nil when its state is not spelled), and the tokens
// of the data it passes, nil when it passes none.
type interceptSite struct {
	file   string
	line   int
	target *wildPattern
	data   []parser.Token
}

var interceptAnnouncers = []string{"announce", "announceInterception", "processState"}

// interceptData is the component variable holds in the interceptor method
// caller of pr, as the calls announcing that interception point pass it, or
// "".
func (r *Resolver) interceptData(variable, caller string, pr *parser.ParseResult, tr *callTrace, ctx lookupCtx) string {
	if pr == nil || caller == "" || ctx.depth >= maxAssignedDepth || len(r.owner().InferArgsFiles) == 0 {
		return ""
	}

	root, key, ok := interceptDataKey(variable)
	if !ok || !interceptorSignature(pr, caller, root) {
		return ""
	}

	var (
		comps  []string
		passed int
	)

	sites := r.interceptSites()
	for i := range sites {
		s := &sites[i]
		if s.target != nil && !s.target.match(caller) {
			continue
		}

		value, given, literal := argsMember(s.data, key)

		switch {
		case s.data == nil:
			// A call passing no data says nothing about any key.
			continue
		case !literal:
			if s.target == nil {
				continue
			}

			return ""
		case !given:
			continue
		case s.target == nil:
			return ""
		}

		// The value's own lookup counts a level, as a passed name's does.
		comp := r.renderedValueD(value, viewArgSite{file: s.file, line: s.line}, lookupCtx{depth: ctx.depth, callers: ctx.callers})
		if comp == "" {
			return ""
		}

		passed++

		for alt := range strings.SplitSeq(comp, "|") {
			if !lex.ContainsFold(comps, alt) {
				comps = append(comps, alt)
			}
		}
	}

	if len(comps) == 0 {
		return ""
	}

	slices.Sort(comps)

	answer := strings.Join(comps, "|")
	tr.addf("%q is what the %d call(s) announcing %s pass it: %q", variable, passed, caller, answer)

	return answer
}

// interceptDataKey splits `arguments.data.content` or `data.content` (or
// interceptData) into the struct's name and the key read from it.
func interceptDataKey(variable string) (root, key string, ok bool) {
	rest := variable
	if after, scoped := strings.CutPrefix(strings.ToLower(variable), "arguments."); scoped {
		rest = variable[len(variable)-len(after):]
	}

	root, key, ok = strings.Cut(rest, ".")
	if !ok || key == "" || strings.ContainsAny(key, ".[(") {
		return "", "", false
	}

	if !strings.EqualFold(root, "data") && !strings.EqualFold(root, "interceptData") {
		return "", "", false
	}

	return root, key, true
}

// interceptorSignature reports whether the function caller in pr takes the
// interceptor's arguments, event and then the data struct named root, and
// does not assign that struct anything else itself.
func interceptorSignature(pr *parser.ParseResult, caller, root string) bool {
	i := slices.IndexFunc(pr.Funcs, func(f parser.FunctionDef) bool { return strings.EqualFold(f.Name, caller) })
	if i < 0 {
		return false
	}

	fd := &pr.Funcs[i]
	if len(fd.Arguments) < 2 || !strings.EqualFold(fd.Arguments[0].Name, "event") || !strings.EqualFold(fd.Arguments[1].Name, root) {
		return false
	}

	scope := pr.FuncScopeAt(int(fd.Line))
	if scope.Start == -1 {
		return false
	}

	body := sourceText(pr.Content, scope.Start, scope.End+1)
	writes := func() []parser.Assignment { return onLines(pr.Assignments(), scope.Start, scope.End) }

	return !setsName(body, writes, root) && !setsName(body, writes, "arguments."+root)
}

// interceptSites are the workspace's announces, read once per scan from the
// files the caller index says call one of the announcers.
func (r *Resolver) interceptSites() []interceptSite {
	o := r.owner()

	o.mu.RLock()
	sites, built := o.interceptSiteList, o.interceptSitesBuilt
	o.mu.RUnlock()

	if built {
		return sites
	}

	var files []string
	for _, name := range interceptAnnouncers {
		files = append(files, r.callerFiles(name)...)
	}

	slices.Sort(files)

	sites = []interceptSite{}

	for _, file := range slices.Compact(files) {
		text, err := r.readText(file)
		if err != nil {
			continue
		}

		sites = append(sites, interceptSitesIn(r.plainParseOf(file, text), file)...)
	}

	o.mu.Lock()
	if !o.interceptSitesBuilt {
		o.interceptSiteList, o.interceptSitesBuilt = sites, true
	}

	sites = o.interceptSiteList
	o.mu.Unlock()

	return sites
}

// interceptSitesIn reads the announces the parse pr of file recorded: the
// state first, by position or as state=, and the data second, or as data= or
// interceptData=.
func interceptSitesIn(pr *parser.ParseResult, file string) []interceptSite {
	calls := parsedCalls(pr, interceptAnnouncers...)
	out := make([]interceptSite, 0, len(calls))

	for _, call := range calls {
		site := interceptSite{file: file, line: int(call.name().Line)}

		for i, piece := range producerSplit(call.args()) {
			if len(piece) == 0 {
				continue
			}

			value, name := piece, ""
			if len(piece) > 2 && piece[0].Kind == parser.TokIdent && (piece[1].Kind == parser.TokEquals || piece[1].Kind == parser.TokColon) {
				value, name = piece[2:], strings.ToLower(piece[0].Value)
			}

			switch {
			case name == "state" || name == "" && i == 0:
				site.target = interceptTarget(value)
			case name == "data" || name == "interceptdata" || name == "" && i == 1:
				site.data = value
			}
		}

		out = append(out, site)
	}

	return out
}

// interceptTarget is a pattern for the interception points a state value can
// name: its literal parts, with what is computed matching any run of name
// characters. It is nil when nothing is spelled.
func interceptTarget(value []parser.Token) *wildPattern {
	var b wildBuilder

	spelledParts(value, &b, false)

	return b.pattern(wildWord, false)
}
