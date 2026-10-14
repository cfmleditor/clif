package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// TestBox's xUnit entry contract runs the selected beforeTests before test*
// methods. Retain only literal closure registrations through the source-backed
// BaseSpec API. This contract does not apply to arbitrary direct invocations,
// callback-based registration, BDD hooks or calls through aliases.
func (r *Resolver) registeredAssertion(comp string, call *parser.CallSite, pr *parser.ParseResult, dir string) *parser.FunctionDef {
	if pr == nil || !strings.HasPrefix(strings.ToLower(call.Caller), "test") || len(call.Chain) != 0 {
		return nil
	}

	receiver := strings.ToLower(call.Variable)
	if receiver != "$assert" && receiver != "this.$assert" && receiver != "variables.$assert" {
		return nil
	}

	self := pr.URI.Path()

	registrar := r.ResolveFunc(self, "addAssertions", dir)
	if registrar == nil || len(registrar.Arguments) != 1 || !strings.EqualFold(registrar.Arguments[0].Name, "assertions") || !strings.EqualFold(filepath.Base(registrar.URI.Path()), "BaseSpec.cfc") {
		return nil
	}

	source := r.sourceInfo(registrar.URI.Path())
	api := source.methods["addassertions"]

	contract := wheelsTokens(`if(isStruct(arguments.assertions)){structAppend(this.$assert,arguments.assertions,true);return this;}`)
	if !api.public || !api.literalDefaults || !strings.HasPrefix(api.body, contract) {
		return nil
	}

	if r.ResolveFunc(self, "structAppend", dir) != nil || r.ResolveFunc(self, "isStruct", dir) != nil {
		return nil
	}

	assertion := r.ComponentPath("testbox.system.Assertion", filepath.Dir(registrar.URI.Path()))
	if assertion == "" || !cfpath.SamePath(assertion, r.ComponentPath(comp, dir)) {
		return nil
	}
	// The receiver is the BaseSpec-created assertion object, not a coincidentally
	// named field. Child writes and escapes are checked separately below.
	constructor := parser.ApplicationConstructor(source.content)
	if !strings.Contains(wheelsTokens(constructor), wheelsTokens(`variables.$assert=this.$assert=new testbox.system.Assertion();`)) || !assertionProviderStable(&source) {
		return nil
	}

	hook := r.ResolveFunc(self, "beforeTests", dir)
	if hook == nil || len(hook.Arguments) != 0 || hook.URI == registrar.URI {
		return nil
	}

	method := r.sourceInfo(hook.URI.Path()).methods["beforetests"]
	if !method.public || !method.literalDefaults {
		return nil
	}

	tokens := producerTokens(method.body)
	if len(tokens) == 0 {
		return nil
	}

	if len(tokens) >= 6 && wheelsTokens(producerText(tokens[:6])) == wheelsTokens(`super.beforeTests();`) {
		parent, _ := r.extendsOf(hook.URI.Path(), hook.URI)

		parentFile := r.ComponentPath(parent, filepath.Dir(hook.URI.Path()))
		if !r.harmlessAssertionHook(r.ResolveFunc(parentFile, "beforeTests", filepath.Dir(hook.URI.Path()))) {
			return nil
		}

		tokens = tokens[6:]
	}

	members := literalAssertionRegistrations(tokens)
	if !members[strings.ToLower(call.FuncName)] {
		return nil
	}

	seen := map[string]bool{}
	for file := self; file != ""; {
		if seen[pathKey(file)] || len(seen) >= 12 {
			return nil
		}

		seen[pathKey(file)] = true

		current := r.handlerParse(file)
		if current == nil {
			return nil
		}
		// The known registrar implementation handles both literal and runtime
		// records. It is not an arbitrary child writer.
		expectedRegistrations := 0
		if cfpath.SamePath(file, hook.URI.Path()) {
			expectedRegistrations = assertionRegistrationCount(method.body)
		}

		if !cfpath.SamePath(file, registrar.URI.Path()) && (!assertionStorageStable(current.Content) || assertionRegistrationCount(current.Content) != expectedRegistrations) {
			return nil
		}

		if cfpath.SamePath(file, registrar.URI.Path()) {
			break
		}

		parent, ok := r.extendsOf(file, current.URI)
		if !ok {
			return nil
		}

		file = r.ComponentPath(parent, filepath.Dir(file))
		if file == "" {
			return nil
		}
	}

	if !seen[pathKey(registrar.URI.Path())] {
		return nil
	}
	// Unknown code receiving the instance may replace its assertion field before
	// this call, even when it never spells $assert itself.
	scope := pr.FuncScopeAt(int(call.Line))
	if !call.HasSpan || scope.Start < 0 {
		return nil
	}

	from := 0
	for range scope.Start {
		next := strings.IndexByte(pr.Content[from:], '\n')
		if next < 0 {
			return nil
		}

		from += next + 1
	}

	bodyFrom := assertionCallBodyStart(pr.Content, from, int(call.StartByte), call.Caller)
	if bodyFrom < 0 {
		return nil
	}

	if !guardValueStable(pr.Content, bodyFrom, int(call.StartByte), "$assert") {
		return nil
	}

	return &parser.FunctionDef{Name: call.FuncName, URI: hook.URI, Line: hook.Line}
}

func (r *Resolver) harmlessAssertionHook(fd *parser.FunctionDef) bool {
	if fd == nil || len(fd.Arguments) != 0 {
		return false
	}

	plan := r.producerFor(fd)
	if plan == nil || !instanceLiteralDefaults(plan) {
		return false
	}

	for i := range plan.body {
		node := &plan.body[i]
		if node.kind != "set" || strings.Contains(strings.ToLower(node.target), "$assert") {
			return false
		}

		literal := producerTokens(node.expression)
		if len(literal) != 1 || literal[0].Kind != parser.TokNumber && literal[0].Kind != parser.TokString {
			return false
		}

		if literal[0].Kind == parser.TokString && strings.Contains(literal[0].Value, "#") {
			return false
		}
	}

	return true
}

