// The component a call's receiver holds: scoped and file-level refs, arguments, Application.cfc and the extends chain.

package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// ComponentOf reports the component variable holds at line, or "" when that
// is not known or is not a component. It is the answer go-to-type-definition
// wants, and it runs the receiver lookup CanResolveCall runs, so the two cannot
// disagree about what a variable holds.
//
// Two of CanResolveCall's fallbacks are left out on purpose. A
// componentResolver is tried against the variable name, but not against the
// whole line, which exists for chained calls and would answer for whatever
// else the line holds. And "$any", "$builtin." and a registry family (any one
// of its members, whose marker would split on its "|" into bare member paths)
// are not one component, so they are reported as unknown.
func (r *Resolver) ComponentOf(variable string, line uint32, pr *parser.ParseResult, baseDir string) string {
	caller := pr.FuncScopeAt(int(line)).Name

	comp, _ := r.receiverComponent(variable, line, caller, "", pr, baseDir, nil)

	if comp == "" {
		comp, _, _ = parser.ResolveFromCallMatch(variable, r.Resolvers)
	}

	if comp == "$any" || strings.HasPrefix(comp, "$builtin.") || strings.HasPrefix(comp, familyPrefix) {
		return ""
	}

	return comp
}

func (r *Resolver) recordReceiver(variable, name string, scope parser.RefScope, line uint32, caller string, pr *parser.ParseResult, calls ...*parser.CallSite) string {
	if ref := funcScopedRef(pr, line, name, scope, calls...); ref != nil {
		return ref.Component
	}

	root, _, _ := strings.Cut(name, ".")

	explicit := strings.HasPrefix(strings.ToLower(variable), "variables.") || strings.HasPrefix(strings.ToLower(variable), "this.")
	if !explicit && argumentOf(pr, caller, root) != nil {
		return ""
	}

	if !explicit {
		if strings.HasPrefix(strings.ToLower(variable), "local.") || strings.HasPrefix(strings.ToLower(variable), "arguments.") {
			return ""
		}

		if fs := pr.FuncScopeAt(int(line)); fs.Start != -1 {
			if pr.FuncDeclares(fs.Start, fs.End, root) {
				return ""
			}
		}
	}

	if ref := fileLevelRef(pr, line, name, scope); ref != nil {
		return ref.Component
	}

	return ""
}

// receiverComponent finds the component a qualified call's receiver holds at
// line: a ref scoped to the enclosing function, the nearest preceding ref in the
// file, an Application.cfc ref, an ARGUMENTS.x type, and refs up the extends
// chain, in that order. It is canResolveCall's lookup, shared with
// ComponentOf so that go-to-type-definition answers exactly as call
// resolution does; the componentResolver fallbacks stay with each caller.
//
// member reports the one case that is not a component at all: an ARGUMENTS.x
// of primitive type calling a known member method, which canResolveCall
// accepts outright. It needs funcName; ComponentOf passes none.
func (r *Resolver) receiverComponent(variable string, line uint32, caller, funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace) (comp string, member bool) {
	return r.receiverComponentD(variable, line, caller, funcName, pr, baseDir, tr, lookupCtx{})
}

// inferredReceiver is the receiver steps that read something other than a ref
// the file holds: a closure's builder parameter and a builder's restrictions, what
// every subclass of an abstract file holds, and the call a variable was last
// assigned from.
func (r *Resolver) inferredReceiver(variable string, line uint32, caller, funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace, ctx lookupCtx) string {
	if comp := r.builderMember(variable, line, caller, funcName, pr, baseDir, tr, ctx); comp != "" {
		return comp
	}

	if comp := r.emptyCollectionField(variable, line, caller, funcName, pr, tr); comp != "" {
		return comp
	}

	if comp := r.subclassComponent(variable, line, pr, tr, ctx.leaf); comp != "" {
		return comp
	}

	if comp := r.assignedFromCall(variable, line, caller, pr, baseDir, tr, ctx); comp != "" {
		return comp
	}

	if comp := r.muraObjectArg(variable, pr, tr, ctx); comp != "" {
		return comp
	}

	return r.includerHeld(variable, pr, tr, ctx)
}

