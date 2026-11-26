package lint

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/parser"
)

func init() {
	registerTagRules(DefaultRegistry)
}

type bannedTagRule struct {
	code     string
	tag      string
	severity string
}

var bannedTagRules = []bannedTagRule{
	{"AVOID_USING_CFDUMP_TAG", "cfdump", "WARNING"},
	{"AVOID_USING_CFABORT_TAG", "cfabort", "WARNING"},
	{"AVOID_USING_CFEXECUTE_TAG", "cfexecute", "WARNING"},
	{"AVOID_USING_CFINSERT_TAG", "cfinsert", "WARNING"},
	{"AVOID_USING_CFUPDATE_TAG", "cfupdate", "WARNING"},
	{"AVOID_USING_CFFORM_TAG", "cfform", "WARNING"},
	{"AVOID_USING_CFINPUT_TAG", "cfinput", "WARNING"},
	{"AVOID_USING_CFINVOKE_TAG", "cfinvoke", "WARNING"},
	{"AVOID_USING_CFFILE_TAG", "cffile", "WARNING"},
	{"AVOID_USING_CFDIRECTORY_TAG", "cfdirectory", "WARNING"},
	{"AVOID_USING_CFCOOKIE_TAG", "cfcookie", "WARNING"},
	{"AVOID_USING_CFHTTP_TAG", "cfhttp", "WARNING"},
	{"AVOID_USING_CFMODULE_TAG", "cfmodule", "WARNING"},
	{"AVOID_USING_CFINCLUDE_TAG", "cfinclude", "WARNING"},
}

func registerTagRules(reg *Registry) {
	for _, spec := range bannedTagRules {
		s := spec
		reg.Register(Rule{
			Code:            s.code,
			DefaultSeverity: s.severity,
			Check: func(fc *FileContext) []cflint.Issue {
				return checkBannedTag(fc, s.tag)
			},
		})
	}

	reg.Register(Rule{
		Code:            "AVOID_USING_DEBUG_ATTR",
		DefaultSeverity: "WARNING",
		Check:           checkDebugAttr,
	})
	reg.Register(Rule{
		Code:            "AVOID_USING_CFSETTING_DEBUG",
		DefaultSeverity: "WARNING",
		Check:           checkCFSettingDebug,
	})
	reg.Register(Rule{
		Code:            "ADDTOKEN_ATTR",
		DefaultSeverity: "WARNING",
		Check:           checkAddTokenAttr,
	})
	reg.Register(Rule{
		Code:            "FILE_SHOULD_START_WITH_LOWERCASE",
		DefaultSeverity: "INFO",
		Check:           checkFileStartsLowercase,
	})
	reg.Register(Rule{
		Code:            "AVOID_EMPTY_FILES",
		DefaultSeverity: "WARNING",
		Check:           checkEmptyFile,
	})
	reg.Register(Rule{
		Code:            "NEVER_USE_QUERY_IN_CFM",
		DefaultSeverity: "WARNING",
		Check:           checkQueryInCFM,
	})
}

func checkBannedTag(fc *FileContext, tagName string) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, "<"+tagName) < 0 {
		return nil
	}

	var out []cflint.Issue

	for _, tag := range parser.CFTags(src, tagName) {
		if tag.Close {
			continue
		}

		msg := fmt.Sprintf("Avoid using the %s tag.", tagName)
		out = append(out, fc.IssueAt(tag.Start, msg, tagName))
	}

	return out
}

func checkDebugAttr(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, "debug") < 0 {
		return nil
	}

	var out []cflint.Issue

	for _, tag := range parser.CFTags(src, "cfquery", "cfstoredproc", "cfldap", "cfobjectcache") {
		if tag.Close {
			continue
		}

		if val, ok := tag.Attr("debug"); ok {
			low := strings.ToLower(strings.TrimSpace(val))
			if low != "false" && low != "no" && low != "0" {
				out = append(out, fc.IssueAt(tag.Start, "Avoid using the debug attribute on <"+tag.Name+">.", "debug"))
			}
		}
	}

	return out
}

