package resolve

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// A custom tag's attributes scope is what the tag calling it passes:
//
//	<cf_fileselector name="newfile" bean="#rc.contentBean#" …>
//
// MasaCMS's admin calls its file picker this way, and the picker's
// attributes.bean was typed nowhere. customTagAttribute reads every
// `<cf_name …>`, and `<cfmodule template="…/name.cfm" …>`, in the workspace
// and types an attribute given as one `#expr#` where the call is made, as
// viewArgs does a ColdBox view's args; the calls are the alternatives, and a
// tag handing an attribute on (`<cf_filetools bean="#attributes.bean#">`)
// passes what it holds. Which file a tag name runs is decided at run time
// (customTagPaths, the caller's directory), so a name two templates in the
// workspace answer to is not claimed. A call passing the attribute as text,
// or an attributeCollection, leaves it untyped, as does a tag that sets the
// attribute itself; a call omitting it says nothing, and a cfparam default
// fills only what was not passed. The calls are found in the batch file list,
// so this runs in a batch scan only.

// customTagSite is one call of a custom tag: where it is, the attributes it
// passes by lowercased name with their raw values, and opaque when it passes
// an attributeCollection.
type customTagSite struct {
	file   string
	line   int
	attrs  map[string]string
	opaque bool
}

var (
	customTagStartRe = regexp.MustCompile(`(?i)<cf_(\w+)\b|<cfmodule\b`)
	tagAttributeRe   = regexp.MustCompile(`(?s)([\w:.-]+)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	wholeHashExprRe  = regexp.MustCompile(`(?s)^\s*#([^#]+)#\s*$`)
	cfparamTagRe     = regexp.MustCompile(`(?is)<cfparam\b[^>]*>`)
)

// customTagAttribute is the component the custom tag pr holds as variable
// (`attributes.x`), as the tags calling it pass it, or "".
func (r *Resolver) customTagAttribute(variable string, pr *parser.ParseResult, tr *callTrace, ctx lookupCtx) string {
	rest, ok := strings.CutPrefix(strings.ToLower(variable), "attributes.")
	if !ok || rest == "" || strings.ContainsAny(rest, ".[(") || pr == nil || !pr.URI.IsFile() || ctx.depth >= maxAssignedDepth || len(r.owner().InferArgsFiles) == 0 {
		return ""
	}

	file := cfpath.FromURI(string(pr.URI))
	if !strings.EqualFold(filepath.Ext(file), ".cfm") {
		return ""
	}

	tag := strings.ToLower(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)))

	sites, templates := r.customTagSites()
	if templates[tag] != 1 || len(sites[tag]) == 0 {
		return ""
	}

	// A tag that sets the attribute reads its own value; a cfparam only
	// fills one that was not passed.
	if setsName(cfparamTagRe.ReplaceAllString(pr.Content, ""), "attributes."+rest) {
		return ""
	}

	var comps []string

	for i := range sites[tag] {
		s := &sites[tag][i]
		if s.opaque {
			return ""
		}

		raw, passed := s.attrs[rest]
		if !passed {
			continue
		}

		m := wholeHashExprRe.FindStringSubmatch(raw)
		if m == nil {
			return ""
		}

		comp := r.renderedValueD(significantTokens(m[1]), viewArgSite{file: s.file, line: s.line}, lookupCtx{depth: ctx.depth, callers: ctx.callers})
		if comp == "" {
			return ""
		}

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
	tr.addf("%q is what the tags calling this one pass it: %q", variable, answer)

	return answer
}

// customTagSites are the workspace's custom tag calls by tag name, and how
// many templates answer to each name, read once per scan from the batch file
// list. One goroutine builds it while the others wait: every scan worker asks
// at once, and each reading the whole workspace for itself took 5s of tassweb's
// scan. The files are read in parallel.
func (r *Resolver) customTagSites() (map[string][]customTagSite, map[string]int) {
	o := r.owner()

	if sites, templates, ok := o.cachedTagSites(); ok {
		return sites, templates
	}

	o.tagSitesBuild.Lock()
	defer o.tagSitesBuild.Unlock()

	if sites, templates, ok := o.cachedTagSites(); ok {
		return sites, templates
	}

	files := o.InferArgsFiles
	found := make([]map[string][]customTagSite, len(files))

	failed := forEachFile(files, func(i int, file string) { found[i] = r.customTagCallsInFile(file) })

	sites, templates := map[string][]customTagSite{}, map[string]int{}

	if len(failed) > 0 {
		// A file whose calls could not be read may call any tag, so no
		// tag's callers are known: every attribute declines, and no
		// include is read from them.
		found = nil
	}

	for i, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".cfm") {
			templates[strings.ToLower(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)))]++
		}

		if found == nil {
			continue
		}

		for tag, calls := range found[i] {
			sites[tag] = append(sites[tag], calls...)
		}
	}

	o.mu.Lock()
	o.tagSites, o.tagTemplates = sites, templates
	o.mu.Unlock()

	return sites, templates
}