// familyOverGuess is the family an assignment of the receiver proves over a
// parse-time dynamic guess, or the guess when no family does: a construction
// a registry recipe's own listing serves is a member of the family the
// recipes' guards keep, and the guess the parse could not walk stays its
// guess against anything else.
func (r *Resolver) familyOverGuess(comp, variable string, line uint32, caller string, pr *parser.ParseResult, baseDir string, tr *callTrace, ctx lookupCtx) string {
	if comp != "$any" || !r.registriesReach(pr) {
		return comp
	}

	family := r.assignedFamily(variable, line, caller, pr, baseDir, ctx)
	if family == "" {
		return comp
	}

	tr.addf("%q's dynamic guess is overwritten by the family the construction's listing serves: %q", variable, family)

	return family
}

// registriesReach reports whether a registry recipe can reach pr: its own
// text holds one, or an includer's does. Only then can an assignment prove a
// family, and the walk that looks for one is skipped everywhere else.
func (r *Resolver) registriesReach(pr *parser.ParseResult) bool {
	if pr == nil {
		return false
	}

	// Asked for every call on a dynamic receiver: the recipe's word is
	// looked for once per parse.
	own, _ := pr.Memo("holdsRegistry", func() any {
		return mentions(pr.Content, "ComponentListPackageAsStruct")
	}).(bool)

	if own && len(r.registriesOf(pr)) > 0 {
		return true
	}

	return pr.URI != "" && len(r.includeRegistries(cfpath.FromURI(string(pr.URI)))) > 0
}

// keyedReceiverStem is the collection a keyed element read the parser filed
// as a receiver names — the parser spells the read as the receiver with its
// key off (`db[]`) — or "" when the variable is not one.
func keyedReceiverStem(variable string) (stem string) {
	m := keyedReceiver(strings.TrimSpace(variable))
	if m == nil {
		return ""
	}

	return m[1]
}

// speculativeReceiverOver is the guess a batch scan recorded for the
// receiver, when the origin the lookup ran at proves nothing of its own —
// ok reports there is one.
func (r *Resolver) speculativeReceiverOver(variable, comp string, member bool, caller string, pr *parser.ParseResult) (string, bool) {
	if comp != "" || member || pr == nil {
		return "", false
	}

	key := SpeculativeReceiver{File: pr.URI.Path(), Caller: caller, Variable: strings.ToLower(variable)}

	guess := r.SpeculativeReceivers[key]

	return guess, guess != ""
}

// keyedReceiverElement is the element a keyed element read the parser filed
// as a receiver holds: db['test'].findOne(…)'s receiver is the read, and the
// element is the collection the read indexes' stated one. Only what the
// collection's interface proves answers; a dynamic guess is no answer.
func (r *Resolver) keyedReceiverElement(variable string, line uint32, caller string, pr *parser.ParseResult, baseDir string, tr *callTrace, ctx lookupCtx) (elem string) {
	stem := keyedReceiverStem(variable)

	if stem != "" {
		elem = r.elementOf(stem, int(line), 0, caller, pr, baseDir, ctx.depth+1, ctx.leaf)

		if elem != "" && strings.HasPrefix(elem, "$") && !strings.HasPrefix(elem, familyPrefix) {
			tr.addf("%q's keyed element read %q names reads against a dynamic element — no answer of its own", variable, stem)

			elem = ""
		}

		if elem == "" {
			elem = r.nativeElementOf(stem, line, caller, pr, baseDir, ctx)
		}

		if elem != "" {
			tr.addf("%q is the keyed element read %q names, whose collection's element is %q", variable, stem, elem)
		}
	}

	return elem
}

// nativeElementOf is the interface each struct-notation element of the
// native handle a variable holds is, as the handle's contract states it: a
// MongoDB database handle's db["coll"] is a collection. A variable holding
// no native handle, or one whose contract states no element, answers "".
func (r *Resolver) nativeElementOf(variable string, line uint32, caller string, pr *parser.ParseResult, baseDir string, ctx lookupCtx) string {
	if ctx.depth >= maxAssignedDepth {
		return ""
	}

	comp, _ := r.receiverComponentD(variable, line, caller, "", pr, baseDir, nil, lookupCtx{depth: ctx.depth + 1, leaf: ctx.leaf, callers: ctx.callers})

	elem, _ := nativetype.Element(comp)

	return elem
}

