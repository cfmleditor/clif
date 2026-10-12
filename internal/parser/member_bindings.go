package parser

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// MemberReceiverName preserves a record's path. A scoped variable and an
// unrelated record member with the same final name are different bindings.
func MemberReceiverName(variable string) (string, RefScope, bool) {
	scope, name, ok := strings.Cut(variable, ".")
	if !ok {
		return "", RefAny, false
	}

	switch {
	case strings.EqualFold(scope, "variables"):
		return name, RefVariables, strings.Contains(name, ".")
	case strings.EqualFold(scope, "this"):
		return name, RefThis, strings.Contains(name, ".")
	case strings.EqualFold(scope, "local"), strings.EqualFold(scope, "arguments"):
		return name, RefAny, strings.Contains(name, ".")
	case strings.EqualFold(scope, "application"), strings.EqualFold(scope, "request"), strings.EqualFold(scope, "session"), strings.EqualFold(scope, "server"):
		return variable, RefAny, strings.Contains(name, ".")
	default:
		return variable, RefAny, true
	}
}

// Explicit writes type one member, never its whole container. Conflicting or
// unknown writes and parent replacements withhold the member's component.
// applyMemberBindings types the members a component assigns. hasBindings is
// hasMemberBinding of the content, which the caller has already asked.
func (pr *ParseResult) applyMemberBindings(hasBindings bool) {
	if !hasBindings {
		return
	}

	writes := pr.collectionWrites()
	locals, _, key := pr.collectionKeys(writes)
	nodes := map[string]*collectionNode{}
	identities := map[string]collectionWrite{}
	dependencies := map[string][]memberDependency{}

	for _, w := range writes {
		if w.element {
			continue
		}

		if _, _, ok := MemberReceiverName(w.target); !ok {
			continue
		}

		identity := key(w.target, w.function)

		n := nodes[identity]
		if n == nil {
			n = &collectionNode{}
			nodes[identity] = n
			identities[identity] = w
		}

		if receiver, methods := pr.memberWriteDependency(w); receiver != "" {
			dependencies[identity] = append(dependencies[identity], memberDependency{key(receiver, w.function), methods})

			continue
		}

		source := pr.memberValueSource(w.expression, w.function, locals)
		if w.unknown || source.Component == "" || strings.HasPrefix(source.Component, "$") {
			n.invalid = true

			continue
		}

		n.addSource(source)
	}

	descendants := memberDescendants(nodes)

	spans := newMemberSpans(nodes, identities, writes, key)

	for _, w := range writes {
		if w.target == "" && w.unknown {
			withholdForClosure(nodes, w.function)

			continue
		}

		// One scanner reads the expression both ways: there is one write per
		// assignment in the file, and a scanner each was most of what this
		// loop allocated.
		sc := NewScanner(w.expression)
		start := sc.Save()

		if !w.element && emptyCollectionAt(sc) {
			continue
		}

		sc.Restore(start)

		if source, indexed := collectionReadAt(sc); source != "" && !indexed {
			invalidate(descendants[key(source, w.function)])
		}

		parent := key(w.target, w.function)
		if w.element && (strings.EqualFold(w.target, "variables") || strings.EqualFold(w.target, "this")) {
			parent = strings.ToLower(w.target)
		}

		spans.replaced(pr, descendants[parent], w.offset)
	}

	components := pr.settleMemberDependencies(nodes, dependencies)

	// In the order the members are first written. Ranging over nodes put a
	// function's refs in a different order on every parse of the same text.
	order := slices.SortedFunc(maps.Keys(nodes), func(a, b string) int {
		return cmp.Or(cmp.Compare(identities[a].offset, identities[b].offset), strings.Compare(a, b))
	})

	for _, identity := range order {
		at, cleared := spans.cleared[nodes[identity]]
		pr.fileMemberRef(identity, identities[identity], components[identity], at, cleared)
	}
}

