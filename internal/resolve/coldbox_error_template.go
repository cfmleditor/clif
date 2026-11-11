package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// ColdBox renders an application's error page by including the template its
// config names as customErrorTemplate, from inside Bootstrap's
// processException(), where `var oException = new ExceptionBean( … )` is the
// exception the page reads. The include is computed (`include
// "#bugReportRelativePath#"`), so the include graph never sees it, and every
// oException call in ColdBox's own Whoops.cfm and BugReport.cfm was "has no
// component ref".
//
// coldboxErrorHosts is that include, for a template some config/ColdBox.cfc
// names: Bootstrap's own site, checked by its text and by the function it sits
// in. It needs ColdBox's source; a stub has no body to read.

// customErrorTemplateValues are the templates text, a config, sets
// customErrorTemplate to: `customErrorTemplate : "/path"` or `= "/path"`,
// written in code, with a literal holding no expression.
func customErrorTemplateValues(text string) []string {
	var out []string

	for _, code := range textCode(text) {
		for i := 0; i+2 < len(code); i++ {
			if !isWord(code[i], "customErrorTemplate") || code[i+1].Kind != parser.TokColon && !isAssignAt(code, i+1) {
				continue
			}

			if lit, ok := stringLiteral(code[i+2]); ok && lit != "" && !strings.ContainsAny(lit, "#\"'") {
				out = append(out, lit)
			}
		}
	}

	return out
}

const coldboxErrorInclude = `include "#bugReportRelativePath#"`

type includeHost struct {
	path string
	line int
}

func (r *Resolver) coldboxErrorHosts(file string) []includeHost {
	if r.Index == nil {
		return nil
	}

	var hosts []includeHost

	for _, config := range r.Index.FindFilesByBasename("ColdBox") {
		if !strings.EqualFold(filepath.Base(filepath.Dir(config)), "config") {
			continue
		}

		text, err := r.readText(config)
		if err != nil || !mentions(text, "customErrorTemplate") {
			continue
		}

		named := false

		for _, value := range customErrorTemplateValues(text) {
			if cfpath.SamePath(r.IncludePath(value, config), file) {
				named = true
			}
		}

		if !named {
			continue
		}

		if host, ok := r.bootstrapErrorSite(filepath.Dir(config)); ok && !containsHost(hosts, host) {
			hosts = append(hosts, host)
		}
	}

	return hosts
}

// bootstrapErrorSite is the line of Bootstrap's processException() that
// includes the custom error template, read from dir.
func (r *Resolver) bootstrapErrorSite(dir string) (includeHost, bool) {
	path := r.ComponentPath("coldbox.system.Bootstrap", dir)
	if path == "" {
		return includeHost{}, false
	}

	hpr := r.handlerParse(path)
	if hpr == nil {
		return includeHost{}, false
	}

	at := strings.Index(hpr.Content, coldboxErrorInclude)
	if at < 0 {
		return includeHost{}, false
	}

	line := strings.Count(hpr.Content[:at], "\n")
	if !strings.EqualFold(hpr.FuncScopeAt(line).Name, "processException") {
		return includeHost{}, false
	}

	return includeHost{path: path, line: line}, true
}

func containsHost(hosts []includeHost, h includeHost) bool {
	for _, x := range hosts {
		if cfpath.SamePath(x.path, h.path) && x.line == h.line {
			return true
		}
	}

	return false
}