// argumentHolderType is the component an argument's own holder carries, as
// its declared state says: the component the argument was typed to, the
// type it declares, or the bean the body constructs when the caller passes
// none, or with ctx.callers what every caller passes it. member marks an
// untyped argument whose method is a member method (dynamic). The
// argument's presence stops the lookup either way: an argument shadows what
// the outer scopes hold.
func (r *Resolver) argumentHolderType(variable, name string, arg *parser.Argument, funcName string, pr *parser.ParseResult, caller string, tr *callTrace, ctx lookupCtx) (string, bool) {
	if arg.Component != "" {
		return arg.Component, false
	}

	if strings.Contains(arg.Type, ".") {
		tr.addf("resolved %q via enclosing function argument", variable)

		return arg.Type, false
	}

	// An argument declared a native type (query, struct, array, string…) holds
	// one, which the engine enforces: a method that type's contract has is
	// accepted, as it is on a local of the type.
	if native := nativeArgumentFamily(arg.Type); native != "" {
		if _, ok := nativetype.Method(nativetype.Type(native), funcName); ok {
			tr.addf("resolved %q to %q via its declared type %q", variable, nativetype.Type(native), arg.Type)

			return nativetype.Type(native), false
		}
	}

	// A bare type is read as a bare return type is (namedType).
	// `required Duration otherDuration` was left untyped, so every call on
	// the argument was unresolved.
	if path := r.bareArgumentComponent(arg.Type, pr); path != "" {
		tr.addf("resolved %q to %q via its declared type %q", variable, path, arg.Type)

		return path, false
	}

	if parser.IsMemberMethod(funcName) {
		return "$any", true
	}

	// A defaulted argument: the body guards its absence and constructs what
	// it holds then — the construction is the argument's component.
	if comp := r.defaultedArgumentComponent(name, pr, caller); comp != "" {
		tr.addf("resolved %q to %q: the body constructs it when the caller passes none", variable, comp)

		return comp, false
	}

	// What its callers pass, only when canResolveCall is asking again for
	// a call nothing else answered (ctx.callers).
	if ctx.callers {
		if comp := r.argumentFromCallers("arguments."+name, caller, pr, ctx); comp != "" {
			tr.addf("resolved %q to %q: what every caller of %s passes it", variable, comp, caller)

			return comp, false
		}
	}

	tr.addf("argument %q shadows component fields", name)

	return "", false
}

// applicationRefComponent is the component an Application.cfc ref gives a
// receiver nothing nearer typed. proven says a verified factory result stood in
// for the parser's guess and is the whole answer.
func (r *Resolver) applicationRefComponent(variable, lookupVar, baseDir string, tr *callTrace) (comp string, proven bool) {
	// A verified factory result takes precedence over the parser's
	// class-level return guess from Application.cfc. Conflicting startup
	// writes withhold that result rather than restoring the old guess.
	if component, ok := r.wheelsStartupComponent(variable, baseDir, tr); ok {
		return component, true
	}

	for _, hit := range r.applicationRefs(baseDir, lookupVar) {
		comp = hit.ref.Component

		tr.addf("resolved %q to %q via %s ComponentRef", variable, comp, hit.file)

		if comp != "" {
			break
		}
	}

	return comp, false
}

// untypedReceiverComponent is the last of the answers for a receiver every
// ref missed, in this order: what its base classes hold, what its assignments
// and callers give it, and a shared-scope variable a template the
// application's Application.cfc includes sets up (see startup.go).
func (r *Resolver) untypedReceiverComponent(variable, lookupVar string, line uint32, caller, funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace, ctx lookupCtx) string {
	comp := r.inheritedReceiver(variable, lookupVar, pr, baseDir, tr)

	if comp == "" {
		comp = r.inferredReceiver(variable, line, caller, funcName, pr, baseDir, tr, ctx)
	}

	if comp == "" {
		comp = r.startupComponent(variable, baseDir, tr)
	}

	return comp
}

