package resolve

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// Mura's display objects are templates a renderer includes on request:
//
//	#$.dspObject_include( theFile = 'collection/includes/dsp_content_list.cfm', iterator = iterator )#
//
// contentRenderer.dspObject_Include includes the file inside the function, so
// the template's arguments scope is the call's, and a bare name it never sets
// reads that scope first. On MasaCMS the content lists read
// arguments.iterator, and pagination a bare iterator, typed nowhere else.
//
// muraObjectArg reads every dspObject_include call whose theFile could name
// the template and types the value each passes for the name where the call is
// made, as viewArgs does a ColdBox view's args. A computed theFile
// (`'collection/layouts/#objectParams.layout#/index.cfm'`) names every
// template its literal parts allow. Every call passing the name must type it,
// and they are the alternatives; one that omits it says nothing about
// arguments.X, while a bare name it omits is the renderer's own and leaves it
// untyped. A call passing an argumentCollection could pass anything, and
// leaves the name untyped; so does one passing the name whose theFile is not
// spelled, while one that neither spells theFile nor passes the name is
// passed over, since it may never include the template. A name
// dspObject_Include declares itself is not asked about, since its default
// reaches the template from every call that omits it. The calls are found
// through the batch caller index, so this runs in a batch scan only.

// muraObjectSite is one dspObject_include call: where it is, the templates it
// can include (nil when its theFile is not spelled), and the values it passes
// by name, lowercased. opaque is set by an argumentCollection.
type muraObjectSite struct {
	file   string
	line   int
	target *regexp.Regexp
	named  map[string][]parser.Token
	opaque bool
}

// maxHandOffDepth bounds a display object's arguments a level past an
// assignment's: typing `arguments.item = arguments.iterator.next()` costs a
// level before the iterator is asked about, and MasaCMS's content list is
// handed its iterator through two layouts, which ran the shared bound out one
// template short of the collection that builds it.
const maxHandOffDepth = maxAssignedDepth + 1

var muraObjectStartRe = regexp.MustCompile(`(?i)\bdspObject_include\s*\(`)

// muraObjectArg is the component variable holds in the display-object
// template pr, as the dspObject_include calls naming it pass it, or "".
func (r *Resolver) muraObjectArg(variable string, pr *parser.ParseResult, tr *callTrace, ctx lookupCtx) string {
	if pr == nil || !pr.URI.IsFile() || ctx.depth >= maxHandOffDepth || len(r.owner().InferArgsFiles) == 0 {
		return ""
	}

	name, scoped := variable, false
	if rest, ok := strings.CutPrefix(strings.ToLower(variable), "arguments."); ok {
		name, scoped = variable[len(variable)-len(rest):], true
	}

	if name == "" || strings.ContainsAny(name, ".[(") || !scoped && isCFMLScope(name) || r.muraIncludeDeclares(name) {
		return ""
	}

	// A template that sets the name reads its own value. A name it passes
	// on to a display object of its own (`iterator = iterator`) is not one
	// it sets.
	own := withoutMuraObjectArgs(pr.Content)
	if setsName(own, "arguments."+name) || !scoped && setsName(own, name) {
		return ""
	}

	file := cfpath.FromURI(string(pr.URI))
	slashed := filepath.ToSlash(file)

	var (
		comps  []string
		passed int
	)

	for i := range r.muraObjectSites() {
		s := &r.muraObjectSites()[i]
		if s.target != nil && !s.target.MatchString(slashed) {
			continue
		}

		value, ok := s.named[strings.ToLower(name)]

		switch {
		case s.opaque || s.target == nil && ok:
			return ""
		case s.target == nil:
			// A call naming its file only by position may never include
			// this template, and does not pass the name if it does.
			continue
		case !ok && !scoped:
			// A bare name a call does not pass is the renderer's own.
			return ""
		case !ok:
			continue
		}

		// The value's own lookup counts a level, which is what bounds a
		// template handing on what it was handed, round a cycle included.
		comp := r.renderedValueD(value, viewArgSite{file: s.file, line: s.line}, lookupCtx{depth: ctx.depth, callers: ctx.callers})
		if comp == "" {
			return ""
		}

		passed++

		for alt := range strings.SplitSeq(comp, "|") {
			if !containsFold(comps, alt) {
				comps = append(comps, alt)
			}
		}
	}

	if len(comps) == 0 {
		return ""
	}

	slices.Sort(comps)

	answer := strings.Join(comps, "|")
	tr.addf("%q is what the %d dspObject_include call(s) naming this template pass it: %q", variable, passed, answer)

	return answer
}

