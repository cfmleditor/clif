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

		lines, data := r.computedIncludeLines(path)
		if len(lines) == 0 {
			continue
		}

		if !containsFoldBytes(data, suffix+`"`) && !containsFoldBytes(data, suffix+`'`) && !containsFoldBytes(data, suffix+`/"`) {
			continue
		}

		for _, line := range lines {
			hosts = append(hosts, includeHost{path: path, line: line})
		}
	}

	return hosts
}

// computedIncludeLines is the 0-based lines of path's computed includes,
// read once per scan, and the bytes when it has any. Most candidates have
// none, and each template used to read and lowercase every one of them,
// which was a tenth of tassweb's heap at its peak.
func (r *Resolver) computedIncludeLines(path string) ([]int, []byte) {
	o := r.owner()

	o.mu.RLock()
	lines, ok := o.computedIncludes[path]
	o.mu.RUnlock()

	if ok && len(lines) == 0 {
		return nil, nil
	}

	data, err := r.fs().ReadFile(path)
	if err != nil {
		return nil, nil
	}

	if ok {
		return lines, data
	}

	lines = []int{}
	for _, m := range computedIncludeRe.FindAllIndex(data, -1) {
		lines = append(lines, bytes.Count(data[:m[0]], []byte("\n")))
	}

	o.mu.Lock()
	if o.computedIncludes == nil {
		o.computedIncludes = map[string][]int{}
	}

	o.computedIncludes[path] = lines
	o.mu.Unlock()

	return lines, data
}

// containsFoldBytes reports whether data holds lower, an ASCII lowercase
// string, in any case, without lowercasing data.
func containsFoldBytes(data []byte, lower string) bool {
	if lower == "" {
		return true
	}

	first, upper := lower[0], lower[0]
	if first >= 'a' && first <= 'z' {
		upper = first - ('a' - 'A')
	}

	for i := 0; i+len(lower) <= len(data); i++ {
		if c := data[i]; c != first && c != upper {
			continue
		}

		if bytes.EqualFold(data[i:i+len(lower)], []byte(lower)) {
			return true
		}
	}

	return false
}

// containsFoldString is containsFoldBytes for a string.
func containsFoldString(s, lower string) bool {
	if lower == "" {
		return true
	}

	first, upper := lower[0], lower[0]
	if first >= 'a' && first <= 'z' {
		upper = first - ('a' - 'A')
	}

	for i := 0; i+len(lower) <= len(s); i++ {
		if c := s[i]; c != first && c != upper {
			continue
		}

		if strings.EqualFold(s[i:i+len(lower)], lower) {
			return true
		}
	}

	return false
}
