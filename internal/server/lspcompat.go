package server

import (
	"encoding/json/v2"

	"github.com/cfmleditor/clif/internal/lspdoc"
	"go.lsp.dev/protocol"
)

// optStr wraps a string for protocol fields typed as Optional[string].
func optStr(s string) protocol.Optional[string] {
	return protocol.NewOptional(s)
}

// tooltip wraps a string for protocol fields typed as InlayHintTooltip (documentation/message unions).
func tooltip(s string) protocol.InlayHintTooltip {
	return protocol.String(s)
}

// changeRangeAndText is lspdoc.ChangeRangeAndText.
func changeRangeAndText(change protocol.TextDocumentContentChangeEvent) (protocol.Range, string, bool) {
	return lspdoc.ChangeRangeAndText(change)
}

// argAs decodes the i'th ExecuteCommand argument (a raw JSON LSPAny) as a T,
// and reports whether it was there and of that type.
func argAs[T any](args []protocol.LSPAny, i int) (T, bool) {
	var v T

	if i < 0 || i >= len(args) {
		return v, false
	}

	if err := json.Unmarshal(args[i], &v); err != nil {
		var zero T

		return zero, false
	}

	return v, true
}

// argString decodes the i'th ExecuteCommand argument as a string.
func argString(args []protocol.LSPAny, i int) (string, bool) { return argAs[string](args, i) }

// argBool decodes the i'th ExecuteCommand argument as a bool.
//
// Unlike argString and argFloat it does not report whether the argument was
// there: a flag that is absent, and one that is present and false, both mean
// the caller is not asking for the behaviour. Distinguishing them would only
// invite a caller to treat "missing" as some third thing.
func argBool(args []protocol.LSPAny, i int) bool {
	v, _ := argAs[bool](args, i)

	return v
}

// argFloat decodes the i'th ExecuteCommand argument as a float64.
func argFloat(args []protocol.LSPAny, i int) (float64, bool) { return argAs[float64](args, i) }
