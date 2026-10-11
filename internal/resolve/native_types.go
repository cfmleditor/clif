package resolve

import (
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
		return nativetype.Type("array")
	}

	return ""
}
