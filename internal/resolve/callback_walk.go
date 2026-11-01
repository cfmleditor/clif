package resolve

import (
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// callbackInvocationComponent is the component the provider passes in the
// callback's parameter slot: every invocation the provider can make must pass
// the same known component.
//
// The provider's plan is walked path by path. An invocation inside an if, a
// try or a loop counts as one at the top level does, with the value the
// argument holds on that path; statements that do not mention the callback
// are evaluated as the producer interpreter evaluates them anywhere. What
// names the callback is tracked per path too, so `arguments.input = subquery`
// after the invocation stops arguments.input naming it on that path, and a
// guard that proves a name holds no function (`isClosure( x ) ||
// isCustomFunction( x )` false, `isNull( x )` true) drops it on the branch it
// proves it for. A callback handed whole to another method of a known
// component is followed into that method, where its parameter names it.
// Anything else done with the callback (stored, passed to unknown code, part
// of a larger expression) gives no answer.
//
// qb's spec helper testCase() calls its callback inside a try, and QueryBuilder's
// union(), with() and join() guard theirs with isClosure() and replace it with
// the subquery it filled; unionAll() and withRecursive() hand it on.
func callbackInvocationComponent(eval *producerEvaluation, nodes []producerNode, env producerEnvironment, callback string, parameter int) string {
	w := callbackWalk{parameter: parameter}

	eval.wanted = map[string]bool{}
	markSetTargets(nodes, eval.wanted)

	w.walk(eval, nodes, env, callbackAliases(callback), nil)

	if w.failed || w.answer == "" || *eval.budget <= 0 {
		return ""
	}

	// Keep a source-relative result tied to the provider's actual declaration.
	if path := eval.resolver.ComponentPath(w.answer, filepath.Dir(eval.fd.URI.Path())); path != "" {
		return path
	}

	return w.answer
}

// maxCallbackForwards bounds how many methods deep a callback is followed.
const maxCallbackForwards = 4

// callbackWalk gathers the values a provider invokes a callback with.
type callbackWalk struct {
	answer    string
	parameter int
	forwards  int
	failed    bool
}

// callbackAliasSet is the names that hold the callback on one path.
type callbackAliasSet map[string]bool

// callbackAliases names a variable both qualified and bare, as CFML reads
// either for an argument or a local.
func callbackAliases(path string) callbackAliasSet {
	a := callbackAliasSet{}
	a.add(path)

	return a
}

func (a callbackAliasSet) add(path string) {
	path = normalizeProducerPath(path)
	a[path] = true

	if after, ok := strings.CutPrefix(path, "arguments."); ok {
		a[after] = true
	}
}

func (a callbackAliasSet) remove(path string) {
	path = normalizeProducerPath(path)
	delete(a, path)

	if after, ok := strings.CutPrefix(path, "arguments."); ok {
		delete(a, after)
	} else if !strings.Contains(path, ".") {
		delete(a, "arguments."+path)
	}
}

// walk runs nodes from env with aliases naming the callback, and returns the
// environment and aliases after them and whether every path stopped. observe,
// when set, collects every state the walk passes through, for the handlers of
// an enclosing try.
func (w *callbackWalk) walk(eval *producerEvaluation, nodes []producerNode, env producerEnvironment, aliases callbackAliasSet, observe *callbackState) (producerEnvironment, callbackAliasSet, bool) {
	for i := range nodes {
		if w.failed {
			return env, aliases, true
		}

		var stopped bool

		env, aliases, stopped = w.step(eval, &nodes[i], env, aliases, observe)

		if observe != nil {
			observe.join(env, aliases)
		}

		if stopped {
			return env, aliases, true
		}
	}

	return env, aliases, false
}

// callbackState is an environment and the aliases beside it, joined over
// paths.
type callbackState struct {
	env     producerEnvironment
	aliases callbackAliasSet
}

func (s *callbackState) join(env producerEnvironment, aliases callbackAliasSet) {
	s.env = mergeProducerEnvironments(s.env, env)
	maps.Copy(s.aliases, aliases)
}

func (w *callbackWalk) step(eval *producerEvaluation, node *producerNode, env producerEnvironment, aliases callbackAliasSet, observe *callbackState) (producerEnvironment, callbackAliasSet, bool) {
	if !callbackMentioned(node, aliases) {
		result := eval.execute([]producerNode{*node}, env)

		return result.environment, aliases, result.stopped
	}

	switch node.kind {
	case "set":
		return w.set(eval, node, env, aliases)
	case "effect", "return":
		if !w.invocation(eval, node.expression, env, aliases) && !w.forward(eval, producerTokens(node.expression), env, aliases) {
			w.failed = true
		}

		return env, aliases, node.kind == "return"
	case "block":
		return w.walk(eval, node.body, env, aliases, observe)
	case "if":
		tokens := producerTokens(node.expression)
		if !w.conditionSafe(eval, tokens, env, aliases) {
			w.failed = true

			return env, aliases, true
		}

		yesAliases, noAliases := maps.Clone(aliases), maps.Clone(aliases)
		narrowCallbackAliases(tokens, yesAliases, true)
		narrowCallbackAliases(tokens, noAliases, false)

		yes, yesAliases, yesStopped := w.walk(eval, node.body, maps.Clone(env), yesAliases, observe)
		no, noAliases, noStopped := w.walk(eval, node.alternative, maps.Clone(env), noAliases, observe)

		return joinCallbackPaths(env, aliases, yes, yesAliases, yesStopped, no, noAliases, noStopped)
	case "try":
		observed := &callbackState{env: maps.Clone(env), aliases: maps.Clone(aliases)}
		body, bodyAliases, bodyStopped := w.walk(eval, node.body, maps.Clone(env), maps.Clone(aliases), observed)

		if observe != nil {
			observe.join(observed.env, observed.aliases)
		}

		handled, handledAliases, handledStopped := w.walk(eval, node.alternative, observed.env, observed.aliases, observe)

		return joinCallbackPaths(env, aliases, body, bodyAliases, bodyStopped, handled, handledAliases, handledStopped)
	case "loop":
		return w.loop(eval, node, env, aliases, observe)
	default:
		w.failed = true

		return env, aliases, true
	}
}

// set is a walk step for an assignment that mentions the callback.
func (w *callbackWalk) set(eval *producerEvaluation, node *producerNode, env producerEnvironment, aliases callbackAliasSet) (producerEnvironment, callbackAliasSet, bool) {
	target := normalizeProducerPath(node.target)
	tokens := producerTokens(node.expression)
	rhs := normalizeProducerPath(producerPath(tokens))
	env = maps.Clone(env)
	aliases = maps.Clone(aliases)

	switch {
	case aliases[rhs]:
		// A copy of the callback names it too, unless it leaves the
		// function's own scopes.
		if !callbackLocalTarget(target) {
			w.failed = true
		}

		aliases.add(target)
		env[target] = producerUnknown()
	case w.invocation(eval, node.expression, env, aliases), w.forward(eval, tokens, env, aliases):
		aliases.remove(target)
		env[target] = producerUnknown()
	case callbackReferenceIn(tokens, aliases):
		w.failed = true
	default:
		// The callback's name is given another value.
		aliases.remove(target)

		if !eval.assign(node, env) {
			w.failed = true
		}
	}

	return env, aliases, false
}

// loop is a walk step for a loop that mentions the callback: its body is
// walked twice, the second time from what the first may have left, so an
// argument a later iteration holds is read as well.
func (w *callbackWalk) loop(eval *producerEvaluation, node *producerNode, env producerEnvironment, aliases callbackAliasSet, observe *callbackState) (producerEnvironment, callbackAliasSet, bool) {
	if callbackReferenceIn(producerTokens(node.header), aliases) {
		w.failed = true

		return env, aliases, true
	}

	invalidateProducerInstances(env)

	entry := maps.Clone(env)
	if node.target != "" {
		entry[normalizeProducerPath(node.target)] = producerUnknown()
	}

	first, firstAliases, firstStopped := w.walk(eval, node.body, maps.Clone(entry), maps.Clone(aliases), observe)
	if firstStopped {
		return env, aliases, false
	}

	again := maps.Clone(aliases)
	maps.Copy(again, firstAliases)

	second, secondAliases, _ := w.walk(eval, node.body, mergeProducerEnvironments(entry, first), again, observe)
	maps.Copy(again, secondAliases)

	return mergeProducerEnvironments(env, second), again, false
}

// callbackLocalTarget reports whether target is a name the function owns:
// a local or an argument.
func callbackLocalTarget(target string) bool {
	root, _, qualified := strings.Cut(target, ".")

	return !qualified || root == "arguments"
}

// joinCallbackPaths is the state after two alternative paths from env.
func joinCallbackPaths(env producerEnvironment, aliases callbackAliasSet, a producerEnvironment, aAliases callbackAliasSet, aStopped bool, b producerEnvironment, bAliases callbackAliasSet, bStopped bool) (producerEnvironment, callbackAliasSet, bool) {
	switch {
	case aStopped && bStopped:
		return env, aliases, true
	case aStopped:
		return b, bAliases, false
	case bStopped:
		return a, aAliases, false
	default:
		joined := maps.Clone(aAliases)
		maps.Copy(joined, bAliases)

		return mergeProducerEnvironments(a, b), joined, false
	}
}

// invocation reports whether expression is an invocation of the callback,
// and records the argument it passes in the callback's slot. An invocation
// that is part of a larger expression, passes the callback on, or passes an
// argument of no known component fails the walk.
func (w *callbackWalk) invocation(eval *producerEvaluation, expression string, env producerEnvironment, aliases callbackAliasSet) bool {
	tokens := producerTokens(expression)

	open := slices.IndexFunc(tokens, func(t parser.Token) bool { return t.Kind == parser.TokLParen })
	if open < 0 || !aliases[normalizeProducerPath(producerPath(tokens[:open]))] {
		return false
	}

	if producerGroupEnd(tokens, open, parser.TokLParen, parser.TokRParen) != len(tokens)-1 {
		w.failed = true

		return false
	}

	pieces := producerSplit(tokens[open+1 : len(tokens)-1])
	if w.parameter >= len(pieces) || callbackReferenceIn(tokens[open+1:], aliases) {
		w.failed = true

		return false
	}

	value := eval.expression(producerText(pieces[w.parameter]), env).component()
	if value == "" || w.answer != "" && !strings.EqualFold(w.answer, value) {
		w.failed = true

		return false
	}

	w.answer = value

	return true
}

// forward reports whether tokens are a call that hands the callback whole
// to a method of a known component, and follows it there: the method's
// parameter that receives it names the callback in its body. A method that
// never invokes it adds nothing; one whose plan cannot be read fails the walk.
func (w *callbackWalk) forward(eval *producerEvaluation, tokens []parser.Token, env producerEnvironment, aliases callbackAliasSet) bool {
	fd, method, args, ok := w.forwardTarget(eval, tokens, env, aliases)
	if !ok {
		return false
	}

	if fd == nil {
		return true
	}

	if w.forwards >= maxCallbackForwards {
		w.failed = true

		return true
	}

	local := *eval
	local.fd = fd
	local.receiver = args.receiver
	local.callerArguments = false
	local.instance = nil
	local.wanted = map[string]bool{}
	markSetTargets(method.body, local.wanted)

	inner := callbackWalk{answer: w.answer, parameter: w.parameter, forwards: w.forwards + 1}
	inner.walk(&local, method.body, local.bindArguments(fd, method, args.values, args.known), callbackAliases("arguments."+args.parameter), nil)

	if inner.failed {
		w.failed = true
	}

	w.answer = inner.answer

	return true
}

// callbackForwardArgs is what a forwarding call passes: the callee's
// parameter that receives the callback and the other arguments' values.
type callbackForwardArgs struct {
	values    map[string]producerValue
	parameter string
	receiver  string
	known     bool
}

// forwardTarget reads a call that hands the callback on. ok is false when
// tokens are not one call passing a name of the callback whole as one
// argument. fd is nil for a builtin predicate, which inspects its argument
// and never invokes it.
func (w *callbackWalk) forwardTarget(eval *producerEvaluation, tokens []parser.Token, env producerEnvironment, aliases callbackAliasSet) (fd *parser.FunctionDef, method *producerMethod, args callbackForwardArgs, ok bool) {
	open := slices.IndexFunc(tokens, func(t parser.Token) bool { return t.Kind == parser.TokLParen })

	// The call's name is the identifier before the argument list it ends
	// with; what comes before it is the receiver.
	last := -1

	for i := open; i >= 0 && i < len(tokens); {
		end := producerGroupEnd(tokens, i, parser.TokLParen, parser.TokRParen)
		if end < 0 {
			return nil, nil, args, false
		}

		if end == len(tokens)-1 {
			last = i

			break
		}

		next := slices.IndexFunc(tokens[end+1:], func(t parser.Token) bool { return t.Kind == parser.TokLParen })
		if next < 0 {
			return nil, nil, args, false
		}

		i = end + 1 + next
	}

	if last < 1 || tokens[last-1].Kind != parser.TokIdent {
		return nil, nil, args, false
	}

	name := tokens[last-1].Value
	if callbackReferenceIn(tokens[:last-1], aliases) {
		return nil, nil, args, false
	}

	slot, named, passed := callbackSlot(producerSplit(tokens[last+1:len(tokens)-1]), aliases)
	if !passed {
		return nil, nil, args, false
	}

	receiver := tokens[:max(last-2, 0)]
	if last >= 2 && tokens[last-2].Kind != parser.TokDot {
		return nil, nil, args, false
	}

	target := eval.fd.URI.Path()
	if eval.receiver != "" {
		target = eval.receiver
	}

	if len(receiver) > 0 {
		root := normalizeProducerPath(producerPath(receiver))
		if root != "this" && root != "variables" {
			target = eval.expression(producerText(receiver), env).component()
		}
	}

	if len(receiver) == 0 && callbackPredicate(name) && eval.resolver.ResolveFunc(target, name, eval.baseDir) == nil {
		return nil, nil, args, true
	}

	if target == "" || strings.Contains(target, "|") {
		return nil, nil, args, false
	}

	fd = eval.resolver.ResolveFunc(target, name, eval.baseDir)
	if fd == nil {
		return nil, nil, args, false
	}

	method = eval.resolver.producerFor(fd)
	if method == nil {
		return nil, nil, args, false
	}

	switch {
	case named == "argumentcollection":
		args.parameter = strings.TrimPrefix(slot, "arguments.")
	case named != "":
		args.parameter = named
	default:
		index, err := strconv.Atoi(strings.TrimPrefix(slot, "@"))
		if err != nil || index >= len(method.parameters) {
			return nil, nil, args, false
		}

		args.parameter = method.parameters[index]
	}

	if !slices.Contains(method.parameters, args.parameter) {
		return nil, nil, args, false
	}

	pieces := producerSplit(tokens[last+1 : len(tokens)-1])
	if named != "argumentcollection" {
		args.values, args.known = eval.arguments(callbackValueArguments(pieces, 0, callbackPieceOffset(pieces, slot, named)), env)
	}

	args.receiver = target

	return fd, method, args, true
}

// callbackSlot finds the one argument that passes the callback whole: its
// position ("@n") or, for a named argument, its name. `argumentCollection =
// arguments` passes every argument under its own name, the callback's
// included, and is reported with the callback's name as the slot.
func callbackSlot(pieces [][]parser.Token, aliases callbackAliasSet) (slot, named string, ok bool) {
	found := false

	for i, piece := range pieces {
		expr, name := piece, ""
		if len(piece) > 2 && piece[0].Kind == parser.TokIdent && (piece[1].Kind == parser.TokEquals || piece[1].Kind == parser.TokColon) {
			expr, name = piece[2:], strings.ToLower(piece[0].Value)
		}

		path := normalizeProducerPath(producerPath(expr))

		switch {
		case name == "argumentcollection" && path == "arguments":
			for alias := range aliases {
				if strings.HasPrefix(alias, "arguments.") {
					slot = alias
				}
			}

			if slot == "" || found {
				return "", "", false
			}

			found, named = true, name
		case aliases[path]:
			if found {
				return "", "", false
			}

			found, named, slot = true, name, producerPosition(i)
		case callbackReferenceIn(expr, aliases):
			return "", "", false
		}
	}

	return slot, named, found
}

// callbackPieceOffset is the offset callbackValueArguments recognises the
// callback's argument by: the first token of the piece in the slot.
func callbackPieceOffset(pieces [][]parser.Token, slot, named string) int {
	for i, piece := range pieces {
		expr, name := piece, ""
		if len(piece) > 2 && piece[0].Kind == parser.TokIdent && (piece[1].Kind == parser.TokEquals || piece[1].Kind == parser.TokColon) {
			expr, name = piece[2:], strings.ToLower(piece[0].Value)
		}

		if (named != "" && name == named || named == "" && producerPosition(i) == slot) && len(expr) > 0 {
			return expr[0].Offset
		}
	}

	return -1
}

// callbackPredicate reports whether name is a builtin that inspects a value
// and never invokes it.
func callbackPredicate(name string) bool {
	switch strings.ToLower(name) {
	case "isclosure", "iscustomfunction", "isnull", "issimplevalue", "isstruct", "isarray",
		"isobject", "isquery", "isvalid", "isinstanceof", "isboolean", "isnumeric", "isdate":
		return true
	}

	return false
}

// conditionSafe reports whether every mention of the callback in an if's
// condition hands it whole to a predicate or to a method the walk can follow
// (forward); anything else might invoke it unseen or keep it.
func (w *callbackWalk) conditionSafe(eval *producerEvaluation, tokens []parser.Token, env producerEnvironment, aliases callbackAliasSet) bool {
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Kind != parser.TokLParen {
			continue
		}

		end := producerGroupEnd(tokens, i, parser.TokLParen, parser.TokRParen)
		if end < 0 {
			return false
		}

		if !callbackReferenceIn(tokens[i:end+1], aliases) {
			i = end

			continue
		}

		if i == 0 || tokens[i-1].Kind != parser.TokIdent {
			// A bracketing group: look inside it.
			continue
		}

		// The call starts at the receiver chain before the name.
		start := i - 1

	receiver:
		for start >= 2 && tokens[start-1].Kind == parser.TokDot {
			switch tokens[start-2].Kind {
			case parser.TokRParen:
				open := producerGroupStart(tokens, start-2)
				if open < 1 || tokens[open-1].Kind != parser.TokIdent {
					return false
				}

				start = open - 1
			case parser.TokIdent:
				start -= 2
			default:
				break receiver
			}
		}

		if !w.forward(eval, tokens[start:end+1], env, aliases) || w.failed {
			return false
		}

		i = end
	}

	// Every mention must have been inside a call handled above.
	return !callbackReferenceIn(callbackOutsideCalls(tokens), aliases)
}

