package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// A builtin spelling cannot establish a return interface when a callable
// argument or earlier local binding owns that name. Line-based source recovery
// conservatively withholds callbacks sharing a line with a shadowing parameter.
func (r *Resolver) builtinCallShadowed(name string, line uint32, caller string, pr *parser.ParseResult) bool {
	if argumentOf(pr, caller, name) != nil {
		return true
	}

	if scope, ok := enclosingScope(pr, line); ok {
		if _, _, assigned := localAssignmentAt(pr.Content, name, scope.Start, int(line)); assigned {
			return true
		}
	}

	for _, span := range r.closuresOf(pr) {
		if int(line) < span.start || int(line) > span.end {
			continue
		}

		for _, parameter := range span.params {
			if strings.EqualFold(parameter, name) {
				return true
			}
		}
	}

	return false
}