// customTagCallsInFile is customTagCallsIn over one file, nil when it holds
// none.
func (r *Resolver) customTagCallsInFile(file string) map[string][]customTagSite {
	text, err := r.readText(file)
	if err != nil || parser.IndexFold(text, "<cf_") < 0 && parser.IndexFold(text, "<cfmodule") < 0 {
		return nil
	}

	return customTagCallsIn(text, file)
}

func (r *Resolver) cachedTagSites() (map[string][]customTagSite, map[string]int, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.tagSites, r.tagTemplates, r.tagSites != nil
}

// customTagCallsIn reads the custom tag calls in content: `<cf_name …>`, and
// `<cfmodule template="…/name.cfm" …>`.
func customTagCallsIn(content, file string) map[string][]customTagSite {
	out := map[string][]customTagSite{}
	lines := newLineIndex(content)

	for _, m := range customTagStartRe.FindAllStringSubmatchIndex(content, -1) {
		end := quotedTagEnd(content, m[1])
		if end < 0 {
			continue
		}

		site := customTagSite{file: file, line: lines.lineOf(m[0]), attrs: map[string]string{}}

		for _, a := range tagAttributeRe.FindAllStringSubmatch(content[m[1]:end], -1) {
			value := a[2]
			if value == "" {
				value = a[3]
			}

			key := strings.ToLower(a[1])
			if key == "attributecollection" {
				site.opaque = true
			}

			site.attrs[key] = value
		}

		tag := ""
		if m[2] >= 0 {
			tag = strings.ToLower(content[m[2]:m[3]])
		} else if template := site.attrs["template"]; strings.EqualFold(filepath.Ext(template), ".cfm") && !strings.Contains(template, "#") {
			tag = strings.ToLower(strings.TrimSuffix(filepath.Base(filepath.FromSlash(template)), filepath.Ext(template)))
		}

		if tag != "" {
			out[tag] = append(out[tag], site)
		}
	}

	return out
}

// quotedTagEnd is the index of the `>` ending the tag whose name ends at
// from, stepping over quoted attribute values, or -1.
func quotedTagEnd(content string, from int) int {
	var quote byte

	for i := from; i < len(content); i++ {
		c := content[i]

		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '>':
			return i
		}
	}

	return -1
}

// attributeIncludeTargets is what `<cfinclude template="#attributes.name#">`
// in the custom tag tagFile includes: the template each call of the tag
// passes as name, written as a literal and resolved as a cfinclude in the tag
// is. RustCFML's test runner calls `<cf_runtest file="core/test_x.cfm">` six
// hundred times, and runtest.cfm includes its harness and then the file, so
// each test calls the harness's assert() bare. A call passing the attribute
// computed, or an attributeCollection, names nothing it can be read from, and
// a tag name two templates answer to is not claimed, as customTagAttribute
// does not. The calls are found in the batch file list, so this runs in a
// batch scan only.
func (r *Resolver) attributeIncludeTargets(name, tagFile string) []string {
	if len(r.owner().InferArgsFiles) == 0 || !strings.EqualFold(filepath.Ext(tagFile), ".cfm") {
		return nil
	}

	tag := strings.ToLower(strings.TrimSuffix(filepath.Base(tagFile), filepath.Ext(tagFile)))

	sites, templates := r.customTagSites()
	if templates[tag] != 1 {
		return nil
	}

	var out []string

	for i := range sites[tag] {
		raw, passed := sites[tag][i].attrs[name]
		if !passed || strings.Contains(raw, "#") {
			continue
		}

		if target := r.IncludePath(raw, tagFile); target != "" && !slices.Contains(out, target) {
			out = append(out, target)
		}
	}

	slices.Sort(out)

	return out
}
