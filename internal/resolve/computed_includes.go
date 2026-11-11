package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
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

// computedIncludeOffsets is where each computed include in s's code starts,
// in order: `<cfinclude template="#…#"`, at its "<", and a script `include
// "#…#"`, the path a string that begins and ends with an expression. A file
// with no word include has none, which spares reading the code of most
// candidate files (finding these was once 14% of a tassweb scan's CPU).
func computedIncludeOffsets(s string) []int {
	// Only a file where some include's path opens with a quoted # is read as
	// code: the word alone is in most CFML files, and reading each one's code
	// was more than a pass over a Mura scan's text.
	if parser.IndexFold(s, "include") < 0 || !mayIncludeComputed(s) {
		return nil
	}

	return computedIncludesRead(s)
}

// computedIncludesRead is computedIncludeOffsets without the checks in front.
func computedIncludesRead(s string) []int {
	var offsets []int

	computed := func(t parser.Token) bool {
		lit, ok := stringLiteral(t)

		return ok && len(lit) >= 2 && lit[0] == '#' && lit[len(lit)-1] == '#'
	}

	for _, code := range textCode(s) {
		for i := 0; i+1 < len(code); i++ {
			switch {
			case isWord(code[i], "cfinclude") && i > 0 && code[i-1].Kind == parser.TokLT && code[i-1].Offset+1 == code[i].Offset:
				if i+3 < len(code) && isWord(code[i+1], "template") && isAssignAt(code, i+2) && computed(code[i+3]) {
					offsets = append(offsets, code[i-1].Offset)
				}
			case isWord(code[i], "include") && code[i+1].Offset > code[i].Offset+len(code[i].Value) && computed(code[i+1]):
				offsets = append(offsets, code[i].Offset)
			}
		}
	}

	slices.Sort(offsets)

	return offsets
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
		var lines parser.LineTable

		for _, at := range computedIncludeOffsets(text) {
			if lines == nil {
				lines = parser.NewLineTable(text)
			}

			inc.lines = append(inc.lines, lines.LineOf(at))
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

// mayIncludeComputed reports whether s may hold an include of a computed
// path: "include" or "template", then space, an optional "=", space, and a
// quote opening with '#'. It answers yes wherever it cannot tell (a comment
// in between), so it never hides one computedIncludeOffsets would read.
func mayIncludeComputed(s string) bool {
	for _, word := range []string{"include", "template"} {
		for i := parser.IndexFold(s, word); i >= 0; i = parser.IndexFoldFrom(s, word, i+len(word)) {
			j := skipControl(s, i+len(word))
			if j < len(s) && s[j] == '=' {
				j = skipControl(s, j+1)
			}

			switch {
			case j+1 < len(s) && lex.IsQuote(s[j]) && s[j+1] == '#':
				return true
			case j < len(s) && (s[j] == '/' || s[j] == '<'):
				return true // a comment between; read the code
			}
		}
	}

	return false
}

// skipControl steps over every byte at or below a space: whatever the
// scanner takes for whitespace is among them.
func skipControl(s string, i int) int {
	for i < len(s) && s[i] <= ' ' {
		i++
	}

	return i
}
