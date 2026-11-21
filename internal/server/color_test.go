package server

import (
	"math"
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
)

func TestWorkflowDocumentColorAndPresentation(t *testing.T) {
	root := t.TempDir()
	client := startWorkflow(t, root, false)

	if enabled, ok := client.capabilities.ColorProvider.(protocol.Boolean); !ok || !bool(enabled) {
		t.Fatalf("ColorProvider capability = %#v, want true", client.capabilities.ColorProvider)
	}

	t.Run("detects CSS color literals and ignores selectors, comments, and CFML expressions", func(t *testing.T) {
		src := `<style>
	#abc { color: #ff0000; background: #0f8; }
	/* #112233 should be ignored */
	<!--- #445566 should be ignored --->
	.a { border-color: rgba(0, 128, 255, 0.5); fill: hsl(120, 100%, 50%); }
	.b { color: rgb(255 0 128 / 50%); }
</style>
<div style="color: #00ff0080;"></div>
<cfoutput>
<style>
	.out { color: ##ff0000; background: #abc#; }
</style>
<p style="color: ##00f;"></p>
</cfoutput>
`
		doc := client.open(filepath.Join(root, "styles.cfm"), src)

		var colors []protocol.ColorInformation
		client.call(protocol.MethodTextDocumentDocumentColor, protocol.DocumentColorParams{TextDocument: doc}, &colors)

		want := []struct {
			snippet string
			color   protocol.Color
		}{
			{`color: |#ff0000|;`, protocol.Color{Red: 1, Green: 0, Blue: 0, Alpha: 1}},
			{`background: |#0f8|;`, protocol.Color{Red: 0, Green: 1, Blue: 0x88 / 255.0, Alpha: 1}},
			{`border-color: |rgba(0, 128, 255, 0.5)|;`, protocol.Color{Red: 0, Green: 128.0 / 255, Blue: 1, Alpha: 0.5}},
			{`fill: |hsl(120, 100%, 50%)|;`, protocol.Color{Red: 0, Green: 1, Blue: 0, Alpha: 1}},
			{`color: |rgb(255 0 128 / 50%)|;`, protocol.Color{Red: 1, Green: 0, Blue: 128.0 / 255, Alpha: 0.5}},
			{`style="color: |#00ff0080|;"`, protocol.Color{Red: 0, Green: 1, Blue: 0, Alpha: 0x80 / 255.0}},
			{`.out { color: |##ff0000|;`, protocol.Color{Red: 1, Green: 0, Blue: 0, Alpha: 1}},
			{`style="color: |##00f|;"`, protocol.Color{Red: 0, Green: 0, Blue: 1, Alpha: 1}},
		}

		if len(colors) != len(want) {
			t.Fatalf("colors (%d) = %+v, want %d", len(colors), colors, len(want))
		}

		for i, w := range want {
			wantRange := conformanceDeclarationRange(t, src, w.snippet)
			if colors[i].Range != wantRange {
				t.Errorf("color[%d] range = %+v, want %+v", i, colors[i].Range, wantRange)
			}

			assertColorNear(t, i, colors[i].Color, w.color)
		}
	})

	t.Run("picker writes doubled hash inside cfoutput and single hash outside", func(t *testing.T) {
		src := `<style>.plain { color: #0000ff; }</style>
<cfoutput><style>.escaped { color: ##0000ff; }</style></cfoutput>
<cffunction name="f" output="true"><div style="color: ##0000ff;"></div></cffunction>
`
		doc := client.open(filepath.Join(root, "picker.cfm"), src)
		red := protocol.Color{Red: 1, Green: 0, Blue: 0, Alpha: 1}
		translucentRed := protocol.Color{Red: 1, Green: 0, Blue: 0, Alpha: 0x80 / 255.0}

		for _, tc := range []struct {
			name    string
			snippet string
			color   protocol.Color
			wantHex string
		}{
			{"outside cfoutput opaque", `.plain { color: |#0000ff|; }`, red, "#ff0000"},
			{"inside cfoutput opaque", `.escaped { color: |##0000ff|; }`, red, "##ff0000"},
			{"inside output=true function opaque", `style="color: |##0000ff|;"`, red, "##ff0000"},
			{"outside cfoutput translucent", `.plain { color: |#0000ff|; }`, translucentRed, "#ff000080"},
			{"inside cfoutput translucent", `.escaped { color: |##0000ff|; }`, translucentRed, "##ff000080"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				rng := conformanceDeclarationRange(t, src, tc.snippet)

				var presentations []protocol.ColorPresentation
				client.call(protocol.MethodTextDocumentColorPresentation, protocol.ColorPresentationParams{
					TextDocument: doc,
					Color:        tc.color,
					Range:        rng,
				}, &presentations)

				if len(presentations) == 0 {
					t.Fatal("expected color presentations, got none")
				}

				got := presentations[0]
				if got.Label != tc.wantHex || got.TextEdit.NewText != tc.wantHex || got.TextEdit.Range != rng {
					t.Errorf("first presentation = %+v, want label/newText %q at %+v", got, tc.wantHex, rng)
				}
			})
		}
	})

	t.Run("UTF-16 column accuracy before color literal", func(t *testing.T) {
		src := `<div title="😀" style="color: #ff0000;"></div>`
		doc := client.open(filepath.Join(root, "utf16.cfm"), src)

		var colors []protocol.ColorInformation
		client.call(protocol.MethodTextDocumentDocumentColor, protocol.DocumentColorParams{TextDocument: doc}, &colors)

		if len(colors) != 1 {
			t.Fatalf("colors = %+v, want 1", colors)
		}

		wantRange := conformanceDeclarationRange(t, src, `style="color: |#ff0000|;"`)
		if colors[0].Range != wantRange {
			t.Errorf("range = %+v, want %+v", colors[0].Range, wantRange)
		}
	})
}

func assertColorNear(t *testing.T, idx int, got, want protocol.Color) {
	t.Helper()

	const eps = 1e-6
	if math.Abs(got.Red-want.Red) > eps || math.Abs(got.Green-want.Green) > eps ||
		math.Abs(got.Blue-want.Blue) > eps || math.Abs(got.Alpha-want.Alpha) > eps {
		t.Errorf("color[%d] = %+v, want %+v", idx, got, want)
	}
}
