package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
)

// memberReceiver is the component a member receiver (`a.b`, `prc.x`,
// `args.x`) holds: what the file records for it, then each of the lookups that
// read it from somewhere else, the first to answer winning.
func (r *Resolver) memberReceiver(variable, name string, scope parser.RefScope, line uint32, caller, funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace, ctx lookupCtx) string {
	comp := r.recordReceiver(variable, name, scope, line, caller, pr, ctx.call)
	if comp == "" {
		comp = r.builtinResultField(variable, line, caller, pr, baseDir, ctx.call)
	}

	if comp == "" {
		comp = r.builderMember(variable, line, caller, funcName, pr, baseDir, tr, ctx)
	}

	if comp == "" {
		comp = r.assignedFromCall(variable, line, caller, pr, baseDir, tr, ctx)
	}

	// A view's rc member is what its controller left there, and a
	// template's what the file including it holds.
	if comp == "" {
		comp = r.fw1ViewRc(variable, "", pr, tr)
	}

	if comp == "" {
		comp = r.coldboxPrcResponse(variable, pr)
	}

	if comp == "" {
		comp = r.viewArgs(variable, pr, tr)
	}

	// An interceptor's data is what the calls announcing it pass.
	if comp == "" {
		comp = r.interceptData(variable, caller, pr, tr, ctx)
	}

	// A custom tag's attributes are what the tags calling it pass.
	if comp == "" {
		comp = r.customTagAttribute(variable, pr, tr, ctx)
	}

	if comp == "" {
		comp = r.includerHeld(variable, pr, tr, ctx)
	}

	if comp == "" {
		comp = r.nativeMember(variable, line, caller, pr, baseDir, tr, ctx)
	}

	return comp
}

// nativeMember is what `holder.key` holds when holder is a struct or a query:
// a struct's value, and a query's column, are of no shape the source gives, so
// dynamic; a query's own properties are what they are.
func (r *Resolver) nativeMember(variable string, line uint32, caller string, pr *parser.ParseResult, baseDir string, tr *callTrace, ctx lookupCtx) string {
	holder, key, ok := strings.CutLast(strings.TrimSuffix(variable, "[]"), ".")
	if !ok || holder == "" || key == "" || ctx.depth >= maxAssignedDepth {
		return ""
	}

	comp, _ := r.receiverComponentD(holder, line, caller, "", pr, baseDir, nil, lookupCtx{depth: ctx.depth + 1, leaf: ctx.leaf, callers: ctx.callers})

	var answer string

	switch comp {
	case nativetype.Type("struct"):
		answer = "$any"
	case nativetype.Type("query"), "$builtin.querynew", "$builtin.queryexecute":
		switch strings.ToLower(key) {
		case "columnlist":
			answer = nativetype.Type("string")
		case "recordcount", "currentrow":
			answer = nativetype.Type("numeric")
		default:
			answer = "$any"
		}
	default:
		return ""
	}

	if strings.HasSuffix(variable, "[]") && answer != "$any" {
		return ""
	}

	tr.addf("resolved %q to %q: it is a member of %q", variable, answer, holder)

	return answer
}
