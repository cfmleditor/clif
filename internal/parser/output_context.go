package parser

import (
	"regexp"
	"strings"
)

// Where ColdFusion evaluates #...# in the text of a tag file.
//
// Outside the contexts below, a hash in text is a literal character: a CSS
// colour, an href="#", a jQuery "#id" selector. Pairing those across a chunk
// of text swallowed whatever JavaScript, CSS or attribute text sat between two
// of them, and every name( in it became a call. The rules were checked against
// Adobe ColdFusion 2025 with one probe template per context:
//
//	evaluated      <cfoutput> bodies, including HTML attributes and <script>
//	               inside them; <cfquery> and <cfmail> bodies; the body of a
//	               <cffunction output="true|yes">; the attributes of every
//	               <cf...> tag, <cf_...> custom tag, <cfmodule> and
//	               cfimport-prefixed tag (<t:echo v="#x()#">), in any context
//	not evaluated  plain text, HTML attributes and <script> bodies outside
//	               those; <cfsavecontent>, <cfxml> and custom-tag bodies
//	               outside <cfoutput>; the body of a <cffunction> whose
//	               output is unset or false; a template <cfinclude>d from
//	               inside a <cfoutput> (each template is compiled on its own)
//
// <cfmailpart> is taken with <cfmail>, and <cfdocument> and its item and
// section tags are taken as evaluated: the probe server has no PDF service to
// confirm them, and wrongly scanning costs a false positive where wrongly
// skipping would lose a call. <cfcomponent output="true"> is taken the same
// way; the probe could not exercise it, because ColdFusion rejects free
// #...# text in a component body at compile time.
//
// Attributes are handled by the tag walk, not here: this answers only whether
// the text between tags is output. The evaluated tags are outputTagKind's
// outputTag.

// outputTagKind sorts a lowercased tag name into the ones the walk acts on.
func outputTagKind(name []byte) outputKind {
	switch string(name) {
	case "cfoutput", "cfquery", "cfmail", "cfmailpart", "cfdocument", "cfdocumentitem", "cfdocumentsection":
		return outputTag
	case "cfscript":
		return outputScript
	case "cfimport":
		return outputImport
	case "cffunction":
		return outputFunction
	case "cfcomponent":
		return outputComponent
	default:
		return outputOther
	}
}

type outputKind uint8

const (
	outputOther outputKind = iota
	outputTag
	outputScript
	outputImport
	outputFunction
	outputComponent
)

// outputContext returns the byte ranges of content whose text ColdFusion
// evaluates, sorted and disjoint, and the tag prefixes content declares with
// <cfimport prefix="...">, lowercased.
//
// It fails open. A range left unclosed at the end of the file runs to the end,
// and a close with nothing open is ignored, so a file the walk misreads is
// scanned more than it should be rather than less.
func outputContext(content string) (spans [][2]int, prefixes []string) {
	var (
		depth           int
		fnOut, compOut  bool
		start           = -1
		wasActive       bool
		boundaryChanged = func(at int) {
			active := depth > 0 || fnOut || compOut
			switch {
			case active && !wasActive:
				start = at
			case !active && wasActive && start >= 0:
				if at > start {
					spans = append(spans, [2]int{start, at})
				}

				start = -1
			}

			wasActive = active
		}
	)

	for pos := 0; pos < len(content); {
		i := strings.IndexByte(content[pos:], '<')
		if i < 0 {
			break
		}

		i += pos

		if strings.HasPrefix(content[i:], "<!---") {
			pos = skipTagComment(content, i)

			continue
		}

		kind, closing, isCF := cfTagAt(content, i)
		if !isCF {
			pos = i + 1

			continue
		}

		end := tagEndIndex(content[i:])
		if end < 0 {
			pos = i + 1

			continue
		}

		tagEnd := i + end + 1
		tag := content[i:tagEnd]

		switch {
		case kind == outputOther:
		case kind == outputScript && !closing:
			// A script body holds no tag text, and a "<cfoutput" in a string
			// there is not a tag.
			if c := indexCFTag(content[tagEnd:], "/cfscript>"); c >= 0 {
				pos = tagEnd + c

				continue
			}

			pos = len(content)

			continue
		case kind == outputImport && !closing:
			if pf := strings.TrimSpace(getAttr(tag, "prefix")); pf != "" {
				prefixes = append(prefixes, strings.ToLower(pf))
			}
		case kind == outputTag && closing:
			if depth > 0 {
				depth--

				boundaryChanged(i)
			}
		case kind == outputTag && !strings.HasSuffix(strings.TrimRight(tag[:len(tag)-1], " \t\r\n"), "/"):
			depth++

			boundaryChanged(tagEnd)
		case kind == outputFunction && closing:
			if fnOut {
				fnOut = false

				boundaryChanged(i)
			}
		case kind == outputFunction && isTruthy(getAttr(tag, "output")):
			fnOut = true

			boundaryChanged(tagEnd)
		case kind == outputComponent && closing:
			if compOut {
				compOut = false

				boundaryChanged(i)
			}
		case kind == outputComponent && isTruthy(getAttr(tag, "output")):
			compOut = true

			boundaryChanged(tagEnd)
		}

		pos = tagEnd
	}

	if wasActive && start >= 0 && start < len(content) {
		spans = append(spans, [2]int{start, len(content)})
	}

	return spans, prefixes
}