// receiverComponentD is receiverComponent with a lookupCtx: the depth of
// assignments read to type a variable, which bounds `x = y.f()` through
// `y = z.g()`, and the subclass a handoff reads on behalf of.
func (r *Resolver) receiverComponentD(variable string, line uint32, caller, funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace, ctx lookupCtx) (comp string, member bool) {
	defer func() {
		if guess, ok := r.speculativeReceiverOver(variable, comp, member, caller, pr); ok {
			comp = guess
			tr.addf("speculative receiver %q -> %q (origin unproven)", variable, guess)
		}
	}()

	if guarded := r.guardedReceiver(variable, pr, baseDir, ctx); guarded != "" {
		tr.addf("resolved %q to %q through an exact positive isInstanceOf guard", variable, guarded)

		return guarded, false
	}

	if name, scope, record := parser.MemberReceiverName(variable); record {
		// A keyed element read whose key holds a dot — drivers[form.type] —
		// reads as a member to the scope split, and is the read it spells.
		if elem := r.keyedReceiverElement(variable, line, caller, pr, baseDir, tr, ctx); elem != "" {
			return elem, false
		}

		return r.memberReceiver(variable, name, scope, line, caller, funcName, pr, baseDir, tr, ctx), false
	}

	// Strip scope prefix for matching (VARIABLES.x -> x). Bracket-aware: a "."
	// inside a "[...]" subscript (e.g. "linkMap[arguments.startSource]") is not a
	// scope prefix and must not be stripped there.
	lookupVar := parser.StripReceiverScope(variable)
	scope := parser.ReceiverRefScope(variable)

	// Each lookup below is tried while comp is still empty. A matching ref
	// with no component is logged and passed over, as it always was.

	// A callback parameter owns its spelling even when no invocation type is
	// known. It must not borrow a same-named outer local or component field.
	ref := funcScopedRef(pr, line, lookupVar, scope, ctx.call)
	if r.callbackParameterShadows(variable, ref, pr, ctx.call) {
		return r.ownedCallbackComponent(variable, line, caller, funcName, pr, baseDir, tr, ctx), false
	}

	// Try function-scoped refs first
	method := receiverMethod(funcName, ctx.call)
	ref = funcRefAfterDefault(ref, pr, line, variable, lookupVar, scope, method, ctx.call, tr)

	if ref != nil {
		comp = ref.Component

		tr.addf("resolved %q to %q via function-scoped ComponentRef", variable, comp)
	}

	// Mura's event, in the code Mura hands one to.
	if ev := r.muraEvent(variable, comp, line, caller, pr); ev != "" {
		tr.addf("resolved %q to %q: Mura calls this file with its event", variable, ev)

		return ev, false
	}

	// An argument shadows a same-named component field. In particular, an
	// untyped parameter in a sibling method must not borrow an injected field.
	argumentReceiver := !strings.Contains(variable, ".") || strings.HasPrefix(strings.ToLower(variable), "arguments.")

	if comp == "" && argumentReceiver {
		if arg := argumentOf(pr, caller, lookupVar); arg != nil {
			return r.argumentHolderType(variable, lookupVar, arg, funcName, pr, caller, tr, ctx)
		}
	}

	if comp == "" {
		if best := fileRefAfterDefault(pr, line, variable, lookupVar, scope, method, tr); best != nil {
			comp = best.Component

			tr.addf("resolved %q to %q via file-level ComponentRef (nearest preceding assignment at line %d)", variable, comp, best.Line+1)
		}
	}

	// A parse-time dynamic guess is a weak claim; a family an assignment
	// proves is the sharper answer.
	comp = r.familyOverGuess(comp, variable, line, caller, pr, baseDir, tr, ctx)

	// Fall back to Application.cfc component refs
	if comp == "" {
		var proven bool

		if comp, proven = r.applicationRefComponent(variable, lookupVar, baseDir, tr); proven {
			return comp, false
		}
	}

	// A keyed element read as a receiver — db['test'].findOne(…) — is the
	// element the collection's interface states: the parser files the read
	// as the receiver, its key spelled off.
	if comp := r.keyedReceiverElement(variable, line, caller, pr, baseDir, tr, ctx); comp != "" {
		return comp, false
	}

	if comp == "" {
		comp = r.untypedReceiverComponent(variable, lookupVar, line, caller, funcName, pr, baseDir, tr, ctx)
	}

	return comp, false
}

