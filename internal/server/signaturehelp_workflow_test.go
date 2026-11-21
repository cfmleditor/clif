package server

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

func TestWorkflowSignatureHelpTracksArguments(t *testing.T) {
	root := t.TempDir()
	client := startWorkflow(t, root, false)
	client.open(filepath.Join(root, "Service.cfc"), "component { function greet(required string who, numeric count=1) {} }")

	for _, tc := range []struct {
		name, call string
		wrap       string
		want       uint32
	}{
		{"out-of-order named argument", "greet(COUNT=2|, who=\"Ada\");", "", 1},
		{"quoted comma", "greet(\"Ada,Lovelace\", 2|);", "", 1},
		{"multiline incomplete call", "greet(\n    \"Ada\",\n    2|", "", 1},
		{"nested delimiters and comments", "greet({name: 'Ada', details: [1, 2]}, /* ignored , ) */ 2|);", "", 1},
		{"call in an interpolated string", `writeOutput("#greet('Ada,Lovelace', 2|)#");`, "", 1},
		{"UTF-16 column", `greet("😀,Ada", 2|);`, "", 1},
		{"script block after markup", `greet("Ada,", 2|);`, "<p>It's ready</p><cfscript>%s</cfscript>", 1},
		{"tag expression after markup", `greet("Ada,",
  2|)`, "<p>It's ready</p><cfset result = %s>", 1},
		{"comparison is positional", "greet(count == 2|, 3);", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wrap, ext := tc.wrap, ".cfm"
			if wrap == "" {
				wrap, ext = "component {\n  function greet(required string who, numeric count=1) {}\n  function run() {\n    %s\n  }\n}", ".cfc"
			}

			source := strings.Replace(fmt.Sprintf(wrap, tc.call), "|", "", 1)
			line, char := cursorPosition(t, source, tc.call)
			doc := client.open(filepath.Join(root, tc.name+ext), source)

			var help *protocol.SignatureHelp

			client.call(protocol.MethodTextDocumentSignatureHelp, protocol.SignatureHelpParams{
				TextDocument: doc,
				Position:     protocol.Position{Line: line, Character: char},
			}, &help)

			if help == nil || len(help.Signatures) != 1 || !strings.HasPrefix(help.Signatures[0].Label, "greet(") {
				t.Fatalf("signature help = %+v, want greet signature", help)
			}

			if got, ok := help.ActiveParameter.Get(); !ok || got != tc.want {
				t.Errorf("active parameter = %v (present: %v), want %d", got, ok, tc.want)
			}
		})
	}
}
