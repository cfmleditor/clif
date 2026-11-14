package server

import (
	"testing"

	"go.lsp.dev/protocol"
)

func TestWantsKind(t *testing.T) {
	refactor := protocol.CodeActionKindRefactor

	for _, c := range []struct {
		only []protocol.CodeActionKind
		want bool
	}{
		{nil, true},
		{[]protocol.CodeActionKind{protocol.CodeActionKindQuickFix}, false},
		{[]protocol.CodeActionKind{protocol.CodeActionKindQuickFix, protocol.CodeActionKindRefactor}, true},
		{[]protocol.CodeActionKind{"source"}, false},
	} {
		if got := wantsKind(c.only, refactor); got != c.want {
			t.Errorf("only %v: got %v, want %v", c.only, got, c.want)
		}
	}
}
