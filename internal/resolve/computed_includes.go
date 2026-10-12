package resolve

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// A template may be included by a computed path: Masa's form builder keeps
// `variables.templatePath = "/muraWRM#…#/core/utilities/formbuilder/templates"`
// and includes `<cfinclude template="#templatePath#">` from functions that
// set `var mmRBF = application.rbFactory` first. The include graph cannot see
// that edge, and every mmRBF call in the 30-odd field templates was "has no
// component ref".
//
// computedIncludeHosts treats a file as an includer of a template when it
// holds both a computed include and a string literal naming the template's
// directory by at least its last minDirSegments segments; each of its computed
// includes is a site, and includerHeld requires every site to agree, as it does
// for a literal include. Candidate files come from the batch caller index (a
// name ending a quoted string), so this runs in a batch scan only.

const minDirSegments = 3

// computedIncludeAt matches the two forms of a computed include,
// `<cfinclude template="#…#"` and a script `include "#…#"`, where one starts at
// the beginning of its input: computedIncludeOffsets runs it only where the
// word include is, rather than running computedIncludeRe over every byte of
// every candidate file (14% of a tassweb scan's CPU).
var computedIncludeAt = regexp.MustCompile(`^(?i:<cfinclude\s+template\s*=\s*["']#[^"']*#["']|include\s+["']#[^"']*#["'])`)

// computedIncludeOffsets is the start of each computedIncludeRe match in s, as
// FindAllStringIndex gives them: no match starts inside an earlier one.
func computedIncludeOffsets(s string) []int {
	var offsets []int

	for at, end := 0, 0; ; {
		i := parser.IndexFoldFrom(s, "include", at)
		if i < 0 {
			return offsets
		}

		at = i + len("include")

		// <cfinclude: the match starts at its "<".
		start := i - 3
		if start >= end && i >= 3 && strings.EqualFold(s[start:i], "<cf") {
			if m := computedIncludeAt.FindStringIndex(s[start:]); m != nil {
				offsets, end = append(offsets, start), start+m[1]

				continue
			}
		}

		// A bare include must start a word (\b): the character before it is
		// not a word character.
		if i >= end && (i == 0 || !isRegexWordByte(s[i-1])) {
			if m := computedIncludeAt.FindStringIndex(s[i:]); m != nil {
				offsets, end = append(offsets, i), i+m[1]
			}
		}
	}
}

// isRegexWordByte is \w as RE2 reads it.
func isRegexWordByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func (r *Resolver) computedIncludeHosts(file string) []includeHost {
	if len(r.owner().InferArgsFiles) == 0 {
		return nil
	}

	segs := strings.Split(filepath.ToSlash(filepath.Dir(file)), "/")
	if len(segs) < minDirSegments {
		return nil
	}

	suffix := strings.ToLower(strings.Join(segs[len(segs)-minDirSegments:], "/"))
	last := segs[len(segs)-1]

	var hosts []includeHost

	for _, path := range r.callerFiles(quotedCallerKey(last)) {
		if strings.EqualFold(filepath.Clean(path), filepath.Clean(file)) {
			continue
		}

		inc := r.computedIncludesOf(path)
		if len(inc.lines) == 0 {
			continue
		}

		if parser.IndexFold(inc.content, suffix+`"`) < 0 && parser.IndexFold(inc.content, suffix+`'`) < 0 && parser.IndexFold(inc.content, suffix+`/"`) < 0 {
			continue
		}

		for _, line := range inc.lines {
			hosts = append(hosts, includeHost{path: path, line: line})
		}
	}

	return hosts
}

// computedIncludeFile is a file's computed includes: their 0-based lines, and
// the file's text when it has any, for the directory test each template asks.
type computedIncludeFile struct {
	lines   []int
	content string
}

// computedIncludesOf is path's computed includes, read once per scan. Most
// candidates have none and keep nothing; each template used to read and
// lowercase every one of them, which was a tenth of tassweb's heap at its
// peak.
func (r *Resolver) computedIncludesOf(path string) computedIncludeFile {
	o := r.owner()

	o.mu.RLock()
	inc, ok := o.computedIncludes[path]
	o.mu.RUnlock()

	if ok {
		return inc
	}

	if text, err := r.readText(path); err == nil {
		for _, at := range computedIncludeOffsets(text) {
			inc.lines = append(inc.lines, strings.Count(text[:at], "\n"))
		}

		if len(inc.lines) > 0 {
			inc.content = text
		}
	}

	o.mu.Lock()
	if o.computedIncludes == nil {
		o.computedIncludes = map[string]computedIncludeFile{}
	}

	o.computedIncludes[path] = inc
	o.mu.Unlock()

	return inc
}