// funcScopedRef is the first ref for name inside the function enclosing line,
// among those scope admits.
func funcScopedRef(pr *parser.ParseResult, line uint32, name string, in parser.RefScope, calls ...*parser.CallSite) *parser.ComponentRef {
	var call *parser.CallSite
	if len(calls) > 0 {
		call = calls[0]
	}

	return funcScopedRefFor(pr, line, name, in, "", call)
}

// funcScopedRefFor is funcScopedRef for a call of method on the variable: a
// native ref without that method is passed over (nativeDefaultOverwritten),
// so an earlier or later assignment the variable does have is found instead.
func funcScopedRefFor(pr *parser.ParseResult, line uint32, name string, in parser.RefScope, method string, call *parser.CallSite) *parser.ComponentRef {
	// Each function's scope, then the file's top level. The scopes are read in
	// place: copying them to append the top level was 2.7 GB of a tassweb scan.
	for scopeIndex := 0; scopeIndex <= len(pr.Scopes); scopeIndex++ {
		var scope parser.FuncScope
		if scopeIndex < len(pr.Scopes) {
			scope = pr.Scopes[scopeIndex]
		} else {
			scope = parser.FuncScope{Start: 0, End: lineStarts.of(pr.Content, parser.NewLineTable).Lines() - 1}
		}

		if int(line) < scope.Start || int(line) > scope.End {
			continue
		}

		var refs []parser.ComponentRef
		if scopeIndex == len(pr.Scopes) {
			refs = pr.TopLevelClosureRefs()
		} else {
			refs, _ = pr.FuncRefs(scope.Start, scope.End)
		}

		// A closure holding line may declare the name itself, which shadows
		// the function's: the innermost closure's latest declaration at or
		// before line wins. Failing that, the function's own, and of those the
		// latest at or before line, as fileLevelRef chooses: `var exporter` in
		// one <cfcase> and again in the next is two variables in practice.
		// Only a forward reference takes the first.
		var closure, wide *parser.ComponentRef

		for i := range refs {
			ref := &refs[i]
			if scopeIndex == len(pr.Scopes) && !ref.HasClosureSpan {
				continue
			}

			if !strings.EqualFold(ref.Variable, name) || !ref.VisibleAtCall(line, call) || !in.Admits(ref) || assignedByCallAt(ref, line) {
				continue
			}

			// Runtime collection assignments describe the value after assignment,
			// never an incoming argument or local before that assignment.
			if ref.Line > line && strings.HasPrefix(ref.Component, parser.NativeTypePrefix) {
				continue
			}

			if nativeDefaultOverwritten(ref.Component, method) {
				continue
			}

			// An unknown later store cannot inherit a prior runtime interface.
			// Recovery is statement-line based; same-line and branch dominance
			// need a separate byte-aware flow proof.
			if strings.HasPrefix(ref.Component, parser.NativeTypePrefix) && nativeRefMissesUse(pr.Assignments(), pr.Content, name, scope.Start, int(ref.Line), int(line)) {
				continue
			}

			switch {
			case ref.VisibleTo == 0 && !ref.HasClosureSpan:
				if wide == nil || ref.Line <= line && (wide.Line > line || ref.Line > wide.Line) {
					wide = ref
				}
			case ref.Line <= line && (closure == nil || ref.ClosureFrom > closure.ClosureFrom || ref.VisibleFrom > closure.VisibleFrom ||
				ref.VisibleFrom == closure.VisibleFrom && ref.Line > closure.Line):
				closure = ref
			}
		}

		if closure != nil {
			return closure
		}

		if scopeIndex < len(pr.Scopes) {
			return reachingUnion(pr, pr.Assignments(), scope.Start, name, line, namedRefs(refs, name), wide)
		}

		return wide
	}

	return nil
}

// receiverMethod is the method called on the receiver itself: funcName, or
// for a chained call (`x.a().b()`, funcName b) the chain's first hop. A
// property hop (`x.p.b()`) calls nothing on x, so it names no method.
func receiverMethod(funcName string, call *parser.CallSite) string {
	if call != nil && len(call.Chain) > 0 {
		if _, property := parser.PropertyName(call.Chain[0]); property {
			return ""
		}

		return parser.CallHopName(call.Chain[0])
	}

	return funcName
}