// producerGroupStart is the index of the opening parenthesis that the one at
// end closes, or -1.
func producerGroupStart(tokens []parser.Token, end int) int {
	depth := 0

	for i := end; i >= 0; i-- {
		switch tokens[i].Kind {
		case parser.TokRParen:
			depth++
		case parser.TokLParen:
			depth--
			if depth == 0 {
				return i
			}
		default:
		}
	}

	return -1
}

// callbackOutsideCalls is tokens with every call's argument list removed,
// leaving what a condition does with values directly.
func callbackOutsideCalls(tokens []parser.Token) []parser.Token {
	var out []parser.Token

	for i := 0; i < len(tokens); i++ {
		if tokens[i].Kind == parser.TokLParen && i > 0 && tokens[i-1].Kind == parser.TokIdent {
			if end := producerGroupEnd(tokens, i, parser.TokLParen, parser.TokRParen); end > 0 {
				i = end

				continue
			}
		}

		out = append(out, tokens[i])
	}

	return out
}

// narrowCallbackAliases drops from aliases the names an if's condition, taken
// as truth, proves hold no function: isNull( x ), or isClosure( x ) and
// isCustomFunction( x ) both false.
func narrowCallbackAliases(tokens []parser.Token, aliases callbackAliasSet, truth bool) {
	facts := callbackFacts(tokens, truth)

	for name, f := range facts {
		if f&factNull != 0 || f&factNotClosure != 0 && f&factNotCustom != 0 {
			aliases.remove(name)
		}
	}
}

