package server

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

// Adapted from IntelliLucee's CFMLCompletionContributorTest at
// 83e5e802d98d06845b220dfdd72db23da47c8be7.
func TestIntelliLuceeComponentPathCompletion(t *testing.T) {
	root := t.TempDir()

	models := filepath.Join(root, "models")
	if err := os.Mkdir(models, 0o700); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"User.cfc", "Order.cfc"} {
		path := filepath.Join(models, name)
		writeCFC(t, path, "component {}")
	}

	for _, tc := range []struct {
		name, marked string
		want         bool
	}{
		{"tag.cfm", `<cfset unrelated = 1><cfobject name="service" component="models.U|">`, true},
		{"factory.cfs", `var unrelated = 1; var user = createObject("component", "models.U|");`, true},
		{"java.cfs", `var unrelated = 1; var user = createObject("java", "models.U|");`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := startWorkflow(t, root, false)
			for _, name := range []string{"User.cfc", "Order.cfc"} {
				client.open(filepath.Join(models, name), "component {}")
			}

			prefix, suffix, _ := strings.Cut(tc.marked, "|")
			doc := client.open(filepath.Join(root, tc.name), prefix+suffix)

			var result protocol.CompletionList
			client.call(protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
				TextDocument: doc,
				Position:     protocol.Position{Character: uint32(len(prefix))},
				Context:      protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
			}, &result)

			for _, label := range []string{"models.User", "models.Order", "unrelated"} {
				want := label == "models.User" && tc.want

				got := slices.ContainsFunc(result.Items, func(item protocol.CompletionItem) bool { return item.Label == label })
				if got != want {
					t.Errorf("completion contains %q = %t, want %t", label, got, want)
				}
			}
		})
	}
}
