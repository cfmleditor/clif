package resolve

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// Pending is a classification left open because the rules that decide it
// compare how many components declare a name with a threshold, and the index
// is still growing (ClassifyOptions.DeferCounts). It holds what the open rules
// read of the call and the chain of rules in order, so settling it after the
// run asks the same questions of the finished index that an immediate
// classification would have asked of whatever the index held at the time.
type Pending struct {
	call   parser.CallSite
	reason string
	slots  []pendingSlot
}

// pendingSlot is one rule of the chain that could still decide the call: a
// count rule, to be asked at settle time, or the first rule that applies
// whatever the counts say, which ends the chain.
type pendingSlot struct {
	rule  int
	known bool
}

// deferrable reports whether reason is one the count rules decide: a receiver
// the source never typed. Any other reason keeps its immediate verdict.
func deferrable(reason string) bool {
	if !strings.HasPrefix(reason, "variable '") || !strings.HasSuffix(reason, "' has no component ref") {
		return false
	}

	_, missingBase := MissingBaseOf(reason)

	return !missingBase
}

// classifyPending is ClassifyReasonRule for a run that settles its count rules
// later. It returns a verdict when no count rule can matter: the call is a
// built-in, or a rule that reads no count applies before any count rule could.
// Otherwise it returns a Pending and the verdict is a placeholder.
func (r *Resolver) classifyPending(call *parser.CallSite, reason string, opt ClassifyOptions, pr *parser.ParseResult, baseDir string) (v Verdict, base, rule string, pending *Pending) {
	if reason == "" || IsBuiltinCall(call) || !deferrable(reason) {
		v, base, rule = r.ClassifyReasonRule(call, reason, opt, pr, baseDir)

		return v, base, rule, nil
	}

	in := classifyInput{call: call, reason: reason, opt: opt, pr: pr, baseDir: baseDir}

	var slots []pendingSlot

	for i := range classifyRules {
		cr := &classifyRules[i]
		if cr.judgement && opt.KeepUntypedHops {
			continue
		}

		if cr.guard != nil {
			if cr.guard(r, &in) {
				slots = append(slots, pendingSlot{rule: i})
			}

			continue
		}

		if !cr.applies(r, &in) {
			continue
		}

		if len(slots) == 0 {
			return cr.verdict, "", cr.name, nil
		}

		slots = append(slots, pendingSlot{rule: i, known: true})

		break
	}

	if len(slots) == 0 {
		return Unresolved, "", "", nil
	}

	// Only what the count rules read is kept: a call holds its line of source,
	// and a Pending outlives the file.
	held := parser.CallSite{
		Variable: strings.Clone(call.Variable),
		FuncName: strings.Clone(call.FuncName),
		Chain:    slices.Clone(call.Chain),
	}

	return Unresolved, "", "", &Pending{call: held, reason: reason, slots: slots}
}

// Settle decides a pending call against the index as it now stands, and names
// the rule that decided it ("" for Unresolved). Call it once the run has
// indexed everything it will.
func (p *Pending) Settle(r *Resolver) (v Verdict, rule string) {
	in := classifyInput{call: &p.call, reason: p.reason}

	for _, s := range p.slots {
		cr := &classifyRules[s.rule]
		if s.known || cr.applies(r, &in) {
			return cr.verdict, cr.name
		}
	}

	return Unresolved, ""
}
