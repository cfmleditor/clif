package unresolved

import (
	"fmt"
	"io"
)

// SpeculativeMethodConnection is a per-call declaration shortcut, not a proven
// receiver identity or return chain. The original diagnostic remains available.
// Candidate locations and the original call use zero-based source lines.
type SpeculativeMethodConnection struct {
	Call
	Target       Candidate   `json:"target"`
	Alternatives []Candidate `json:"alternatives,omitempty"`
	TargetCount  int         `json:"targetCount"`
	Policy       string      `json:"policy"`
	Ownership    string      `json:"ownership"`
}

func classifyMethodShortcuts(rep *Report, verbose io.Writer) {
	var remaining []Call

	for i := range rep.Calls {
		call := &rep.Calls[i]

		targets := call.methodTargets
		if call.Unchecked > 0 || len(targets) == 0 {
			remaining = append(remaining, *call)

			continue
		}
		// Nearness ranks declarations. Lexical source order breaks ties; it does
		// not establish ownership. Alternatives remain available for later review.
		policy := "ranked-global-method"
		if len(targets) == 1 {
			policy = "unique-global-method"
		}

		count := len(targets)
		if len(targets) > maxCandidates {
			targets = targets[:maxCandidates]
		}

		listed := make([]Candidate, len(targets))
		copy(listed, targets)

		for i := range listed {
			listed[i].Confidence = ConfidenceLow
		}

		rep.SpeculativeMethodConnections = append(rep.SpeculativeMethodConnections, SpeculativeMethodConnection{
			Call: *call, Target: listed[0], Alternatives: listed[1:], TargetCount: count, Policy: policy, Ownership: "unproven",
		})
		if verbose != nil {
			_, _ = fmt.Fprintf(verbose, "  ? method shortcut %s:%d: %s -> %s:%d (%s; ownership and upstream chain unproven)\n", call.File, call.Line+1, call.CallText(), listed[0].File, listed[0].Line+1, policy)
		}
	}

	rep.Calls = remaining
}
