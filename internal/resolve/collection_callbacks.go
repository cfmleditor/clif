package resolve

import (
	"regexp"
	"strings"

	"github.com/cfmleditor/clif/internal/nativetype"

	"github.com/cfmleditor/clif/internal/parser"
)

// Admit literal entityLoad array forms: name alone, a filter struct, and an
// explicit false unique flag. Dynamic names/options and single-entity results
// do not establish a collection. Require indexed ORM metadata, not a basename.
func (r *Resolver) loadedEntityElement(arguments, baseDir string) string {
	pieces := producerSplit(producerTokens(arguments))
	if len(pieces) < 1 || len(pieces) > 3 || len(pieces[0]) != 1 || pieces[0][0].Kind != parser.TokString || strings.Contains(pieces[0][0].Value, "#") {
		return ""
	}

	if len(pieces) > 1 {
		filter := pieces[1]
		if len(filter) < 2 || filter[0].Kind != parser.TokLBrace || producerGroupEnd(filter, 0, parser.TokLBrace, parser.TokRBrace) != len(filter)-1 {
			return ""
		}
	}

	if len(pieces) == 3 && (len(pieces[2]) != 1 || !strings.EqualFold(pieces[2][0].Value, "false")) {
		return ""
	}

	return r.withSubclasses(r.nearestEntity(strings.Trim(pieces[0][0].Value, "\"'"), baseDir))
}

// hqlEntityRe is a literal HQL query that returns the entities it selects from:
// `FROM Booking WHERE …`, or `SELECT b FROM Booking b …`. A projection
// (`SELECT b.name`), a join, which returns tuples, and a second entity in the
// FROM list are not matched.
var hqlEntityRe = regexp.MustCompile(`(?is)^\s*(?:select\s+(\w+)\s+)?from\s+([A-Za-z_][\w.]*)(?:\s+(?:as\s+)?(\w+))?(\s|$)`)

// queriedEntityElement is the entity a literal ormExecuteQuery() returns an array
// of. Only the one-argument form and the one with a struct or array of
// parameters are admitted: a boolean second or third argument asks for a single
// entity (`unique`), and a computed query names nothing. Indexed ORM metadata is
// required, as for entityLoad.
func (r *Resolver) queriedEntityElement(arguments, baseDir string) string {
	pieces := producerSplit(producerTokens(arguments))
	if len(pieces) < 1 || len(pieces) > 2 || len(pieces[0]) != 1 || pieces[0][0].Kind != parser.TokString || strings.Contains(pieces[0][0].Value, "#") {
		return ""
	}

	if len(pieces) == 2 {
		params := pieces[1]
		if len(params) < 2 || params[0].Kind != parser.TokLBrace && params[0].Kind != parser.TokLBracket {
			return ""
		}
	}

	match := hqlEntityRe.FindStringSubmatch(strings.Trim(pieces[0][0].Value, "\"'"))
	if match == nil {
		return ""
	}

	// `select x from Entity y` returns the entity only when x is its alias.
	if match[1] != "" && !strings.EqualFold(match[1], match[3]) {
		return ""
	}

	if strings.Contains(strings.ToLower(pieces[0][0].Value), " join ") || strings.Contains(strings.Split(strings.ToLower(pieces[0][0].Value), " where ")[0], ",") {
		return ""
	}

	return r.withSubclasses(r.nearestEntity(match[2], baseDir))
}

