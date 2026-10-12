package resolve

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
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

var computedIncludeRe = regexp.MustCompile(`(?i)<cfinclude\s+template\s*=\s*["']#[^"']*#["']|\binclude\s+["']#[^"']*#["']`)

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

		if !hasFoldASCII(inc.content, suffix+`"`) && !hasFoldASCII(inc.content, suffix+`'`) && !hasFoldASCII(inc.content, suffix+`/"`) {
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

	if data, err := r.fs().ReadFile(path); err == nil {
		for _, m := range computedIncludeRe.FindAllIndex(data, -1) {
			inc.lines = append(inc.lines, bytes.Count(data[:m[0]], []byte("\n")))
		}

		if len(inc.lines) > 0 {
			inc.content = string(data)
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

// hasFoldASCII reports whether s holds lower, an ASCII lowercase string, in
// any case, without lowercasing s: the search for a needle the code spells.
func hasFoldASCII[T string | []byte](s T, lower string) bool {
	for i := 0; i+len(lower) <= len(s); i++ {
		j := 0

		for j < len(lower) {
			c := s[i+j]
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}

			if c != lower[j] {
				break
			}

			j++
		}

		if j == len(lower) {
			return true
		}
	}

	return false
}