// What a condition proves about a name's value.
const (
	factNull = 1 << iota
	factNotClosure
	factNotCustom
)

// callbackFacts is what holds of each name in every case where the
// condition tokens evaluate to truth.
func callbackFacts(tokens []parser.Token, truth bool) map[string]int {
	tokens = producerUnwrap(tokens)

	if parts := splitBoolean(tokens, "||", "or"); len(parts) > 1 {
		return combineFacts(parts, truth, !truth)
	}

	if parts := splitBoolean(tokens, "&&", "and"); len(parts) > 1 {
		return combineFacts(parts, truth, truth)
	}

	if len(tokens) > 1 && (tokens[0].Kind == parser.TokBang || strings.EqualFold(tokens[0].Value, "not")) {
		return callbackFacts(tokens[1:], !truth)
	}

	if len(tokens) < 4 || tokens[0].Kind != parser.TokIdent || tokens[1].Kind != parser.TokLParen || tokens[len(tokens)-1].Kind != parser.TokRParen || producerGroupEnd(tokens, 1, parser.TokLParen, parser.TokRParen) != len(tokens)-1 {
		return nil
	}

	name := strings.TrimPrefix(normalizeProducerPath(producerPath(tokens[2:len(tokens)-1])), "arguments.")
	if name == "" {
		return nil
	}

	switch strings.ToLower(tokens[0].Value) {
	case "isnull":
		if truth {
			return map[string]int{name: factNull}
		}
	case "isclosure":
		if !truth {
			return map[string]int{name: factNotClosure}
		}
	case "iscustomfunction":
		if !truth {
			return map[string]int{name: factNotCustom}
		}
	}

	return nil
}

