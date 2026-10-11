package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
)

// Explicit function declarations establish collection interfaces even when the
// body has an unknown producer. Property types do not enforce generated getter
// results; struct declarations also admit CFCs. Neither supplies this contract.
func (r *Resolver) declaredCollectionReturn(fd *parser.FunctionDef) string {
	family := strings.ToLower(fd.ReturnType)
	if family != "array" && family != "query" {
		return ""
	}

	if fd.Generated {
		return ""
	}

	return nativetype.Type(family)
}

// BuiltinReturnLookup keeps engine-specific producers aligned between parse and
// resolution. Configured component resolvers and declared methods retain priority.
func (r *Resolver) BuiltinReturnLookup(name string) string {
	if native := nativetype.EngineProducer(name, r.NativeEngine); native != "" {
		return native
	}

	return docs.LookupBuiltinReturnComponent(name)
}

// BuiltinReturnLookupFor checks source ownership before supplying a native
// return type. An inherited method with an unknown result still shadows the
// builtin. Use this callback when parsing a particular file.
func (r *Resolver) BuiltinReturnLookupFor(file string) func(string) string {
	return func(name string) string {
		native := r.BuiltinReturnLookup(name)
		if native == "" || r.ResolveFunc(file, name, filepath.Dir(file)) != nil {
			return ""
		}

		return native
	}
}

// nativeLiteral describes a whole literal expression. Expression suffixes are
// rejected so a literal followed by a call is not mistaken for that call's result.
func nativeLiteral(expression string) string {
	tokens := producerUnwrap(producerTokens(expression))
	if len(tokens) == 0 {
		return ""
	}

	if len(tokens) == 1 {
		switch tokens[0].Kind {
		case parser.TokString:
			return nativetype.Type("string")
		case parser.TokNumber:
			return nativetype.Type("numeric")
		default:
			return ""
		}
	}

	if tokens[0].Kind == parser.TokLBracket && producerGroupEnd(tokens, 0, parser.TokLBracket, parser.TokRBracket) == len(tokens)-1 {
		// Square brackets also delimit ordered structs. Inspect the first
		// entry's separator, not colons/equals inside nested array values or
		// ternaries. [:] and [=] are the empty ordered-struct forms.
		body := tokens[1 : len(tokens)-1]
		if len(body) == 1 && (body[0].Kind == parser.TokColon || body[0].Kind == parser.TokEquals) {
			return nativetype.Type("struct")
		}

		if len(body) >= 3 && (body[1].Kind == parser.TokColon || body[1].Kind == parser.TokEquals) {
			return nativetype.Type("struct")
		}

		return nativetype.Type("array")
	}

	return ""
}