// skipTagComment returns the offset just past the nested <!--- ---> comment
// opening at i, or the end of content when it is unclosed.
func skipTagComment(content string, i int) int {
	depth := 0

	for j := i; j < len(content); {
		switch {
		case strings.HasPrefix(content[j:], "<!---"):
			depth++
			j += 5
		case strings.HasPrefix(content[j:], "--->"):
			depth--
			j += 4

			if depth == 0 {
				return j
			}
		default:
			j++
		}
	}

	return len(content)
}

func isTagNameByte(c byte) bool {
	return c == '_' || c == ':' || c == '-' || c == '.' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// customPrefixTag reports whether the tag opening at s is a cfimport-prefixed
// custom tag, <prefix:name ...>, for one of prefixes. Its attributes are
// evaluated wherever it stands.
func customPrefixTag(s string, prefixes []string) bool {
	if len(prefixes) == 0 || len(s) < 3 || s[0] != '<' || s[1] == '/' {
		return false
	}

	colon := strings.IndexByte(s[1:min(len(s), 64)], ':')
	if colon <= 0 {
		return false
	}

	prefix := s[1 : 1+colon]
	for _, c := range []byte(prefix) {
		if !isTagNameByte(c) || c == ':' {
			return false
		}
	}

	for _, p := range prefixes {
		if strings.EqualFold(p, prefix) {
			return true
		}
	}

	return false
}

// markupClosingTag finds evidence of HTML: a closing tag, a doctype or an
// HTML comment. A script fragment can hold "a<b", but not "</div>".
var markupClosingTag = regexp.MustCompile(`(?i)</[a-z][a-z0-9-]*\s*>|<!doctype|<!--[^-]`)

// isMarkupTemplate reports whether a .cfm or .cfml file with no CF tags is an
// HTML template rather than CFScript. ColdFusion runs such a file as literal
// text: with no <cfoutput> it evaluates nothing, and reading it as script
// turned "Rich Text (with Images)" into a call to Text.
func isMarkupTemplate(fileURI, content string) bool {
	ext := strings.ToLower(fileURI[strings.LastIndexByte(fileURI, '.')+1:])
	if ext != "cfm" && ext != "cfml" {
		return false
	}

	// Every form markupClosingTag finds starts "</" or "<!": a file with
	// neither, which is any script-only template, is not scanned.
	if !strings.Contains(content, "</") && !strings.Contains(content, "<!") {
		return false
	}

	return !containsCFTag(content) && markupClosingTag.MatchString(content)
}

// cfTagAt reads the name of the tag opening at content[i] ('<'): which kind
// of tag the output-context walk sorts it as, whether it is a close tag, and
// whether it is a CF tag at all. Every tag in the file reaches here, so the
// name is sorted from bytes folded on the stack rather than a lowercased copy.
func cfTagAt(content string, i int) (kind outputKind, closing, isCF bool) {
	closing = i+1 < len(content) && content[i+1] == '/'

	nameStart := i + 1
	if closing {
		nameStart++
	}

	nameEnd := nameStart
	for nameEnd < len(content) && isTagNameByte(content[nameEnd]) {
		nameEnd++
	}

	if nameEnd-nameStart < 2 || toLowerByte(content[nameStart]) != 'c' || toLowerByte(content[nameStart+1]) != 'f' {
		return outputOther, closing, false
	}

	var buf foldScratch

	return outputTagKind(buf.lowerFold(content[nameStart:nameEnd])), closing, true
}

// outputState is where the tag walk stands in the contexts outputContext
// describes, kept by the walk itself as it passes each tag (track) rather
// than read from ranges a walk of its own computed first: the two walks found
// the same tags, and the second was 10-15% of a tag component's parse. One
// state serves every region of a file in order, so an output context opened
// before a <cfscript> block is still open after it. It fails open as
// outputContext does: an unclosed context runs to the end of the file, and a
// close with nothing open is ignored.
type outputState struct {
	depth          int32
	fnOut, compOut bool
}

// active reports whether text at the walk's position is evaluated.
func (o *outputState) active() bool {
	return o.depth > 0 || o.fnOut || o.compOut
}

// track moves o past the tag opening at src[idx] ('<').
func (o *outputState) track(src string, idx int) {
	kind, closing, isCF := cfTagAt(src, idx)
	if !isCF {
		return
	}

	switch kind {
	case outputTag, outputFunction, outputComponent:
	default:
		return
	}

	end := tagEndIndex(src[idx:])
	if end < 0 {
		return
	}

	tag := src[idx : idx+end+1]

	switch {
	case kind == outputTag && closing:
		if o.depth > 0 {
			o.depth--
		}
	case kind == outputTag:
		if !strings.HasSuffix(strings.TrimRight(tag[:len(tag)-1], " \t\r\n"), "/") {
			o.depth++
		}
	case kind == outputFunction && closing:
		o.fnOut = false
	case kind == outputFunction:
		o.fnOut = o.fnOut || isTruthy(getAttr(tag, "output"))
	case kind == outputComponent && closing:
		o.compOut = false
	case kind == outputComponent:
		o.compOut = o.compOut || isTruthy(getAttr(tag, "output"))
	}
}

// cfimportPrefixes is the tag prefixes content declares with <cfimport>,
// lowercased: outputContext's walk, run only on a file that mentions one.
// A prefix applies to the whole file, so the tag walk cannot collect it as it
// goes.
func cfimportPrefixes(content string) []string {
	if IndexFold(content, "<cfimport") < 0 {
		return nil
	}

	_, prefixes := outputContext(content)

	return prefixes
}
