package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func linkedRangesAt(t *testing.T, s *Server, docURI uri.URI, line, char uint32) *protocol.LinkedEditingRanges {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentLinkedEditingRange, protocol.LinkedEditingRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     protocol.Position{Line: line, Character: char},
	})

	got, err := s.handleLinkedEditingRange(context.Background(), raw)
	if err != nil {
		t.Fatalf("handleLinkedEditingRange error: %v", err)
	}

	res, _ := got.(*protocol.LinkedEditingRanges)

	return res
}

func TestLinkedEditingRangeAdvertised(t *testing.T) {
	s := newTestServer()
	if p, _ := s.capabilities().LinkedEditingRangeProvider.(protocol.Boolean); !bool(p) {
		t.Error("linkedEditingRangeProvider is not advertised")
	}
}

func TestLinkedEditingRangeCFMLAndHTMLTags(t *testing.T) {
	src := strings.Join([]string{
		`<cfoutput>`,
		`	<div class="box">`,
		`		<cfloop from="1" to="3" index="i">`,
		`			<cfif i EQ 1>`,
		`				<cfset x = i>`,
		`			</cfif>`,
		`		</cfloop>`,
		`	</div>`,
		`</cfoutput>`,
	}, "\n")

	s := newTestServer()
	docURI := uri.File("/workspace/view.cfm")
	openDoc(t, s, docURI, src)

	// Cursor on opening `cfoutput` (line 0, col 3).
	res := linkedRangesAt(t, s, docURI, 0, 3)
	if res == nil || len(res.Ranges) != 2 {
		t.Fatalf("cfoutput opening = %+v, want 2 ranges", res)
	}

	if res.Ranges[0].Start.Line != 0 || res.Ranges[0].Start.Character != 1 || res.Ranges[0].End.Character != 9 {
		t.Errorf("open range = %+v, want line 0 [1,9]", res.Ranges[0])
	}

	if res.Ranges[1].Start.Line != 8 || res.Ranges[1].Start.Character != 2 || res.Ranges[1].End.Character != 10 {
		t.Errorf("close range = %+v, want line 8 [2,10]", res.Ranges[1])
	}

	// Cursor on closing `</div>` (line 7, col 4).
	divRes := linkedRangesAt(t, s, docURI, 7, 4)
	if divRes == nil || len(divRes.Ranges) != 2 || divRes.Ranges[0].Start.Line != 1 || divRes.Ranges[1].Start.Line != 7 {
		t.Fatalf("div closing = %+v, want lines 1 and 7", divRes)
	}

	// Cursor on attribute (`class="box"`) must return nil.
	if attrRes := linkedRangesAt(t, s, docURI, 1, 8); attrRes != nil {
		t.Fatalf("expected nil on attribute, got %+v", attrRes)
	}

	// Cursor on void tag (`<cfset>`) must return nil.
	if voidRes := linkedRangesAt(t, s, docURI, 4, 6); voidRes != nil {
		t.Fatalf("expected nil on void tag, got %+v", voidRes)
	}
}

func TestLinkedEditingRangeLinearOnMismatchedTags(t *testing.T) {
	var b strings.Builder
	for range 3000 {
		b.WriteString("<div>")
	}

	for range 3000 {
		b.WriteString("</span>")
	}

	content := b.String()

	start := time.Now()
	_, _, found := findLinkedTagPair(content, 2)
	dur := time.Since(start)

	if found {
		t.Fatal("expected no match for mismatched tags")
	}

	if dur > 200*time.Millisecond {
		t.Fatalf("findLinkedTagPair took %v on 6000 mismatched tags; expected O(N)", dur)
	}
}