func literalAssertionRegistrations(tokens []parser.Token) map[string]bool {
	members := map[string]bool{}

	for len(tokens) > 0 {
		if len(tokens) < 6 || !strings.EqualFold(tokens[0].Value, "addAssertions") || tokens[1].Kind != parser.TokLParen || tokens[2].Kind != parser.TokLBrace {
			return nil
		}

		end := producerGroupEnd(tokens, 2, parser.TokLBrace, parser.TokRBrace)
		if end < 0 || end+2 >= len(tokens) || tokens[end+1].Kind != parser.TokRParen || tokens[end+2].Kind != parser.TokSemicolon {
			return nil
		}

		for _, piece := range producerSplit(tokens[3:end]) {
			if len(piece) < 6 || piece[1].Kind != parser.TokColon && piece[1].Kind != parser.TokEquals {
				return nil
			}

			name := piece[0].Value
			if piece[0].Kind == parser.TokString {
				name = diString(piece[:1])
			} else if piece[0].Kind != parser.TokIdent {
				return nil
			}

			if name == "" || strings.ContainsAny(name, "#.") || !strings.EqualFold(piece[2].Value, "function") || piece[3].Kind != parser.TokLParen {
				return nil
			}

			argsEnd := producerGroupEnd(piece, 3, parser.TokLParen, parser.TokRParen)
			if argsEnd < 0 || argsEnd+1 >= len(piece) || piece[argsEnd+1].Kind != parser.TokLBrace || producerGroupEnd(piece, argsEnd+1, parser.TokLBrace, parser.TokRBrace) != len(piece)-1 {
				return nil
			}

			for _, token := range piece[argsEnd+2 : len(piece)-1] {
				if strings.EqualFold(token.Value, "addAssertions") {
					return nil
				}
			}

			members[strings.ToLower(name)] = true
		}

		tokens = tokens[end+3:]
	}

	return members
}

// Outside the registrar, $assert may only be read as a method receiver. Whole
// field publication and extra registration calls require broader effect plans.
func assertionStorageStable(content string) bool { return assertionStorageStableDepth(content, 0) }

func assertionStorageStableDepth(content string, depth int) bool {
	if depth > 8 {
		return false
	}

	tokens := allTokens(content)
	if len(tokens) == 0 {
		return false
	}

	for i, t := range tokens {
		if t.Kind == parser.TokString {
			if strings.EqualFold(diString(tokens[i:i+1]), "$assert") && i+1 < len(tokens) && tokens[i+1].Kind == parser.TokRBracket {
				return false
			}

			for _, expression := range parser.InterpolatedExpressions(t.Value) {
				if !assertionStorageStableDepth(expression, depth+1) {
					return false
				}
			}

			continue
		}

		if t.Kind != parser.TokIdent {
			continue
		}

		if (strings.EqualFold(t.Value, "this") || strings.EqualFold(t.Value, "variables") || strings.EqualFold(t.Value, "isStruct") || strings.EqualFold(t.Value, "structAppend")) && i+1 < len(tokens) && (tokens[i+1].Kind == parser.TokEquals || tokens[i+1].Kind == parser.TokLBracket) {
			return false
		}

		if strings.EqualFold(t.Value, "include") || strings.EqualFold(t.Value, "cfinclude") {
			return false
		}

		if strings.EqualFold(t.Value, "$assert") && (i+3 >= len(tokens) || tokens[i+1].Kind != parser.TokDot || tokens[i+2].Kind != parser.TokIdent || tokens[i+3].Kind != parser.TokLParen) {
			return false
		}
	}

	return true
}

func assertionRegistrationCount(content string) int {
	tokens := allTokens(content)
	count := 0

	for i, t := range tokens {
		if strings.EqualFold(t.Value, "addAssertions") && i+1 < len(tokens) && tokens[i+1].Kind == parser.TokLParen {
			count++
		}
	}

	return count
}

// Source changes introducing another direct assertion writer invalidate the
// framework registration contract. The registrar is the one supported writer.
func assertionProviderStable(source *sourceInfo) bool {
	constructor := parser.ApplicationConstructor(source.content)

	remaining := strings.Replace(wheelsTokens(constructor), wheelsTokens(`variables.$assert=this.$assert=new testbox.system.Assertion();`), "", 1)
	if strings.Contains(remaining, "$assert") {
		return false
	}

	for name := range source.methods {
		method := source.methods[name]
		if name == "addassertions" {
			continue
		}

		tokens := allTokens(method.body)
		for i, t := range tokens {
			if strings.EqualFold(t.Value, "$assert") && i+1 < len(tokens) && (tokens[i+1].Kind == parser.TokEquals || tokens[i+1].Kind == parser.TokLBracket) {
				return false
			}
		}
	}

	return true
}

func assertionCallBodyStart(content string, from, to int, caller string) int {
	if from < 0 || to < from || to > len(content) {
		return -1
	}

	tokens := allTokens(content[from:to])
	for i, t := range tokens {
		if !strings.EqualFold(t.Value, "function") || i+2 >= len(tokens) || !strings.EqualFold(tokens[i+1].Value, caller) || tokens[i+2].Kind != parser.TokLParen {
			continue
		}

		end := producerGroupEnd(tokens, i+2, parser.TokLParen, parser.TokRParen)
		if end < 0 {
			return -1
		}

		for j := end + 1; j < len(tokens); j++ {
			if tokens[j].Kind == parser.TokLBrace {
				return from + tokens[j].Offset + 1
			}
		}
	}

	return -1
}