// funcRefAfterDefault is ref, or when ref is a struct default the method
// contradicts (nativeDefaultOverwritten), the function-scoped ref the variable
// has that does not.
func funcRefAfterDefault(ref *parser.ComponentRef, pr *parser.ParseResult, line uint32, variable, name string, scope parser.RefScope, method string, call *parser.CallSite, tr *callTrace) *parser.ComponentRef {
	if ref == nil || !nativeDefaultOverwritten(ref.Component, method) {
		return ref
	}

	if alt := funcScopedRefFor(pr, line, name, scope, method, call); alt != nil || !structMember(ref, method) {
		tr.setAside(variable, ref, method)

		return alt
	}

	return ref
}

// fileRefAfterDefault is fileLevelRef, passing over a struct default the
// method contradicts as funcRefAfterDefault does.
func fileRefAfterDefault(pr *parser.ParseResult, line uint32, variable, name string, scope parser.RefScope, method string, tr *callTrace) *parser.ComponentRef {
	best := fileLevelRef(pr, line, name, scope)
	if best == nil || !nativeDefaultOverwritten(best.Component, method) {
		return best
	}

	if alt := fileLevelRefFor(pr, line, name, scope, method); alt != nil || !structMember(best, method) {
		tr.setAside(variable, best, method)

		return alt
	}

	return best
}

// structMember reports whether method is a member function the doc sources
// record for the struct ref holds. A default nothing later replaces still
// holds that struct, so Lucee's `myNumb = StructNew(); myNumb.sort()` keeps
// it; with a later ref the contract alone decides (nativeDefaultOverwritten),
// since a component's own sort() is likelier than a struct's.
func structMember(ref *parser.ComponentRef, method string) bool {
	return docs.LookupBuiltinMethod(strings.TrimPrefix(ref.Component, "$builtin."), method)
}

// setAside records a struct default passed over for a method it lacks.
func (t *callTrace) setAside(variable string, ref *parser.ComponentRef, method string) {
	t.addf("set aside %q's %q from line %d: it has no method %q, so the variable holds something assigned since", variable, ref.Component, ref.Line+1, method)
}

// nativeDefaultOverwritten reports whether a variable typed as a struct
// (`variables._parent = StructNew()`, `{}`) is called with a method that
// value does not have. A component's pseudo-constructor commonly sets such a
// default and init() then replaces it with an argument that is untyped or
// declared `struct`, and either way the variable reads as a struct, so every
// method called on what init() stored was checked against one. A method the native type
// lacks is the evidence the variable was reassigned, so the default is set
// aside and the lookup carries on (a componentResolver, the extends chain).
// With no method named, nothing contradicts it.
//
// Only a struct: it is what a placeholder default is (`StructNew()`, `{}`, a
// `type="struct"` argument CFML lets a component through), and a call on any
// other native value that its contract lacks is a finding to report, not a
// sign the variable moved on.
func nativeDefaultOverwritten(component, method string) bool {
	if method == "" || component != nativetype.Type("struct") {
		return false
	}

	// The struct's own contract, not the members the doc sources record for
	// any struct: a component replacing the default may well declare an
	// update(), find() or toJson() of its own, and keeping the struct for one
	// would accept the call without checking the component.
	_, ok := nativetype.Method(component, method)

	return !ok
}

// fileLevelRef is the file-level ref for name that is in force at line.
//
// A scratch variable can be reassigned multiple times in the same file (e.g.
// once per <cfswitch>/<cfcase> branch) — using the first matching ref in file
// order would lock onto whichever branch happens to appear earliest,
// regardless of which branch the call site is actually in. Prefer the ref
// with the highest line number at or before the call site (the assignment
// that's actually in scope there); only fall back to file order for a genuine
// forward reference, where no preceding ref exists.
func fileLevelRef(pr *parser.ParseResult, line uint32, name string, scope parser.RefScope) *parser.ComponentRef {
	return fileLevelRefFor(pr, line, name, scope, "")
}

