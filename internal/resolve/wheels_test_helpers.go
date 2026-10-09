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
			if _, overridden := r.wheelsSource(child).methods["$bindapplicationhelpers"]; overridden {
				return nil
			}
		}

		source := r.wheelsSource(path)
		if !wheelsTestBinds(source) {
			continue
		}

		bindings := r.ComponentPath("wheels.Bindings", filepath.Dir(path))

		data, err := r.fs().ReadFile(bindings)
		if err != nil {
			return nil
		}

		tokens := wheelsTokens(string(data))
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
func wheelsTestBinds(source wheelsSource) bool {
	if source.methods["init"].body == wheelsTokens(`$bindApplicationHelpers();return this;`) && source.methods["$bindapplicationhelpers"].body == wheelsTokens(wheelsTestBinder) {
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

	for _, expected := range significantTokens(wheelsTestInlineBinder) {
		actual := scanner.NextSkipComments()
		if wheelsTokens(actual.Value) != wheelsTokens(expected.Value) {
			return false
		}
	}

	var header []parser.Token

	for {
		t := scanner.NextSkipComments()
		if t.Kind == parser.TokRBrace {
			return len(header) == 0 && scanner.NextSkipComments().Kind == parser.TokEOF
		}

		if t.Kind != parser.TokIdent {
			return false
		}

		if !strings.EqualFold(t.Value, "function") {
			if len(header) >= 4 {
				return false
			}

			header = append(header, t)

			continue
		}

		name, _ := wheelsReadMethod(scanner, header)
		if name == "" {
			return false
		}

		header = nil
	}
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
