package codemap

import (
	"regexp"
	"strings"
	"testing"
)

// TestViewerEscapesEveryInterpolation holds the viewer to one rule: markup is
// built only with the h`...` tag, which escapes every interpolated value, and
// it reaches the DOM only through a sink that escapes anything else.
//
// Every string in the map came from source on disk. A file called
// `<img src=x onerror=alert(1)>.cfm`, or `access="<img ...>"` on a
// <cffunction>, ran script in the reader's browser when a tooltip or the detail
// panel interpolated it into innerHTML. A behavioural test would need a browser;
// the rule is structural, so the test reads the source.
func TestViewerEscapesEveryInterpolation(t *testing.T) {
	for _, p := range viewerMarkupProblems(viewerScript(t, viewerHTML)) {
		t.Error(p)
	}
}

// TestViewerEscapeCheckCatchesTheOldShapes proves the check above is running:
// each of these is a shape the viewer used before the fix.
func TestViewerEscapeCheckCatchesTheOldShapes(t *testing.T) {
	for _, src := range []string{
		"el.innerHTML = `<b>${n.name}</b>`;",
		"tip.html(html);",
		"tags.push('<span class=\"tag\">entry</span>');",
		"x = \"<p>\" + n.file;",
		"setHTML(el, `<li data-id=\"${id}\">`);",
		"setHTML(el, h`<b>${`<i>${n.name}</i>`}</b>`);",
	} {
		if len(viewerMarkupProblems(src)) == 0 {
			t.Errorf("the escape check accepted %s", src)
		}
	}

	for _, src := range []string{
		"setHTML(el, h`<b>${n.name}</b>`);",
		"tip.html(piece(html));",
		"const re = /[&<>\"']/g; const s = \"a\";",
		"// a comment with <b> in it\nconst x = 1 < 2;",
		"const ok = h`<b>${a ? h`<i>x</i>` : \"\"}</b>`;",
	} {
		if p := viewerMarkupProblems(src); len(p) != 0 {
			t.Errorf("the escape check rejected %s: %v", src, p)
		}
	}
}

// viewerScript returns the viewer's own script: the last <script> element, after
// the D3 bundle and the JSON payload.
func viewerScript(t *testing.T, page string) string {
	t.Helper()

	start := strings.LastIndex(page, "<script>")
	if start < 0 {
		t.Fatal("viewer.html has no <script> element")
	}

	end := strings.Index(page[start:], "</script>")
	if end < 0 {
		t.Fatal("viewer.html's last <script> is not closed")
	}

	return page[start+len("<script>") : start+end]
}

var (
	markupRe = regexp.MustCompile(`<[A-Za-z/!]`)
	sinkRe   = regexp.MustCompile(`innerHTML|outerHTML|insertAdjacentHTML|document\.write|\.html\(`)
)

// viewerMarkupProblems tokenises JavaScript just far enough to find its string
// literals, and reports every one holding markup that is not an h`...` template,
// and every HTML sink not fed through piece().
func viewerMarkupProblems(src string) []string {
	var problems []string

	for line := range strings.SplitSeq(src, "\n") {
		code := line
		if i := strings.Index(code, "//"); i >= 0 && !strings.Contains(code[:i], "`") && !strings.Contains(code[:i], `"`) {
			code = code[:i]
		}

		if sinkRe.MatchString(code) && !strings.Contains(code, "piece(") {
			problems = append(problems, "HTML sink not fed through piece(): "+strings.TrimSpace(line))
		}
	}

	s := &jsScan{src: src}
	s.code(false)

	for _, lit := range s.lits {
		if markupRe.MatchString(lit.text) && !lit.tagged {
			problems = append(problems, "markup built outside h`...`: "+lit.text)
		}
	}

	return problems
}

type jsLit struct {
	text   string // a quoted string's body, or a template's text outside ${}
	tagged bool   // an h`...` template
}

type jsScan struct {
	src  string
	pos  int
	lits []jsLit
}

// code scans code until the end of input or, inside a ${...}, the brace closing it.
func (s *jsScan) code(inInterp bool) {
	depth := 0
	last := byte('(') // the last significant byte, to tell a regex from a division

	for s.pos < len(s.src) {
		c := s.src[s.pos]

		switch {
		case c == '/' && s.peek(1) == '/':
			for s.pos < len(s.src) && s.src[s.pos] != '\n' {
				s.pos++
			}

			continue
		case c == '/' && s.peek(1) == '*':
			end := strings.Index(s.src[s.pos+2:], "*/")
			if end < 0 {
				s.pos = len(s.src)

				return
			}

			s.pos += end + 4

			continue
		case c == '/' && strings.IndexByte("(,=:[!&|?{};+-*%<>~^", last) >= 0:
			s.regex()
		case c == '"' || c == '\'':
			s.quoted(c)
		case c == '`':
			s.template(s.taggedH())
		case c == '{':
			depth++
			s.pos++
		case c == '}':
			if inInterp && depth == 0 {
				s.pos++

				return
			}

			depth--
			s.pos++
		default:
			s.pos++
		}

		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			last = c
		}
	}
}

func (s *jsScan) peek(n int) byte {
	if s.pos+n < len(s.src) {
		return s.src[s.pos+n]
	}

	return 0
}

// taggedH reports whether the template at s.pos is tagged with the identifier h.
func (s *jsScan) taggedH() bool {
	i := s.pos - 1
	if i < 0 || s.src[i] != 'h' {
		return false
	}

	return i == 0 || !isJSIdent(s.src[i-1])
}

func isJSIdent(c byte) bool {
	return c == '_' || c == '$' || c == '.' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func (s *jsScan) regex() {
	s.pos++

	inClass := false

	for s.pos < len(s.src) {
		c := s.src[s.pos]
		s.pos++

		switch {
		case c == '\\':
			s.pos++
		case c == '[':
			inClass = true
		case c == ']':
			inClass = false
		case c == '/' && !inClass:
			return
		case c == '\n':
			return
		}
	}
}

func (s *jsScan) quoted(q byte) {
	s.pos++
	start := s.pos

	for s.pos < len(s.src) && s.src[s.pos] != q && s.src[s.pos] != '\n' {
		if s.src[s.pos] == '\\' {
			s.pos++
		}

		s.pos++
	}

	s.lits = append(s.lits, jsLit{text: s.src[start:min(s.pos, len(s.src))]})
	s.pos++
}

func (s *jsScan) template(tagged bool) {
	s.pos++

	var text strings.Builder

	for s.pos < len(s.src) {
		c := s.src[s.pos]

		switch {
		case c == '\\':
			text.WriteString(s.src[s.pos:min(s.pos+2, len(s.src))])
			s.pos += 2
		case c == '`':
			s.pos++
			s.lits = append(s.lits, jsLit{text: text.String(), tagged: tagged})

			return
		case c == '$' && s.peek(1) == '{':
			s.pos += 2

			text.WriteString("${}")
			s.code(true)
		default:
			text.WriteByte(c)

			s.pos++
		}
	}

	s.lits = append(s.lits, jsLit{text: text.String(), tagged: tagged})
}
