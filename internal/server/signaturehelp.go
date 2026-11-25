package server

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/docs"
	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
)

func (s *Server) handleSignatureHelp(_ context.Context, rawParams []byte) (any, error) {
	var params protocol.SignatureHelpParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	defer s.lockDoc(params.TextDocument.URI)()

	s.log.Debug("signatureHelp: request",
		cflog.String("uri", string(params.TextDocument.URI)),
		cflog.Uint32("line", params.Position.Line),
		cflog.Uint32("char", params.Position.Character))

	content, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	line := int(params.Position.Line)
	docURI := params.TextDocument.URI
	call := s.parseResultOf(docURI).SignatureCallAt(content, line, params.Position.Character)

	if call.Name == "" {
		return nil, nil
	}

	// Try builtin functions first.
	if e, ok := docs.LookupFunction(call.Name); ok {
		idx := call.ArgumentIndex

		for i := range e.Params {
			if strings.EqualFold(call.ArgumentName, e.Params[i].Name) {
				idx = i

				break
			}
		}

		return signatureHelp(buildBuiltinSignature(e), idx), nil
	}

	// Try resolving via qualifier (e.g. service.method or getService("x").method).
	if call.Qualifier != "" {
		if def := s.resolveUserFunc(call.Qualifier, call.Name, docURI, params.Position.Line); def != nil {
			return userSignatureHelp(def, &call), nil
		}
	}

	// Prefer the current file, then the first match, without copying the
	// index bucket when many components declare the same name.
	def, _, total := s.index.LookupPreferred(call.Name, docURI)
	if total == 0 {
		return nil, nil
	}

	return userSignatureHelp(def, &call), nil
}

func userSignatureHelp(def *parser.FunctionDef, call *parser.SignatureCall) *protocol.SignatureHelp {
	idx := call.ArgumentIndex

	for i := range def.Arguments {
		if strings.EqualFold(call.ArgumentName, def.Arguments[i].Name) {
			idx = i

			break
		}
	}

	return signatureHelp(buildUserSignature(def), idx)
}

func signatureHelp(sig protocol.SignatureInformation, idx int) *protocol.SignatureHelp {
	active := protocol.NewNullable(conv.Uint32(idx))
	sig.ActiveParameter = active

	return &protocol.SignatureHelp{
		Signatures:      []protocol.SignatureInformation{sig},
		ActiveSignature: new(uint32),
		ActiveParameter: active,
	}
}

func buildUserSignature(def *parser.FunctionDef) protocol.SignatureInformation {
	doc := stubDoc(def)

	var label strings.Builder
	label.WriteString(def.Name)
	label.WriteByte('(')

	paramInfos := make([]protocol.ParameterInformation, 0, len(def.Arguments))

	for i := range def.Arguments {
		arg := &def.Arguments[i]
		if i > 0 {
			label.WriteString(", ")
		}

		paramLabel := ""
		if arg.Required {
			paramLabel += "required "
		}

		if arg.Type != "" {
			paramLabel += arg.Type + " "
		}

		paramLabel += arg.Name
		label.WriteString(paramLabel)

		info := protocol.ParameterInformation{Label: protocol.String(paramLabel)}

		if d := doc; d != nil {
			if text := d.Param(arg.Name); text != "" {
				info.Documentation = tooltip(text)
			}
		}

		paramInfos = append(paramInfos, info)
	}

	label.WriteString(")")

	sig := protocol.SignatureInformation{Label: label.String(), Parameters: paramInfos}
	if doc != nil {
		sig.Documentation = &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: doc.Markdown()}
	}

	return sig
}

func buildBuiltinSignature(e *docs.Entry) protocol.SignatureInformation {
	paramInfos := make([]protocol.ParameterInformation, 0, len(e.Params))

	var label strings.Builder
	label.WriteString(e.Name)
	label.WriteByte('(')

	for i := range e.Params {
		p := &e.Params[i]

		if i > 0 {
			label.WriteString(", ")
		}

		paramLabel := p.Name
		if p.Type != "" {
			paramLabel = p.Type + " " + p.Name
		}

		label.WriteString(paramLabel)

		doc := p.Description
		if len(p.Values) > 0 {
			doc += fmt.Sprintf(" (values: %s)", strings.Join(p.Values, ", "))
		}

		paramInfos = append(paramInfos, protocol.ParameterInformation{
			Label:         protocol.String(paramLabel),
			Documentation: tooltip(doc),
		})
	}

	label.WriteString(")")

	return protocol.SignatureInformation{
		Label:         label.String(),
		Documentation: &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: e.Doc()},
		Parameters:    paramInfos,
	}
}
