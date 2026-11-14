package parser

import (
	"slices"
	"strings"
)

// TagProblemKind is what is wrong with the tags of a page.
type TagProblemKind string

// The problems TagProblems reports.
const (
	// ProblemUnclosedTag is a tag that takes a body, left open: `<cfif>` with
	// no `</cfif>` before its parent closes or the page ends.
	ProblemUnclosedTag TagProblemKind = "unclosed-tag"
	// ProblemUnmatchedClose is a closing tag with no opener: a `</cfif>` with
	// no `<cfif>` open.
	ProblemUnmatchedClose TagProblemKind = "unmatched-close"
	// ProblemMisplacedTag is a tag that has to sit inside another: a
	// `<cfcatch>` outside any `<cftry>`.
	ProblemMisplacedTag TagProblemKind = "misplaced-tag"
	// ProblemMissingAttribute is a tag without an attribute it cannot do
	// without: a `<cffunction>` with no name.
	ProblemMissingAttribute TagProblemKind = "missing-attribute"
	// ProblemDuplicateAttribute is an attribute given twice to one tag.
	ProblemDuplicateAttribute TagProblemKind = "duplicate-attribute"
)

// TagProblem is one problem, at byte offsets of the page: Start to End is the
// text to underline, the tag's name or the attribute.
type TagProblem struct {
	Kind       TagProblemKind
	Start, End int
	Message    string
	// Warning is set on a problem the engine may still run: a repeated
	// attribute is read as its last value, where an unclosed tag does not
	// compile.
	Warning bool
}

// endTagRequired are the CF tags that take a body, and so a closing tag, in
// every use. It is short on purpose: a tag that sometimes stands alone
// (`<cfhttp>`, `<cftransaction action="commit">`, `<cfmail>` with a query)
// is not here, and a close for one is left alone too, since which it is
// cannot be told from the tag's text.
var endTagRequired = map[string]bool{
	"cfif": true, "cfloop": true, "cfoutput": true, "cfquery": true, "cffunction": true,
	"cftry": true, "cfcatch": true, "cffinally": true, "cfswitch": true, "cfcase": true,
	"cfdefaultcase": true, "cfcomponent": true, "cfinterface": true, "cfsavecontent": true,
	"cflock": true, "cfsilent": true, "cfxml": true,
}

// mustSitIn are the tags that mean nothing outside the tag they belong to,
// and the tags that may hold them.
var mustSitIn = map[string][]string{
	"cfcatch": {"cftry"}, "cffinally": {"cftry"},
	"cfelse": {"cfif"}, "cfelseif": {"cfif"},
	"cfcase": {"cfswitch"}, "cfdefaultcase": {"cfswitch"},
	"cfargument": {"cffunction"},
}

// requiredAttribute is the attribute a tag cannot be used without. A tag is
// listed only when every engine insists: `name` on a `<cfquery>` is optional,
// and the others' requirements depend on an action.
var requiredAttribute = map[string]string{
	"cffunction": "name", "cfargument": "name", "cfproperty": "name", "cfparam": "name",
	"cfinclude": "template", "cflocation": "url", "cfswitch": "expression",
}

// expressionTags hold an expression, not attributes: in `<cfset a = 1>` the
// `a` is read as an attribute name, and `a = 1 and a = 2` is no repeat.
var expressionTags = map[string]bool{
	"cfset": true, "cfif": true, "cfelseif": true, "cfreturn": true, "cfwhile": true,
}

type problemFrame struct {
	name       string
	start, end int // the tag's `<` and the end of its name
}