func checkCFSettingDebug(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, "<cfsetting") < 0 {
		return nil
	}

	var out []cflint.Issue

	for _, tag := range parser.CFTags(src, "cfsetting") {
		if tag.Close {
			continue
		}

		if val, ok := tag.Attr("showdebugoutput"); ok {
			low := strings.ToLower(strings.TrimSpace(val))
			if low == "true" || low == "yes" || low == "1" {
				out = append(out, fc.IssueAt(tag.Start, "Avoid using showdebugoutput=true on <cfsetting>.", "showdebugoutput"))
			}
		}
	}

	return out
}

func checkAddTokenAttr(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, "<cflocation") < 0 {
		return nil
	}

	var out []cflint.Issue

	for _, tag := range parser.CFTags(src, "cflocation") {
		if tag.Close {
			continue
		}

		if _, ok := tag.Attr("addtoken"); !ok {
			out = append(out, fc.IssueAt(tag.Start, "cflocation is missing the addtoken attribute.", "cflocation"))
		}
	}

	return out
}

func checkFileStartsLowercase(fc *FileContext) []cflint.Issue {
	if !strings.EqualFold(filepath.Ext(fc.Path), ".cfm") {
		return nil
	}

	base := filepath.Base(fc.Path)
	if strings.EqualFold(base, "Application.cfm") || strings.EqualFold(base, "OnRequestEnd.cfm") {
		return nil
	}

	if base != "" && base[0] >= 'A' && base[0] <= 'Z' {
		return []cflint.Issue{fc.IssueAt(0, fmt.Sprintf("File %s should start with a lowercase letter.", base), base)}
	}

	return nil
}

func checkEmptyFile(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	if strings.TrimSpace(src) == "" {
		return []cflint.Issue{fc.IssueAt(0, "File is empty.", filepath.Base(fc.Path))}
	}

	// Check if file contains only comments and whitespace.
	for i := 0; i < len(src); {
		switch src[i] {
		case ' ', '\t', '\r', '\n':
			i++
		case '<':
			if strings.HasPrefix(src[i:], "<!---") {
				i = parser.SkipCFMLComment(src, i)

				continue
			}

			if strings.HasPrefix(src[i:], "<!--") {
				end := strings.Index(src[i+4:], "-->")
				if end < 0 {
					return []cflint.Issue{fc.IssueAt(0, "File is empty.", filepath.Base(fc.Path))}
				}

				i += 4 + end + 3

				continue
			}

			return nil
		case '/':
			if i+1 < len(src) && src[i+1] == '/' {
				nl := strings.IndexByte(src[i:], '\n')
				if nl < 0 {
					return []cflint.Issue{fc.IssueAt(0, "File is empty.", filepath.Base(fc.Path))}
				}

				i += nl + 1

				continue
			}

			if i+1 < len(src) && src[i+1] == '*' {
				end := strings.Index(src[i+2:], "*/")
				if end < 0 {
					return []cflint.Issue{fc.IssueAt(0, "File is empty.", filepath.Base(fc.Path))}
				}

				i += 2 + end + 2

				continue
			}

			return nil
		default:
			return nil
		}
	}

	return []cflint.Issue{fc.IssueAt(0, "File is empty.", filepath.Base(fc.Path))}
}

func checkQueryInCFM(fc *FileContext) []cflint.Issue {
	if !strings.EqualFold(filepath.Ext(fc.Path), ".cfm") {
		return nil
	}

	src := fc.Text()

	var out []cflint.Issue

	if parser.IndexFold(src, "<cfquery") >= 0 {
		for _, tag := range parser.CFTags(src, "cfquery") {
			if !tag.Close {
				out = append(out, fc.IssueAt(tag.Start, "Avoid using <cfquery> in .cfm files.", "cfquery"))
			}
		}
	}

	if parser.IndexFold(src, "queryExecute") >= 0 {
		out = append(out, findScriptCalls(fc, "queryExecute", "Avoid using queryExecute() in .cfm files.")...)
	}

	return out
}