// combineFacts joins the parts' facts: every part takes the value each when
// all, so their facts add up; otherwise only what every part proves holds.
func combineFacts(parts [][]parser.Token, each bool, all bool) map[string]int {
	if all {
		facts := map[string]int{}

		for _, part := range parts {
			for name, f := range callbackFacts(part, each) {
				facts[name] |= f
			}
		}

		return facts
	}

	var facts map[string]int

	for i, part := range parts {
		got := callbackFacts(part, each)
		if i == 0 {
			facts = got

			continue
		}

		for name, f := range facts {
			facts[name] = f & got[name]
		}
	}

	return facts
}

// splitBoolean splits tokens at the top-level operator spelled op (as two
// tokens) or word, or returns nil when there is none.
func splitBoolean(tokens []parser.Token, op, word string) [][]parser.Token {
	var (
		parts [][]parser.Token
		depth int
		from  int
	)

	for i := 0; i < len(tokens); i++ {
		switch tokens[i].Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth--
		default:
		}

		if depth != 0 {
			continue
		}

		width := 0

		switch {
		case strings.EqualFold(tokens[i].Value, word) && tokens[i].Kind == parser.TokIdent:
			width = 1
		case i+1 < len(tokens) && tokens[i].Value+tokens[i+1].Value == op:
			width = 2
		}

		if width > 0 {
			parts = append(parts, tokens[from:i])
			from = i + width
			i += width - 1
		}
	}

	if parts == nil {
		return nil
	}

	return append(parts, tokens[from:])
}

