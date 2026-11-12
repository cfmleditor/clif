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

	// Lookups that ask for a text together wait on one scan of it: they did
	// each scan it, which on a parallel MasaCMS scan was most of the scans.
	appScanMu.Lock()

	entry, ok := appScans[key]
	if !ok {
		if len(appScans) >= maxAppScans {
			clear(appScans)
		}

		entry = &appScanEntry{}
		appScans[key] = entry
	}
	appScanMu.Unlock()

	entry.once.Do(func() { entry.result = scanApplicationSpans(content) })

	return blankSpans(content, entry.result.blanked), entry.result.sections
}

// appScanEntry is one text's scan, made once by whichever lookup asks first.
type appScanEntry struct {
	once   sync.Once
	result appScanResult
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
	appScans  = map[TextFingerprint]*appScanEntry{}
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

// scanApplicationSpans is applicationScan's work: the functions the parse
// finds in content (FunctionSpansOf, its functions-only reading), each blanked
// from its first header word to its end, and each a section but a function
// assigned to a name, which is a value the constructor sets rather than a
// method. A function nested in another is inside that one's span, and no
// section either: an application's methods are its own, declared at the top.
func scanApplicationSpans(content string) appScanResult {
	var held appScanResult

	spans := FunctionSpansOf("", content)
	for i := range spans {
		s := &spans[i]
		if s.Kind == FunctionNested || s.End <= s.Start {
			continue
		}

		held.blanked = append(held.blanked, [2]int{s.Start, s.End})

		if s.Kind != FunctionValue && s.Name != "" {
			held.sections = append(held.sections, appSection{name: strings.Clone(strings.ToLower(s.Name)), start: s.Start, end: s.End})
		}
	}

	return held
}