// fileLevelRefFor is fileLevelRef for a call of method on the variable, passing
// over a native ref without that method as funcScopedRefFor does.
func fileLevelRefFor(pr *parser.ParseResult, line uint32, name string, scope parser.RefScope, method string) *parser.ComponentRef {
	var best *parser.ComponentRef

	named := pr.RefsNamed(name)

	for _, i := range named {
		ref := &pr.ComponentRefs[i]
		if ref.Line > line || !scope.Admits(ref) || assignedByCallAt(ref, line) || nativeDefaultOverwritten(ref.Component, method) {
			continue
		}

		if best == nil || ref.Line > best.Line {
			best = ref
		}
	}

	if best != nil {
		if topLevelLine(pr, line) {
			return reachingUnion(pr, pr.Assignments(), 0, name, line, func() []*parser.ComponentRef {
				found := make([]*parser.ComponentRef, 0, len(named))

				for _, i := range named {
					if ref := &pr.ComponentRefs[i]; scope.Admits(ref) {
						found = append(found, ref)
					}
				}

				return found
			}, best)
		}

		return best
	}

	for _, i := range named {
		if ref := &pr.ComponentRefs[i]; scope.Admits(ref) && !assignedByCallAt(ref, line) && !nativeDefaultOverwritten(ref.Component, method) {
			return ref
		}
	}

	return nil
}

// assignedByCallAt reports whether ref is the result of a call on line made
// on the variable ref assigns, `x = x.next()`: a receiver on that line is
// what the variable held before it. Mura writes
// `pluginEvent = pluginEvent.init( data ).getEvent()`, and init() was looked
// for in what getEvent() returns.
func assignedByCallAt(ref *parser.ComponentRef, line uint32) bool {
	return ref.Rebinds && ref.Line == line
}

// appRef is a ref for a name in one of the Application files.
type appRef struct {
	ref  *parser.ComponentRef
	file string
}

// maxApplicationHops bounds how many Application files above the nearest one
// applicationRefs will look through.
const maxApplicationHops = 8

// namesAnApplication reports whether text's code names an application, in
// either syntax: `this.name = …` in a component, `<cfapplication name="…">`
// or `cfapplication( name = "…" )`.
func namesAnApplication(text string) bool {
	if !mentions(text, "name") {
		return false
	}

	for _, code := range textCode(text) {
		for i := range code {
			switch {
			case isWord(code[i], "this") && i+3 < len(code) && code[i+1].Kind == parser.TokDot && isWord(code[i+2], "name") && isAssignAt(code, i+3):
				return true
			case isWord(code[i], "cfapplication") && i > 0 && code[i-1].Kind == parser.TokLT && code[i-1].Offset+1 == code[i].Offset:
				if attributeNamed(code[i+1:], "name", parser.TokGT) {
					return true
				}
			case isWord(code[i], "cfapplication") && i+1 < len(code) && code[i+1].Kind == parser.TokLParen:
				if end := producerGroupEnd(code, i+1, parser.TokLParen, parser.TokRParen); end > 0 && attributeNamed(code[i+2:end], "name", parser.TokEOF) {
					return true
				}
			}
		}
	}

	return false
}

// attributeNamed reports whether tokens, up to the first of kind stop, set
// name: `name =`.
func attributeNamed(tokens []parser.Token, name string, stop parser.TokenKind) bool {
	for j, t := range tokens {
		if t.Kind == stop {
			return false
		}

		if isWord(t, name) && isAssignAt(tokens, j+1) {
			return true
		}
	}

	return false
}

// applicationRefs is the first ref for name in the governing Application.cfc
// and then Application.cfm, for each that has one.
//
// The governing files are the nearest ones above baseDir. A guard file that
// names no application does not start one, so the application scope in a
// directory it guards is the one above: Galaxie Blog's org/camden/blog has an
// Application.cfm that only aborts a request for its ini file, and tags/ has a
// bare <cfabort>, while the root Application.cfc is where application.blog is
// set. When the nearest files hold no ref and name no application, the walk
// goes on to the next ones up; a file that names one (this.name,
// <cfapplication name>) is its own application and ends it.
func (r *Resolver) applicationRefs(baseDir, name string) []appRef {
	dir := baseDir

	for range maxApplicationHops {
		appDir := r.FindApplicationRoot(dir)
		if appDir == "" {
			return nil
		}

		if out := r.applicationRefsIn(appDir, name); len(out) > 0 {
			return out
		}

		if r.namesApplication(appDir) {
			return nil
		}

		parent := filepath.Dir(appDir)
		if parent == appDir {
			return nil
		}

		dir = parent
	}

	return nil
}