// callbackMentioned reports whether node, or anything nested in it, names
// the callback, or the whole arguments scope while an argument names it.
func callbackMentioned(node *producerNode, aliases callbackAliasSet) bool {
	for _, text := range []string{node.target, node.expression, node.header} {
		if text == "" {
			continue
		}

		tokens := producerTokens(text)
		if callbackReferenceIn(tokens, aliases) || callbackScopeIn(tokens, aliases) {
			return true
		}
	}

	for _, nodes := range [][]producerNode{node.body, node.alternative} {
		for i := range nodes {
			if callbackMentioned(&nodes[i], aliases) {
				return true
			}
		}
	}

	return false
}

// callbackScopeIn reports whether tokens use the arguments scope whole while
// one of its keys names the callback: argumentCollection = arguments hands it
// on as surely as naming it.
func callbackScopeIn(tokens []parser.Token, aliases callbackAliasSet) bool {
	argument := false

	for alias := range aliases {
		if strings.HasPrefix(alias, "arguments.") {
			argument = true
		}
	}

	if !argument {
		return false
	}

	for i, t := range tokens {
		if t.Kind == parser.TokIdent && strings.EqualFold(t.Value, "arguments") && (i+1 >= len(tokens) || tokens[i+1].Kind != parser.TokDot) && (i == 0 || tokens[i-1].Kind != parser.TokDot) {
			return true
		}
	}

	return false
}

// markSetTargets wants every variable the plan assigns, so that the walk
// keeps what each holds wherever it is assigned.
func markSetTargets(nodes []producerNode, wanted map[string]bool) {
	for i := range nodes {
		if nodes[i].kind == "set" {
			wanted[normalizeProducerPath(nodes[i].target)] = true
		}

		markSetTargets(nodes[i].body, wanted)
		markSetTargets(nodes[i].alternative, wanted)
	}
}
