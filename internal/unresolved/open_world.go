package unresolved

import (
	"fmt"
	"io"
	"strings"
)

// AssumedCall retains the original diagnostic. Open-world assumptions are
// triage classifications, not definitions, verified fixes or resolved calls.
type AssumedCall struct {
	Call
	Policy string `json:"policy"`
}

func openWorldPolicy(call *Call) string {
	if call.Unchecked > 0 {
		return ""
	}

	switch Category(call.Reason) {
	case CategoryVariable:
		return "runtime-receiver"
	case CategoryReturnType:
		return "runtime-return"
	case CategoryObject:
		// Preserve broken inheritance and chains. Only absent literal component
		// paths may be supplied by deployment mappings or external packages.
		if componentRe.MatchString(call.Reason) {
			return "external-component"
		}
	case CategoryMethod:
		if !call.This && call.Variable == "" && (call.Reason == "no qualifier, not in file" || call.Reason == "not found in extends chain") {
			return "runtime-function-mixin"
		}
	}

	return ""
}

func classifyOpenWorld(rep *Report, verbose io.Writer) {
	var remaining []Call

	for i := range rep.Calls {
		call := &rep.Calls[i]

		policy := openWorldPolicy(call)
		if policy == "" {
			remaining = append(remaining, *call)

			continue
		}

		rep.AssumedCalls = append(rep.AssumedCalls, AssumedCall{Call: *call, Policy: policy})
		if verbose != nil {
			_, _ = fmt.Fprintf(verbose, "  ? assumed %s:%d: %s (%s; %s)\n", call.File, call.Line+1, call.CallText(), policy, strings.TrimSpace(call.Reason))
		}
	}

	rep.Calls = remaining
}
