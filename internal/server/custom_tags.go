package server

import (
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// customTagStage answers go-to-definition on a custom tag call (see
// parser.CustomTagCalls): the template the name under the cursor runs, as
// resolve.CustomTagTemplate finds it. ok is false when the cursor is on no
// call, for a tag with a configured prefix that runs nothing, which may be a
// built-in tag the ordinary lookup knows, and for a cfscript call that runs
// nothing, which may be a function named cf_x. Any other tag that resolves to
// nothing is answered with nothing: the word under the cursor is a tag name,
// and a function of the same name is not its definition.
func (s *Server) customTagStage(content string, line, char int, docURI uri.URI) (v any, ok bool) {
	r := s.getResolver()

	call, found := parser.FindCustomTagCall(s.customTagCalls(docURI, content, r.CustomTagPrefixes()), content, line, char)
	if !found {
		return nil, false
	}

	target := r.CustomTagTemplate(&call, cfpath.FromURI(string(docURI)))
	if target == "" {
		return nil, call.Kind != parser.CustomTagPrefixed && !call.Script
	}

	return protocol.Location{URI: cfpath.ToURI(target)}, true
}

// customTagLinks are document links from each custom tag call's name to the
// template it runs. A cfmodule template is linked already, as every
// template= attribute is (parser.ExtractLinks), and is left out.
func (s *Server) customTagLinks(content string, docURI uri.URI) []protocol.DocumentLink {
	r := s.getResolver()
	from := cfpath.FromURI(string(docURI))
	cols := newColMapper(content)

	var links []protocol.DocumentLink

	// A page calls the same tag many times, and each lookup checks the disk:
	// one per tag a request.
	targets := map[tagKey]string{}

	calls := s.customTagCalls(docURI, content, r.CustomTagPrefixes())
	for i := range calls {
		call := &calls[i]
		if call.Kind == parser.CustomTagModuleTemplate {
			continue
		}

		k := tagKey{kind: call.Kind, name: strings.ToLower(call.Name), prefix: call.Prefix, taglib: call.Taglib, dynamic: call.Dynamic}

		target, seen := targets[k]
		if !seen {
			target = r.CustomTagTemplate(call, from)
			targets[k] = target
		}

		if target == "" {
			continue
		}

		targetURI := cfpath.ToURI(target)
		tip := target
		line := conv.Uint32(call.Line)
		lineStart := lineStartOf(content, call.Start)

		links = append(links, protocol.DocumentLink{
			Range: protocol.Range{
				Start: protocol.Position{Line: line, Character: cols.col(line, conv.Uint32(call.Start-lineStart))},
				End:   protocol.Position{Line: line, Character: cols.col(line, conv.Uint32(call.End-lineStart))},
			},
			Target:  &targetURI,
			Tooltip: &tip,
		})
	}

	return links
}

// tagKey is what a custom tag call's template depends on, within one
// document.
type tagKey struct {
	kind                 parser.CustomTagKind
	name, prefix, taglib string
	dynamic              bool
}

// customTagCalls is the custom tag calls in the open document: its cached
// parse keeps them, read from the regions it already split, so neither
// request walks the file again. A document with no parse yet, or one whose
// parse is of other text, is read from content.
func (s *Server) customTagCalls(docURI uri.URI, content string, prefixes []string) []parser.CustomTagCall {
	s.mu.RLock()
	pr := s.parseResults[docURI]
	s.mu.RUnlock()

	if pr != nil && pr.Content == content {
		return pr.CustomTagCalls(prefixes)
	}

	return parser.CustomTagCalls(content, prefixes)
}

// lineStartOf is the offset of the start of the line holding off.
func lineStartOf(content string, off int) int {
	for i := off - 1; i >= 0; i-- {
		if content[i] == '\n' {
			return i + 1
		}
	}

	return 0
}
