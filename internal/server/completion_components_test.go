package server

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

func TestComponentPathCompletionVariants(t *testing.T) {
	root := t.TempDir()

	models := filepath.Join(root, "models")
	if err := os.Mkdir(models, 0o700); err != nil {
		t.Fatal(err)
	}

	writeCFC(t, filepath.Join(root, "Application.cfc"), "component {}")
	writeCFC(t, filepath.Join(models, "User.cfc"), "component {}")
	writeCFC(t, filepath.Join(models, "Order.cfc"), "component {}")

	client := startWorkflow(t, root, false)
	for _, path := range []string{
		filepath.Join(root, "Application.cfc"),
		filepath.Join(models, "User.cfc"),
		filepath.Join(models, "Order.cfc"),
	} {
		client.open(path, "component {}")
	}

	cases := []struct {
		name     string
		file     string
		marked   string
		trigger  protocol.CompletionContext
		want     []string
		forbid   []string
		wantText string
	}{
		{
			name:     "dot trigger inside createObject",
			file:     "dot.cfs",
			marked:   `var x = createObject("component", "models.|");`,
			trigger:  protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindTriggerCharacter, TriggerCharacter: strPtr(".")},
			want:     []string{"models.Order", "models.User"},
			forbid:   []string{"Application", "x"},
			wantText: `var x = createObject("component", "models.User");`,
		},
		{
			name:    "single quotes and empty prefix",
			file:    "single.cfs",
			marked:  `var x = createObject('component', '|');`,
			trigger: protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
			want:    []string{"models.Order", "models.User"},
			forbid:  []string{"Application", "x"},
		},
		{
			name:    "named createObject arguments",
			file:    "named.cfs",
			marked:  `var x = createObject(type="component", class="models.u|");`,
			trigger: protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
			want:    []string{"models.User"},
			forbid:  []string{"models.Order", "x"},
		},
		{
			name:    "named createObject java suppresses completions",
			file:    "named_java.cfs",
			marked:  `var x = createObject(type="java", class="models.U|");`,
			trigger: protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
			forbid:  []string{"models.User", "models.Order", "x"},
		},
		{
			name:    "cfset createObject",
			file:    "cfset.cfm",
			marked:  `<cfset unrelated = 1><cfset u = createObject("component", "models.U|")>`,
			trigger: protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
			want:    []string{"models.User"},
			forbid:  []string{"models.Order", "unrelated"},
		},
		{
			name:    "cfobject type component class attr",
			file:    "cfobject_class.cfm",
			marked:  `<cfobject type="component" class="models.O|" name="o">`,
			trigger: protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
			want:    []string{"models.Order"},
			forbid:  []string{"models.User"},
		},
		{
			name:    "cfobject type java class attr suppresses CFCs",
			file:    "cfobject_java.cfm",
			marked:  `<cfobject type="java" class="models.U|" name="j">`,
			trigger: protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
			forbid:  []string{"models.User", "models.Order"},
		},
		{
			name:    "cfinvoke component attr",
			file:    "cfinvoke.cfm",
			marked:  `<cfinvoke component="models.U|" method="load">`,
			trigger: protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
			want:    []string{"models.User"},
			forbid:  []string{"models.Order"},
		},
		{
			name:    "cfcomponent extends attr",
			file:    "Child.cfc",
			marked:  `<cfcomponent extends="models.U|">`,
			trigger: protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
			want:    []string{"models.User"},
			forbid:  []string{"models.Order"},
		},
		{
			name:    "hash interpolation keeps expression completions",
			file:    "interp.cfm",
			marked:  `<cfset dynPath = "models.User"><cfobject name="s" component="#dyn|">`,
			trigger: protocol.CompletionContext{TriggerKind: protocol.CompletionTriggerKindInvoked},
			want:    []string{"VARIABLES"},
			forbid:  []string{"models.User", "models.Order"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prefix, suffix, ok := strings.Cut(tc.marked, "|")
			if !ok {
				t.Fatal("missing | marker")
			}

			src := prefix + suffix
			doc := client.open(filepath.Join(root, tc.file), src)

			var result protocol.CompletionList
			client.call(protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
				TextDocument: doc,
				Position:     protocol.Position{Character: uint32(len(prefix))},
				Context:      tc.trigger,
			}, &result)

			for _, label := range tc.want {
				if !slices.ContainsFunc(result.Items, func(it protocol.CompletionItem) bool { return it.Label == label }) {
					t.Errorf("missing expected completion %q in %v", label, itemLabels(result.Items))
				}
			}

			for _, label := range tc.forbid {
				if slices.ContainsFunc(result.Items, func(it protocol.CompletionItem) bool { return it.Label == label }) {
					t.Errorf("unexpected completion %q in %v", label, itemLabels(result.Items))
				}
			}

			if tc.wantText != "" {
				idx := slices.IndexFunc(result.Items, func(it protocol.CompletionItem) bool { return it.Label == "models.User" })
				if idx < 0 {
					t.Fatalf("expected models.User in %+v", result.Items)
				}

				te, ok := result.Items[idx].TextEdit.(*protocol.TextEdit)
				if !ok || te == nil {
					t.Fatalf("expected *protocol.TextEdit for models.User, got %#v", result.Items[idx].TextEdit)
				}

				gotText := applyEdit(src, te.Range, te.NewText)
				if gotText != tc.wantText {
					t.Errorf("applied edit = %q, want %q", gotText, tc.wantText)
				}
			}
		})
	}
}

func itemLabels(items []protocol.CompletionItem) []string {
	out := make([]string, len(items))
	for i := range items {
		out[i] = items[i].Label
	}

	return out
}