// memberSpans is where each member is first and last written, and where a
// replacement of its container after the last write ends it. A container
// replaced before the first write (`var temp = 0;` then
// `temp.u = new U()`) does not touch the member. One replaced after the last
// ends it there: the member keeps its type above that line, which is where
// itext.cfc's init() copies `temp.itextUtils` into the component before
// reusing temp for something else. A replacement between two writes, or on
// the line of one, withholds it, as any conflict does.
type memberSpans struct {
	first, last, cleared map[*collectionNode]int
}

func newMemberSpans(nodes map[string]*collectionNode, identities map[string]collectionWrite, writes []collectionWrite, key func(string, string) string) *memberSpans {
	s := &memberSpans{
		first:   make(map[*collectionNode]int, len(nodes)),
		last:    make(map[*collectionNode]int, len(nodes)),
		cleared: map[*collectionNode]int{},
	}

	for identity, n := range nodes {
		s.first[n] = identities[identity].offset
	}

	for _, w := range writes {
		if n := nodes[key(w.target, w.function)]; n != nil && !w.element {
			s.last[n] = max(s.last[n], w.offset)
		}
	}

	return s
}

// replaced applies a replacement of the container of members at offset.
func (s *memberSpans) replaced(pr *ParseResult, members []*collectionNode, offset int) {
	for _, n := range members {
		switch {
		case offset < s.first[n]:
		case pr.lineAt(s.last[n]) < pr.lineAt(offset):
			// On a later line, so a lookup's latest-at-or-before rule tells
			// the member's ref from the replacement's.
			if at, ok := s.cleared[n]; !ok || offset < at {
				s.cleared[n] = offset
			}
		default:
			n.invalid = true
		}
	}
}

// fileMemberRef files the ref for the member identity first written by w,
// and, when its container is replaced at clearedAt, an untyped ref there.
func (pr *ParseResult) fileMemberRef(identity string, w collectionWrite, component string, clearedAt int, cleared bool) {
	name, scope, _ := MemberReceiverName(w.target)
	ref := ComponentRef{Variable: strings.Clone(name), Component: component, URI: pr.URI, Line: conv.Uint32(pr.lineAt(w.offset)), This: scope == RefThis}
	refs := []ComponentRef{ref}

	if cleared && component != "" {
		// The container's replacement: from here the member holds nothing.
		ref.Component, ref.Line = "", conv.Uint32(pr.lineAt(clearedAt))
		refs = append(refs, ref)
	}

	if strings.Contains(identity, "\t") {
		if pr.funcRefsMap == nil {
			pr.funcRefsMap = map[string][]ComponentRef{}
		}

		pr.funcRefsMap[w.function] = append(pr.funcRefsMap[w.function], refs...)
	} else {
		pr.ComponentRefs = append(pr.ComponentRefs, refs...)
	}
}

// memberDescendants files each node under every proper prefix of its identity
// that ends before a dot, so the members a write replaces are one lookup away.
// Every write used to test every node's identity against its own key, which
// made a component holding many members quadratic in them.
func memberDescendants(nodes map[string]*collectionNode) map[string][]*collectionNode {
	descendants := map[string][]*collectionNode{}

	for identity, n := range nodes {
		for i := range len(identity) {
			if identity[i] == '.' {
				descendants[identity[:i]] = append(descendants[identity[:i]], n)
			}
		}
	}

	return descendants
}

// withholdForClosure invalidates the members a closure written in function
// could write: that function's own, which its parameters may shadow, and the
// component's, which it reaches as any function does. Another function's
// locals and arguments are out of its reach. A closure outside any function
// withholds every member. It used to withhold every member in the file, so
// one closure in a ColdBox handler untyped `prc.x = svc.get()` in every
// other action.
func withholdForClosure(nodes map[string]*collectionNode, function string) {
	for identity, n := range nodes {
		owner, _, local := strings.Cut(identity, "\t")
		if function == "" || !local || owner == function {
			n.invalid = true
		}
	}
}

func invalidate(nodes []*collectionNode) {
	for _, n := range nodes {
		n.invalid = true
	}
}

