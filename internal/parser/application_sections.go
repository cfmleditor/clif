package parser

import (
	"slices"
	"strings"
	"sync"
)

// ApplicationSections separates executable constructor text from declarations.
// Byte positions and newlines are retained in each projection. The method map
// is case insensitive; duplicate declarations are withheld rather than selected.
func ApplicationSections(content string) (constructor string, methods map[string]string) {
	constructor, sections := applicationScan(content)
	methods = map[string]string{}
	duplicate := map[string]bool{}

	for _, sec := range sections {
		if _, exists := methods[sec.name]; exists || duplicate[sec.name] {
			methods[sec.name] = ""
			duplicate[sec.name] = true
		} else {
			methods[sec.name] = sec.projection(content)
		}
	}

	return constructor, methods
}

// ApplicationConstructor is ApplicationSections' constructor alone. Most
// callers want only it, and ApplicationSections builds a copy of the whole
// file for every function it finds (94% of a FW/1 scan's peak heap).
func ApplicationConstructor(content string) string {
	constructor, _ := applicationScan(content)

	return constructor
}

// ApplicationMethod is ApplicationSections' methods[name], building only
// that method's projection: "" and true when the name is declared twice.
func ApplicationMethod(content, name string) (string, bool) {
	_, sections := applicationScan(content)

	return methodProjection(content, sections, name)
}

// methodProjection is the projection of the one section named name: "" and
// true when it is declared twice, false when not at all.
func methodProjection(content string, sections []appSection, name string) (string, bool) {
	name = strings.ToLower(name)
	found := -1

	for i := range sections {
		if sections[i].name != name {
			continue
		}

		if found >= 0 {
			return "", true
		}

		found = i
	}

	if found < 0 {
		return "", false
	}

	return sections[found].projection(content), true
}

// ApplicationConstructorAndMethod is ApplicationConstructor and
// ApplicationMethod from one scan, for a caller that needs both.
func ApplicationConstructorAndMethod(content, name string) (constructor, method string, ok bool) {
	constructor, sections := applicationScan(content)
	method, ok = methodProjection(content, sections, name)

	return constructor, method, ok
}

// ApplicationMethodNames is the names ApplicationSections' methods has, each
// once, without building any projection.
func ApplicationMethodNames(content string) []string {
	_, sections := applicationScan(content)
	names := make([]string, 0, len(sections))

	for _, sec := range sections {
		if !slices.Contains(names, sec.name) {
			names = append(names, sec.name)
		}
	}

	return names
}

// appSection is one named function an application declares, by byte span.
type appSection struct {
	name       string
	start, end int
}

// projection is content with everything outside the section blanked but its
// line breaks, so lines and offsets still read as the file's.
func (s appSection) projection(content string) string {
	var b strings.Builder

	b.Grow(len(content))

	blank := func(text string) {
		for i := range len(text) {
			if c := text[i]; c == '\n' || c == '\r' {
				b.WriteByte(c)
			} else {
				b.WriteByte(' ')
			}
		}
	}

	blank(content[:s.start])
	b.WriteString(content[s.start:s.end])
	blank(content[s.end:])

	return b.String()
}

// applicationScan is content with every function blanked (the constructor),
// and the named functions it found, in order. It is asked of the same text
// by many lookups, several of them twice for one file (the constructor, then
// the sections), so the answers are kept, by the text's fingerprint and as
// spans: an entry holds no copy of the file, and the constructor is built
// again from its spans by one byte copy, which is no scan. What it returns is
// shared and must not change.
func applicationScan(content string) (string, []appSection) {
	key := FingerprintOf(content)

	appScanMu.Lock()
	held, ok := appScans[key]
	appScanMu.Unlock()

	if !ok {
		held = scanApplicationSpans(content)

		appScanMu.Lock()
		if len(appScans) >= maxAppScans {
			clear(appScans)
		}

		appScans[key] = held
		appScanMu.Unlock()
	}

	return blankSpans(content, held.blanked), held.sections
}

// maxAppScans bounds applicationScan's memo, a count of files cleared when
// reached.
const maxAppScans = 1024

type appScanResult struct {
	blanked  [][2]int // every function's span, named or not
	sections []appSection
}

