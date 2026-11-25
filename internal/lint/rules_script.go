package lint

import (
	"fmt"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/parser"
)

func init() {
	registerScriptRules(DefaultRegistry)
}

type bannedCallRule struct {
	code     string
	fnName   string
	severity string
}

var bannedCallRules = []bannedCallRule{
	{"AVOID_USING_WRITEDUMP", "writeDump", "WARNING"},
	{"AVOID_USING_ISDEBUGMODE", "isDebugMode", "WARNING"},
	{"AVOID_USING_CREATEOBJECT", "createObject", "INFO"},
	{"AVOID_USING_EVALUATE", "evaluate", "WARNING"},
	{"AVOID_USING_ISDEFINED", "isDefined", "WARNING"},
}

func registerScriptRules(reg *Registry) {
	for _, spec := range bannedCallRules {
		s := spec
		reg.Register(Rule{
			Code:            s.code,
			DefaultSeverity: s.severity,
			Check: func(fc *FileContext) []cflint.Issue {
				msg := fmt.Sprintf("Avoid using %s().", s.fnName)

				return findScriptCalls(fc, s.fnName, msg)
			},
		})
	}

	reg.Register(Rule{
		Code:            "AVOID_USING_ABORT",
		DefaultSeverity: "WARNING",
		Check:           checkScriptAbort,
	})
}

// findScriptCalls scans script regions (and <cfset>/<cfif>/#...# expressions
// inside tag regions) for bare calls to fnName (not member calls like obj.fnName).
func findScriptCalls(fc *FileContext, fnName, msg string) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, fnName) < 0 {
		return nil
	}

	var out []cflint.Issue

	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript {
			continue
		}

		out = append(out, scanScriptTextForCall(fc, r.Text, r.Offset, fnName, msg)...)
	}

	// Also check <cfset>, <cfif>, <cfelseif>, <cfreturn> tag expressions.
	for _, tag := range parser.CFTags(src, "cfset", "cfif", "cfelseif", "cfreturn") {
		if tag.Close {
			continue
		}

		body := tag.Text()
		if len(body) <= len(tag.Name)+2 {
			continue
		}

		innerStart := len(tag.Name) + 1
		inner := body[innerStart : len(body)-1]
		out = append(out, scanScriptTextForCall(fc, inner, tag.Start+innerStart, fnName, msg)...)
	}

	return out
}

func scanScriptTextForCall(fc *FileContext, text string, baseOffset int, fnName, msg string) []cflint.Issue {
	if parser.IndexFold(text, fnName) < 0 {
		return nil
	}

	sc := parser.NewSourceScanner(text)

	var (
		prev parser.Token
		out  []cflint.Issue
	)

	cur := sc.NextSkipComments()
	for cur.Kind != parser.TokEOF {
		next := sc.NextSkipComments()

		if cur.Kind == parser.TokIdent && strings.EqualFold(cur.Value, fnName) && next.Kind == parser.TokLParen {
			// Skip member calls (`obj.fnName(`) and function declarations (`function fnName(`).
			isMember := prev.Kind == parser.TokDot
			isDecl := prev.Kind == parser.TokIdent && strings.EqualFold(prev.Value, "function")

			if !isMember && !isDecl {
				out = append(out, fc.IssueAt(baseOffset+cur.Offset, msg, cur.Value))
			}
		}

		prev = cur
		cur = next
	}

	return out
}

func checkScriptAbort(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, "abort") < 0 {
		return nil
	}

	var out []cflint.Issue

	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript || parser.IndexFold(r.Text, "abort") < 0 {
			continue
		}

		sc := parser.NewSourceScanner(r.Text)

		var prev parser.Token

		cur := sc.NextSkipComments()
		for cur.Kind != parser.TokEOF {
			next := sc.NextSkipComments()

			if cur.Kind == parser.TokIdent && strings.EqualFold(cur.Value, "abort") {
				// Statement `abort;` or `abort "msg";` or `abort showerror="...";`
				// Exclude member access (`x.abort`), struct key (`abort:`), assignment (`abort =`),
				// function decl (`function abort`), or call (`abort()`).
				atStmtStart := prev.Kind == 0 ||
					prev.Kind == parser.TokSemicolon ||
					prev.Kind == parser.TokLBrace ||
					prev.Kind == parser.TokRBrace ||
					prev.Kind == parser.TokRParen ||
					prev.Kind == parser.TokColon ||
					(prev.Kind == parser.TokIdent && strings.EqualFold(prev.Value, "else"))

				notExpr := next.Kind != parser.TokColon &&
					next.Kind != parser.TokEquals &&
					next.Kind != parser.TokLParen &&
					next.Kind != parser.TokDot

				if atStmtStart && notExpr {
					out = append(out, fc.IssueAt(r.Offset+cur.Offset, "Avoid using the abort statement.", "abort"))
				}
			}

			prev = cur
			cur = next
		}
	}

	return out
}