// TagProblems reports the structural faults of the CF tags in content: tags
// left open, closing tags with no opener, tags outside the tag they belong
// to, and attributes missing or repeated. Only the tags above are checked,
// each held to a rule every engine shares, so a page that runs reports
// nothing. It reads the tag regions of a page (script is left to the script
// parse) with the walk the folding ranges use: comments, quoted attribute
// values and a `<` that starts no tag are stepped over as they are there.
func TagProblems(content string) []TagProblem {
	regions, _ := ClassifyRegionsIdx(content)

	var (
		problems []TagProblem
		stack    []problemFrame
	)

	for i := range regions {
		r := &regions[i]
		if r.Kind != RegionTag || r.Island {
			continue
		}

		problems, stack = tagRegionProblems(r.Text, r.Offset, problems, stack)
	}

	// What is still open when the page ends.
	for _, f := range stack {
		if endTagRequired[f.name] {
			problems = append(problems, unclosedProblem(f))
		}
	}

	return problems
}

func unclosedProblem(f problemFrame) TagProblem {
	return TagProblem{Kind: ProblemUnclosedTag, Start: f.start, End: f.end, Message: "Unclosed tag <" + f.name + ">"}
}

// tagRegionProblems walks one tag region, which stands at base in the page,
// adding to problems and stack.
func tagRegionProblems(text string, base int, problems []TagProblem, stack []problemFrame) ([]TagProblem, []problemFrame) {
	ends := newTagEnds(text)

	for pos := 0; pos < len(text); {
		k := strings.IndexByte(text[pos:], '<')
		if k < 0 {
			break
		}

		lt := pos + k
		rest := text[lt:]

		switch {
		case strings.HasPrefix(rest, "<!--") && lt > 0 && (text[lt-1] == '"' || text[lt-1] == '\''):
			// A string in a `#...#` expression, `replace( s, "<!--", "" )`, is
			// no comment: markup's never follows a quote.
			pos = lt + 4
		case strings.HasPrefix(rest, "<!--"):
			if strings.HasPrefix(rest, "<!---") {
				pos = SkipCFMLComment(text, lt)
			} else if c := strings.Index(rest[4:], "-->"); c >= 0 {
				pos = lt + 4 + c + 3
			} else {
				pos = len(text)
			}
		case len(rest) > 1 && rest[1] == '/':
			name := tagName(rest[2:])
			if name == "" {
				pos = lt + 2

				continue
			}

			pos = tagEnd(&ends, lt+2+len(name))
			problems, stack = closingProblem(strings.ToLower(name), base+lt, base+lt+2+len(name), problems, stack)
		default:
			name := tagName(rest[1:])
			if name == "" {
				pos = lt + 1

				continue
			}

			gt := tagEnd(&ends, lt+1+len(name))
			pos = gt

			lower := strings.ToLower(name)
			if len(lower) > 2 && lower[0] == 'c' && lower[1] == 'f' {
				problems = tagAttributeProblems(text[lt:gt], lower, base+lt, base+lt+1+len(name), problems)
				problems, stack = openingProblem(lower, base+lt, base+lt+1+len(name), selfClosing(text, lt, gt), problems, stack)

				continue
			}

			// An HTML tag's quoted values may hold CF tags, `class="a<cfif b>
			// c</cfif>"`, which the walk above stepped over as text.
			problems, stack = embeddedProblems(text, lt+1+len(name), gt, base, &ends, problems, stack)

			// <script> and <style> hold code, where `a<b` is no tag.
			if raw := rawTextEnd(text, gt, name); raw >= 0 && !selfClosing(text, lt, gt) {
				pos = raw
			}
		}
	}

	return problems, stack
}

// embeddedProblems files the CF tags written inside the attribute values of an
// HTML tag, whose attributes stand in text from from to gt.
func embeddedProblems(text string, from, gt, base int, ends *tagEnds, problems []TagProblem, stack []problemFrame) ([]TagProblem, []problemFrame) {
	for i := from; i < gt-1; i++ {
		if text[i] != '<' {
			continue
		}

		closing := text[i+1] == '/'

		nameAt := i + 1
		if closing {
			nameAt++
		}

		name := tagName(text[nameAt:gt])
		if len(name) < 3 || !strings.EqualFold(name[:2], "cf") {
			continue
		}

		lower := strings.ToLower(name)
		if closing {
			problems, stack = closingProblem(lower, base+i, base+nameAt+len(name), problems, stack)

			continue
		}

		end := min(tagEnd(ends, nameAt+len(name)), len(text))
		problems = tagAttributeProblems(text[i:end], lower, base+i, base+nameAt+len(name), problems)
		problems, stack = openingProblem(lower, base+i, base+nameAt+len(name), selfClosing(text, i, end), problems, stack)
	}

	return problems, stack
}