var (
	appScanMu sync.Mutex
	appScans  = map[TextFingerprint]appScanResult{}
)

// blankSpans is content with each span blanked but its line breaks: content
// itself, uncopied, when there is none.
func blankSpans(content string, spans [][2]int) string {
	if len(spans) == 0 {
		return content
	}

	masked := []byte(content)

	for _, sp := range spans {
		for i := sp[0]; i < sp[1]; i++ {
			if masked[i] != '\n' && masked[i] != '\r' {
				masked[i] = ' '
			}
		}
	}

	return string(masked)
}

// scanApplicationSpans is scanApplication's answer as spans, its names copied
// out of content.
func scanApplicationSpans(content string) appScanResult {
	spans, sections := scanApplication(content)

	for i := range sections {
		sections[i].name = strings.Clone(sections[i].name)
	}

	return appScanResult{blanked: spans, sections: sections}
}

// scanApplication is applicationScan's work.
func scanApplication(content string) ([][2]int, []appSection) {
	var (
		spans    [][2]int
		sections []appSection
	)

	// Functions are found by their braces, so strings are read as the parse
	// reads them where the text is CFScript (NewScriptScanner).
	sc := NewScanner(content)
	sc.interpStrings = isScriptFile(content)

	var header []Token

	for {
		t := sc.NextSkipComments()
		if t.Kind == TokEOF {
			break
		}

		if t.Kind != TokIdent {
			if t.Kind == TokDot && len(header) > 0 {
				header = append(header, t)
			} else {
				header = nil
			}

			continue
		}

		start, name := t.Offset, ""

		var end int

		switch {
		case identEq(t.Value, "function"):
			if applicationFunctionHeader(header) {
				start = header[0].Offset
			}

			if next := sc.PeekSkipComments(); next.Kind == TokIdent {
				name = strings.ToLower(next.Value)
			}

			skipMappingFunction(sc)
			end = sc.pos
		case identEq(t.Value, "cffunction"):
			start = max(0, start-1)

			name = applicationTagMethod(sc)

			end = sc.pos
		default:
			if len(header) < 16 {
				header = append(header, t)
			} else {
				header = nil
			}

			continue
		}

		header = nil

		if end > start {
			spans = append(spans, [2]int{start, end})
		}

		if name != "" {
			sections = append(sections, appSection{name: name, start: start, end: end})
		}
	}

	return spans, sections
}

// Access/modifier and return-type tokens are part of a declaration, not
// executable constructor text. Keep arbitrary qualified component return types
// while rejecting statement keywords as a purported return type.
func applicationFunctionHeader(tokens []Token) bool {
	if len(tokens) == 0 {
		return false
	}

	start := 0
	for start < len(tokens) {
		switch strings.ToLower(tokens[start].Value) {
		case "public", "private", "package", "remote", "static", "final", "abstract":
			start++
		default:
			goto returnType
		}
	}

	return true

returnType:
	for i := start; i < len(tokens); i++ {
		if (i-start)%2 == 0 {
			if tokens[i].Kind != TokIdent {
				return false
			}
		} else if tokens[i].Kind != TokDot {
			return false
		}
	}

	switch strings.ToLower(tokens[start].Value) {
	case "return", "var", "new", "component", "include", "import", "throw":
		return false
	}

	return (len(tokens)-start)%2 == 1
}

func applicationTagMethod(sc *Scanner) (name string) {
	for {
		a := sc.NextSkipComments()
		if a.Kind == TokEOF || a.Kind == TokGT {
			break
		}

		if identEq(a.Value, "name") && sc.NextSkipComments().Kind == TokEquals {
			value := sc.NextSkipComments()
			if value.Kind == TokString {
				name = strings.ToLower(strings.Trim(value.Value, `"'`))
			}
		}
	}

	for {
		a := sc.NextSkipComments()
		if a.Kind == TokEOF {
			break
		}

		if a.Kind == TokSlash && identEq(sc.PeekSkipComments().Value, "cffunction") {
			sc.NextSkipComments()

			if sc.PeekSkipComments().Kind == TokGT {
				sc.NextSkipComments()
			}

			break
		}
	}

	return name
}