// muraIncludeDeclares reports whether a dspObject_Include in the index
// declares an argument called name.
func (r *Resolver) muraIncludeDeclares(name string) bool {
	if r.Index == nil {
		return false
	}

	for _, fd := range r.Index.Lookup("dspObject_include") {
		if slices.ContainsFunc(fd.Arguments, func(a parser.Argument) bool { return strings.EqualFold(a.Name, name) }) {
			return true
		}
	}

	return false
}

// muraObjectSites are the workspace's dspObject_include calls, read once per
// scan from the files the caller index says call it.
func (r *Resolver) muraObjectSites() []muraObjectSite {
	o := r.owner()

	o.mu.RLock()
	sites, built := o.muraSites, o.muraSitesBuilt
	o.mu.RUnlock()

	if built {
		return sites
	}

	sites = []muraObjectSite{}

	for _, file := range r.callerFiles("dspObject_include") {
		text, err := r.readText(file)
		if err != nil {
			continue
		}

		sites = append(sites, muraObjectSitesIn(text, file)...)
	}

	o.mu.Lock()
	if !o.muraSitesBuilt {
		o.muraSites, o.muraSitesBuilt = sites, true
	}

	sites = o.muraSites
	o.mu.Unlock()

	return sites
}

// muraObjectSitesIn reads the dspObject_include calls in content.
func muraObjectSitesIn(content, file string) []muraObjectSite {
	var out []muraObjectSite

	lines := newLineIndex(content)

	for _, m := range muraObjectStartRe.FindAllStringIndex(content, -1) {
		tokens := significantTokens(content[m[0]:min(len(content), m[0]+renderWindow)])
		if len(tokens) < 2 || tokens[1].Kind != parser.TokLParen {
			continue
		}

		end := producerGroupEnd(tokens, 1, parser.TokLParen, parser.TokRParen)
		if end < 0 {
			continue
		}

		site := muraObjectSite{file: file, line: lines.lineOf(m[0]), named: map[string][]parser.Token{}}

		for _, piece := range producerSplit(tokens[2:end]) {
			if len(piece) < 3 || piece[0].Kind != parser.TokIdent || piece[1].Kind != parser.TokEquals {
				// A positional argument: theFile is the fourth, and the
				// call cannot be placed.
				continue
			}

			switch key := strings.ToLower(piece[0].Value); key {
			case "thefile":
				site.target = muraObjectTarget(piece[2:])
			case "argumentcollection":
				site.opaque = true
			default:
				site.named[key] = piece[2:]
			}
		}

		out = append(out, site)
	}

	return out
}

// withoutMuraObjectArgs is content with the argument list of each
// dspObject_include call blanked, newlines kept.
func withoutMuraObjectArgs(content string) string {
	matches := muraObjectStartRe.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return content
	}

	b := []byte(content)

	for _, m := range matches {
		window := content[m[0]:min(len(content), m[0]+renderWindow)]

		tokens := significantTokens(window)
		if len(tokens) < 2 || tokens[1].Kind != parser.TokLParen {
			continue
		}

		end := producerGroupEnd(tokens, 1, parser.TokLParen, parser.TokRParen)
		if end < 0 {
			continue
		}

		for i := m[0] + tokens[1].Offset; i < m[0]+tokens[end].Offset; i++ {
			if b[i] != '\n' {
				b[i] = ' '
			}
		}
	}

	return string(b)
}

// muraObjectTarget is a pattern for the templates a theFile value can name:
// its literal parts, with what is computed matching any one path segment.
// Mura looks the file up under each display-object directory, so the pattern
// matches the end of a template's path. It is nil when nothing is spelled.
func muraObjectTarget(value []parser.Token) *regexp.Regexp {
	var b strings.Builder

	literal := false
	wild := func() {
		if !strings.HasSuffix(b.String(), `[^/]*`) {
			b.WriteString(`[^/]*`)
		}
	}

	for _, t := range value {
		switch t.Kind {
		case parser.TokString:
			text := t.Value
			if len(text) >= 2 {
				text = text[1 : len(text)-1]
			}

			for i, part := range strings.Split(text, "#") {
				if i%2 == 1 {
					wild()

					continue
				}

				if b.Len() == 0 {
					part = strings.TrimPrefix(part, "/")
				}

				if part != "" {
					b.WriteString(regexp.QuoteMeta(part))

					literal = true
				}
			}
		case parser.TokAmpersand:
		default:
			wild()
		}
	}

	if !literal {
		return nil
	}

	return regexp.MustCompile(`(?i)(?:^|/)` + b.String() + `$`)
}