// openingProblem files an opening CF tag: its parent must be open, and a tag
// with a body goes on the stack.
func openingProblem(name string, start, end int, selfClosed bool, problems []TagProblem, stack []problemFrame) ([]TagProblem, []problemFrame) {
	if parents, ok := mustSitIn[name]; ok {
		held := slices.ContainsFunc(stack, func(f problemFrame) bool { return slices.Contains(parents, f.name) })
		if !held {
			problems = append(problems, TagProblem{
				Kind: ProblemMisplacedTag, Start: start, End: end,
				Message: "<" + name + "> must be inside a <" + parents[0] + "> tag",
			})
		}
	}

	if endTagRequired[name] && !selfClosed {
		stack = append(stack, problemFrame{name: name, start: start, end: end})
	}

	return problems, stack
}

// closingProblem pairs a closing tag with the nearest open tag of its name.
// The tags left open above it are unclosed; a close with no opener is
// unmatched, when it is a tag that takes a body.
func closingProblem(name string, start, end int, problems []TagProblem, stack []problemFrame) ([]TagProblem, []problemFrame) {
	if !endTagRequired[name] {
		return problems, stack
	}

	for i, s := range slices.Backward(stack) {
		if s.name != name {
			continue
		}

		for _, f := range stack[i+1:] {
			problems = append(problems, unclosedProblem(f))
		}

		return problems, stack[:i]
	}

	problems = append(problems, TagProblem{
		Kind: ProblemUnmatchedClose, Start: start, End: end,
		Message: "Unmatched closing tag </" + name + ">",
	})

	return problems, stack
}

// tagAttributeProblems checks the attributes of the CF tag whose text is tag,
// which stands at start in the page and whose name ends at nameEnd.
func tagAttributeProblems(tag, name string, start, nameEnd int, problems []TagProblem) []TagProblem {
	required, needs := requiredAttribute[name]
	if !needs && expressionTags[name] {
		return problems
	}

	var seen []string

	found := false
	scan := newAttrScanner(tag, cfAttrs)

	for {
		a, ok := scan.next()
		if !ok {
			break
		}

		attr := strings.ToLower(tag[a.nameStart:a.nameEnd])
		if needs && (attr == required || attr == "attributecollection") {
			found = true
		}

		if expressionTags[name] {
			continue
		}

		if slices.Contains(seen, attr) {
			problems = append(problems, TagProblem{
				Kind: ProblemDuplicateAttribute, Start: start + a.nameStart, End: start + a.nameEnd, Warning: true,
				Message: "Duplicate attribute '" + attr + "'",
			})
		}

		seen = append(seen, attr)
	}

	if needs && !found && !valuelessAttribute(tag, required) {
		problems = append(problems, TagProblem{
			Kind: ProblemMissingAttribute, Start: start, End: nameEnd,
			Message: "Tag <" + name + "> requires a '" + required + "' attribute",
		})
	}

	return problems
}

// valuelessAttribute reports whether tag names attr with no value, which the
// attribute scanner passes over: `<cfparam name>` holds one, and is not a tag
// without it.
func valuelessAttribute(tag, attr string) bool {
	lower := strings.ToLower(tag)

	for from := 0; ; {
		i := strings.Index(lower[from:], attr)
		if i < 0 {
			return false
		}

		i += from
		from = i + len(attr)

		before := i == 0 || !isTagNameByte(lower[i-1])
		after := from >= len(lower) || !isTagNameByte(lower[from])

		if before && after {
			return true
		}
	}
}