// Collection callbacks have different invocation layouts: array item/index/array,
// struct key/value/struct, and query row/index/query. Never infer a value type
// from a callback method name alone. Erased array items and struct values remain
// unknown; ORM element metadata can establish an array item independently.
func (r *Resolver) collectionCallbackElement(provider *parser.CallSite, closure *closureSpan, parameter int, pr *parser.ParseResult, baseDir string, ctx lookupCtx) string {
	if provider.Variable == "" && len(provider.Chain) == 0 {
		return r.globalCollectionCallbackElement(provider, closure, parameter, pr, baseDir)
	}

	switch strings.ToLower(provider.FuncName) {
	case "each", "filter", "map", "some", "every":
	default:
		return ""
	}

	tokens := allTokens(pr.Content[provider.StartByte:provider.EndByte])
	if len(tokens) < 3 || tokens[1].Kind != parser.TokLParen {
		return ""
	}

	pieces := producerSplit(tokens[2 : len(tokens)-1])
	if len(pieces) == 0 || len(pieces[0]) == 0 || int(provider.StartByte)+pieces[0][0].Offset != closure.functionFrom {
		return ""
	}

	if parameter == 0 || parameter == 1 {
		start := 0
		if scope, ok := enclosingScope(pr, provider.Line); ok {
			start = scope.Start
		}

		expression := callbackCollectionExpression(provider)
		if parameter == 0 {
			if element := r.elementOf(expression, int(provider.Line), start, provider.Caller, pr, baseDir, 0, ""); element != "" {
				return element
			}
		} else if value := r.structCollectionValue(expression, int(provider.Line), start, provider.Caller, pr, baseDir, 0, ""); value != "" {
			return value
		}
	}

	// A guessed collection family cannot establish callback ownership: a guessed
	// array's index could actually be a struct's CFC value. Preserve that unknown
	// instead of promoting a receiver-family assumption to an invocation contract.
	key := SpeculativeReceiver{File: pr.URI.Path(), Caller: provider.Caller, Variable: strings.ToLower(provider.Variable)}
	if r.SpeculativeReceivers[key] != "" {
		return ""
	}

	component, _ := r.receiverComponentD(provider.Variable, provider.Line, provider.Caller, provider.FuncName, pr, baseDir, nil, lookupCtx{depth: ctx.depth + 1, call: provider, callers: ctx.callers})
	if component == "" {
		return ""
	}

	component, _, _, done := r.walkHops(component, "", provider, pr, baseDir, nil)
	if done {
		return ""
	}

	return nativeCallbackParameter(component, parameter)
}

func nativeCallbackParameter(component string, parameter int) string {
	switch component {
	case nativetype.Type("array"):
		switch parameter {
		case 1:
			return nativetype.Type("numeric")
		case 2:
			return component
		}
	case nativetype.Type("struct"):
		switch parameter {
		case 0:
			return nativetype.Type("string")
		case 2:
			return component
		}
	case nativetype.Type("query"), "$builtin.querynew", "$builtin.queryexecute":
		switch parameter {
		case 0:
			return nativetype.Type("struct")
		case 1:
			return nativetype.Type("numeric")
		case 2:
			return component
		}
	}

	return ""
}

// Only an unshadowed global builtin establishes this invocation layout. The
// array's item still needs independent element evidence; API-required key, row,
// index and collection arguments do not need a guess about the input's contents.
func (r *Resolver) globalCollectionCallbackElement(provider *parser.CallSite, closure *closureSpan, parameter int, pr *parser.ParseResult, baseDir string) string {
	if len(provider.Chain) > 0 || pr.Declares(provider.FuncName) || r.bareFunc(provider.FuncName, pr, baseDir, provider.Caller) != nil {
		return ""
	}

	family := collectionBuiltin(provider.FuncName)
	if family == "" {
		return ""
	}

	tokens := allTokens(pr.Content[provider.StartByte:provider.EndByte])
	if len(tokens) < 3 || tokens[1].Kind != parser.TokLParen {
		return ""
	}

	pieces := producerSplit(tokens[2 : len(tokens)-1])
	if len(pieces) < 2 || len(pieces[1]) == 0 || int(provider.StartByte)+pieces[1][0].Offset != closure.functionFrom {
		return ""
	}

	if family == "array" && parameter == 0 {
		start := 0
		if scope, ok := enclosingScope(pr, provider.Line); ok {
			start = scope.Start
		}

		return r.elementOf(producerText(pieces[0]), int(provider.Line), start, provider.Caller, pr, baseDir, 0, "")
	}

	if family == "struct" && parameter == 1 {
		start := 0
		if scope, ok := enclosingScope(pr, provider.Line); ok {
			start = scope.Start
		}

		return r.structCollectionValue(producerText(pieces[0]), int(provider.Line), start, provider.Caller, pr, baseDir, 0, "")
	}

	return nativeCallbackParameter(nativetype.Type(family), parameter)
}

func collectionBuiltin(name string) string {
	name = strings.ToLower(name)
	for _, family := range []string{"array", "struct", "query"} {
		if method, ok := strings.CutPrefix(name, family); ok {
			switch method {
			case "each", "filter", "map", "some", "every":
				return family
			}
		}
	}

	return ""
}
