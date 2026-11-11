package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// WheelsTest's pseudo-constructor and init bind public Global UDFs without
// replacing existing methods. The known default binder supplies the identity
// of application.wo; unrelated test bases and renamed/overridden loaders do not.
func (r *Resolver) wheelsTestGlobalFunc(chain []string, name string) *parser.FunctionDef {
	for at, path := range chain {
		if !strings.EqualFold(filepath.Base(path), "WheelsTest.cfc") {
			continue
		}

		for _, child := range chain[:at] {
			if _, overridden := r.sourceMethods(child)["$bindapplicationhelpers"]; overridden {
				return nil
			}
		}

		source := r.sourceInfo(path)
		if !wheelsTestBinds(&source) {
			continue
		}

		bindings := r.ComponentPath("wheels.Bindings", filepath.Dir(path))

		text, err := r.readText(bindings)
		if err != nil {
			return nil
		}

		tokens := wheelsTokens(text)
		if strings.Count(tokens, wheelsTokens(`map("global")`)) != 1 || !strings.Contains(tokens, wheelsTokens(`map("global").to("wheels.Global")`)) {
			return nil
		}

		global := r.ComponentPath("wheels.Global", filepath.Dir(path))
		if global == "" || !samePath(filepath.Dir(global), filepath.Dir(path)) {
			return nil
		}

		return r.wheelsPlanFunc(global, name)
	}

	return nil
}

func (r *Resolver) wheelsTestType(path string) bool {
	var chain []string

	seen := make(map[string]bool)
	for len(chain) < 16 && path != "" && !seen[path] {
		seen[path] = true
		chain = append(chain, path)

		next, ok := r.extendsOf(path, cfpath.ToURI(path))
		if !ok {
			break
		}

		path = r.ComponentPath(next, filepath.Dir(path))
	}

	return r.wheelsTestGlobalFunc(chain, "$createObjectFromRoot") != nil
}

const wheelsTestBinder = `
 if (!structKeyExists(application, "wo")) {return this;}
 local.metaIndex = {};
 for (local.fn in getMetaData(application.wo).functions) {local.metaIndex[local.fn.name] = local.fn.access;}
 for (local.key in application.wo) {
  if (!isCustomFunction(application.wo[local.key])) {continue;}
  if (structKeyExists(local.metaIndex, local.key) && local.metaIndex[local.key] neq "public") {continue;}
  if (structKeyExists(variables, local.key) || structKeyExists(this, local.key)) {continue;}
  variables[local.key] = application.wo[local.key];
  this[local.key] = application.wo[local.key];
 }
 return this;`

// Both shipped layouts copy the same public UDFs. The inline layout must be
// the pseudo-constructor itself, followed only by named method declarations;
// a matching string, closure or unused method is not startup evidence.
func wheelsTestBinds(source *sourceInfo) bool {
	if source.methodMap()["init"].body == wheelsTokens(`$bindApplicationHelpers();return this;`) && source.methodMap()["$bindapplicationhelpers"].body == wheelsTokens(wheelsTestBinder) {
		return true
	}

	scanner := parser.NewScanner(source.content)
	if t := scanner.NextSkipComments(); t.Kind != parser.TokIdent || !strings.EqualFold(t.Value, "component") {
		return false
	}

	for {
		t := scanner.NextSkipComments()
		if t.Kind == parser.TokEOF {
			return false
		}

		if t.Kind == parser.TokLBrace {
			break
		}
	}

	pos := 0

	for _, expected := range significantTokens(wheelsTestInlineBinder) {
		actual := scanner.NextSkipComments()
		if wheelsTokens(actual.Value) != wheelsTokens(expected.Value) {
			return false
		}

		pos = actual.Offset + len(actual.Value)
	}

	return onlyMethodsFollow(source, pos)
}

// onlyMethodsFollow reports whether the source's text from pos to its end is
// method declarations the parse found and nothing else but the component's
// closing brace: each a function declared as a statement, with a body
// wheelsReadMethod reads, and only comments and space between them.
func onlyMethodsFollow(source *sourceInfo, pos int) bool {
	content := source.content
	spans := source.functionSpans()

	trivia := func(text string) bool {
		return parser.NewScriptScanner(text).NextSkipComments().Kind == parser.TokEOF
	}

	for i := range spans {
		s := &spans[i]
		if s.Start < pos {
			continue // one before pos, or nested in the method before
		}

		if s.Kind != parser.FunctionDeclared || s.Body < 0 || !trivia(content[pos:s.Start]) {
			return false
		}

		if _, ok := wheelsReadMethod(content, s); !ok {
			return false
		}

		pos = s.End
	}

	rest := parser.NewScriptScanner(content[pos:])
	if rest.NextSkipComments().Kind != parser.TokRBrace {
		return false
	}

	return rest.NextSkipComments().Kind == parser.TokEOF
}

const wheelsTestInlineBinder = `
 if (structKeyExists(application, "wo")) {
 local.metaIndex = {};
 for (local.fn in getMetaData(application.wo).functions) {local.metaIndex[local.fn.name] = local.fn.access;}
 for (local.key in application.wo) {
  if (!isCustomFunction(application.wo[local.key])) {continue;}
  if (structKeyExists(local.metaIndex, local.key) && local.metaIndex[local.key] neq "public") {continue;}
  if (structKeyExists(variables, local.key) || structKeyExists(this, local.key)) {continue;}
  variables[local.key] = application.wo[local.key];
  this[local.key] = application.wo[local.key];
 }
 }`