func (pr *ParseResult) memberSourceComponent(node *collectionNode) string {
	if node.invalid {
		return ""
	}

	answer := ""

	for _, source := range node.sources {
		component := source.Component
		if pr.FuncLookup != nil {
			component = pr.walkChainRest(component, source.Methods)
		} else {
			component = dynamicIfTyped(component, source.Methods)
		}

		if component == "" || strings.HasPrefix(component, "$") || answer != "" && !strings.EqualFold(answer, component) {
			return ""
		}

		answer = component
	}

	return answer
}

func (pr *ParseResult) hasMemberBinding(content string) bool {
	for offset := strings.IndexByte(content, '='); offset >= 0; {
		if pr.memberAssignmentCandidate(content, offset) {
			return true
		}

		next := strings.IndexByte(content[offset+1:], '=')
		if next < 0 {
			return false
		}

		offset += next + 1
	}

	return false
}

func (pr *ParseResult) memberAssignmentCandidate(content string, offset int) bool {
	if offset == 0 || content[offset-1] == '=' || offset+1 < len(content) && content[offset+1] == '=' {
		return false
	}

	end := offset
	for end > 0 && (content[end-1] == ' ' || content[end-1] == '\t') {
		end--
	}

	start := end
	for start > 0 && (isIdentPart(content[start-1]) || content[start-1] == '.') {
		start--
	}

	if _, _, ok := MemberReceiverName(content[start:end]); !ok {
		return false
	}

	cursor := NewScanner(content[offset+1:])
	expression := mappingSourceExpression(cursor)

	first := NewScanner(expression).PeekSkipComments()
	if first.Kind == TokIdent && (identEq(first.Value, "new") || identEq(first.Value, "createObject")) {
		return true
	}

	if pr.FuncLookup != nil || len(pr.Resolvers) > 0 {
		return true
	}

	root, element := collectionRead(expression)
	if root == "" || element {
		return false
	}

	name := StripReceiverScope(root)
	for _, refs := range pr.funcRefsMap {
		if concreteMemberRef(refs, name) {
			return true
		}
	}

	return concreteMemberRef(pr.ComponentRefs, name)
}

func concreteMemberRef(refs []ComponentRef, name string) bool {
	ref := firstRefNamed(refs, name)

	return ref != nil && ref.Component != "" && !strings.HasPrefix(ref.Component, "$")
}

func (pr *ParseResult) memberValueSource(expression, function string, locals map[string]bool) ReturnSource {
	// The legacy scalar-chain parser strips a receiver's last dot segment.
	// Do not let a record read acquire the type of an unrelated scalar.
	if before, _, ok := strings.Cut(expression, "("); ok {
		if receiver, _, ok := strings.CutLast(strings.TrimSpace(before), "."); ok {
			if _, _, record := MemberReceiverName(receiver); record {
				root, _ := factoryReturnChain(expression)
				if pr.resolverSet == nil || pr.resolverSet.Resolve(root) == "" {
					return ReturnSource{}
				}
			}
		}
	}

	if source := pr.collectionValueSource(expression, function, locals); source.Component != "" {
		return source
	}

	root, element := collectionRead(expression)
	if _, _, record := MemberReceiverName(root); record {
		return ReturnSource{}
	}

	if root == "" || element {
		return ReturnSource{}
	}

	name := StripReceiverScope(root)
	scope := ReceiverRefScope(root)

	ref := firstRefIn(pr.funcRefsMap[function], name, scope)
	if scope != RefAny {
		ref = firstRefIn(pr.ComponentRefs, name, scope)
	}

	if ref == nil && (hasPrefixFold(root, "arguments.") || hasPrefixFold(root, "local.") || locals[function+"\t"+strings.ToLower(name)]) {
		return ReturnSource{}
	}

	if ref == nil {
		ref = firstRefIn(pr.ComponentRefs, name, scope)
	}

	if ref == nil || ref.VisibleTo != 0 {
		return ReturnSource{}
	}

	return ReturnSource{Component: pr.settledComponent(ref)}
}