// applicationRefsIn is the first ref for name in appDir's Application.cfc and
// Application.cfm, for each that has one.
func (r *Resolver) applicationRefsIn(appDir, name string) []appRef {
	var out []appRef

	for _, appName := range []string{"Application.cfc", "Application.cfm"} {
		appURI := cfpath.ToURI(filepath.Join(appDir, appName))
		for _, ref := range r.Index.RefsForFile(appURI) {
			if strings.EqualFold(ref.Variable, name) {
				out = append(out, appRef{ref: ref, file: appName})

				break
			}
		}
	}

	return out
}

// namesApplication reports whether an Application file in appDir names an
// application. Cached per directory: it reads the files.
func (r *Resolver) namesApplication(appDir string) bool {
	r.mu.RLock()
	v, ok := r.appNameCache[appDir]
	r.mu.RUnlock()

	if ok {
		return v
	}

	for _, appName := range []string{"Application.cfc", "Application.cfm"} {
		text, err := r.readText(filepath.Join(appDir, appName))
		if err == nil && namesAnApplication(text) {
			v = true

			break
		}
	}

	r.mu.Lock()
	if r.appNameCache == nil {
		r.appNameCache = make(map[string]bool)
	}

	r.appNameCache[appDir] = v
	r.mu.Unlock()

	return v
}

// declaresArgument reports whether any function called caller declares an
// argument called name.
func declaresArgument(pr *parser.ParseResult, caller, name string) bool {
	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		if !strings.EqualFold(f.Name, caller) {
			continue
		}

		for _, arg := range f.Arguments {
			if strings.EqualFold(arg.Name, name) {
				return true
			}
		}
	}

	return false
}

// argumentOf is the argument called name of the function called caller, or
// nil. Only the first function with that name is consulted.
func argumentOf(pr *parser.ParseResult, caller, name string) *parser.Argument {
	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		if !strings.EqualFold(f.Name, caller) {
			continue
		}

		for j := range f.Arguments {
			if strings.EqualFold(f.Arguments[j].Name, name) {
				return &f.Arguments[j]
			}
		}

		return nil
	}

	return nil
}

// walkExtendsRefs walks the extends chain from extends, handing visit the
// first ref for name in each ancestor that has one, until visit returns true.
func (r *Resolver) walkExtendsRefs(extends, baseDir, name string, visit func(ref *parser.ComponentRef, parent string) bool) {
	seen := make(map[string]bool)

	for extends != "" && !seen[extends] {
		seen[extends] = true

		cfcPath := r.ComponentPath(extends, baseDir)
		if cfcPath == "" {
			return
		}

		parentURI := cfpath.ToURI(cfcPath)

		// Ensure the parent is indexed so RefsForFile returns its component refs.
		// (EnsureIndexed is a fast no-op if already indexed.)
		r.EnsureIndexed(cfcPath)

		for _, ref := range r.Index.RefsForFile(parentURI) {
			if strings.EqualFold(ref.Variable, name) {
				if visit(ref, extends) {
					return
				}

				break
			}
		}

		// Walk up the extends chain. The parent was just indexed, so its
		// extends is read from the index: a full parse of each parent on
		// every lookup was 43% of a Masa CMS scan once its display objects
		// walked through contentRenderer.cfc.
		next, ok := r.extendsOf(cfcPath, parentURI)
		if !ok {
			return
		}

		extends = next
	}
}

// softMissing reports whether comp, which names no file, came from a
// dynamicIfMissing resolver: during this resolution or during the parse.
func softMissing(comp, softComp string, pr *parser.ParseResult) bool {
	return (softComp != "" && strings.EqualFold(comp, softComp)) || pr.IsSoftComponent(comp)
}

// componentExists reports whether component, or any of its pipe-delimited
// alternatives, names a file. A dynamic or builtin marker counts as existing:
// it names no file by design.
func (r *Resolver) componentExists(component, baseDir string) bool {
	for alt := range strings.SplitSeq(component, "|") {
		switch {
		case alt == "":
			continue
		case strings.HasPrefix(alt, "$"):
			return true
		case filepath.IsAbs(alt):
			for _, p := range []string{alt, alt + ".cfc"} {
				if info, err := r.fs().Stat(p); err == nil && !info.IsDir() {
					return true
				}
			}
		}

		if r.ComponentPath(alt, baseDir) != "" {
			return true
		}
	}

	return false
}
