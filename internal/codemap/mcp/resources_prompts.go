package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

type resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type promptArg struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type promptDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Arguments   []promptArg `json:"arguments,omitempty"`
}

func (s *Server) resources() []resource {
	var out []resource

	if s.ConfigSummary != nil {
		out = append(out, resource{
			URI:         "clif://config",
			Name:        "Workspace configuration",
			Description: "Resolved .clif.json workspace settings (mappings, frameworks, resolvers, routes).",
			MimeType:    "application/json",
		})
	}

	if s.Store != nil {
		out = append(out,
			resource{
				URI:         "clif://stats",
				Name:        "Code map statistics",
				Description: "Node/edge counts, call-site resolution ratio, and --watch sync status.",
				MimeType:    "application/json",
			},
			resource{
				URI:         "clif://islands",
				Name:        "Code map islands",
				Description: "Top connected components and detached subsystems in the code map.",
				MimeType:    "application/json",
			},
		)
	}

	if out == nil {
		return []resource{}
	}

	return out
}

func (s *Server) readResource(raw json.RawMessage) (any, *rpcError) {
	var p struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpcError{codeInvalidParams, "bad resources/read params: " + err.Error()}
	}

	uriStr := strings.TrimSpace(p.URI)
	if uriStr == "" {
		return nil, &rpcError{codeInvalidParams, "missing required argument: uri"}
	}

	var (
		payload any
		err     error
	)

	switch uriStr {
	case "clif://config":
		if s.ConfigSummary == nil {
			return nil, &rpcError{codeInvalidParams, "resource clif://config is not available on this server"}
		}

		payload, err = s.ConfigSummary()

	case "clif://stats":
		if s.Store == nil {
			return nil, &rpcError{codeInvalidParams, "resource clif://stats needs a code map (--db)"}
		}

		payload, err = s.getStats(nil)

	case "clif://islands":
		if s.Store == nil {
			return nil, &rpcError{codeInvalidParams, "resource clif://islands needs a code map (--db)"}
		}

		found, ierr := s.Store.Islands(false, 50)
		payload, err = wrap("islands", found), ierr

	default:
		return nil, &rpcError{codeInvalidParams, fmt.Sprintf("unknown resource URI %q", uriStr)}
	}

	if err != nil {
		return nil, &rpcError{codeInternal, err.Error()}
	}

	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, &rpcError{codeInternal, "encoding resource: " + err.Error()}
	}

	return map[string]any{
		"contents": []any{
			map[string]any{
				"uri":      uriStr,
				"mimeType": "application/json",
				"text":     string(encoded),
			},
		},
	}, nil
}

var builtinPrompts = []promptDef{
	{
		Name:        "audit-unresolved",
		Description: "Audit unresolved CFML calls in a directory or file and trace root causes with explain_call.",
		Arguments: []promptArg{
			{Name: "path", Description: "File or directory to audit (defaults to workspace root '.')."},
		},
	},
	{
		Name:        "trace-impact",
		Description: "Analyze the blast radius of changing a CFML symbol before editing it.",
		Arguments: []promptArg{
			{Name: "symbol", Description: "Function name, component dot-path, or code-map node id.", Required: true},
		},
	},
	{
		Name:        "review-file",
		Description: "Run a read-only structural, formatting, and resolution review of a CFML file.",
		Arguments: []promptArg{
			{Name: "file", Description: "Path to the CFML file to review.", Required: true},
		},
	},
}

func (*Server) prompts() []promptDef {
	return builtinPrompts
}

func (*Server) getPrompt(raw json.RawMessage) (any, *rpcError) {
	var p struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	}

	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpcError{codeInvalidParams, "bad prompts/get params: " + err.Error()}
	}

	for name, value := range p.Arguments {
		if err := checkPromptArgument(name, value); err != nil {
			return nil, &rpcError{codeInvalidParams, err.Error()}
		}
	}

	switch p.Name {
	case "audit-unresolved":
		target := strings.TrimSpace(p.Arguments["path"])
		if target == "" {
			target = "."
		}

		msg := fmt.Sprintf(
			"Audit unresolved CFML calls under %q:\n"+
				"1. Call find_unresolved_calls with paths=[%q].\n"+
				"2. Group findings by root cause (missing base component, untyped receiver, missing method).\n"+
				"3. For the top 3-5 representative sites, call explain_call to confirm why resolution failed and suggest minimal .clif.json or code fixes.",
			target, target,
		)

		return promptResponse("Audit unresolved CFML calls", msg), nil

	case "trace-impact":
		sym := strings.TrimSpace(p.Arguments["symbol"])
		if sym == "" {
			return nil, &rpcError{codeInvalidParams, "missing required prompt argument: symbol"}
		}

		msg := fmt.Sprintf(
			"Analyze the impact of changing %q:\n"+
				"1. Call explore with query=%q to inspect its definition, direct callers/callees, and initial blast radius.\n"+
				"2. If a code map is loaded, call impact_analysis on the symbol's node id (and get_stats to check resolution coverage).\n"+
				"3. Otherwise call find_references and get_type_hierarchy.\n"+
				"4. Summarize affected entry points, files, and any dynamic/unresolved risk.",
			sym, sym,
		)

		return promptResponse("Trace symbol blast radius", msg), nil

	case "review-file":
		file := strings.TrimSpace(p.Arguments["file"])
		if file == "" {
			return nil, &rpcError{codeInvalidParams, "missing required prompt argument: file"}
		}

		msg := fmt.Sprintf(
			"Review the CFML file %q without modifying it:\n"+
				"1. Call get_outline on %q to inspect its structure and signatures.\n"+
				"2. Call check_structure, check_format, and find_unresolved_calls on [%q].\n"+
				"3. Report structural errors, unresolved calls, formatting diffs, and any suspicious patterns.",
			file, file, file,
		)

		return promptResponse("Review CFML file", msg), nil

	default:
		return nil, &rpcError{codeInvalidParams, fmt.Sprintf("unknown prompt %q", p.Name)}
	}
}

func promptResponse(description, text string) map[string]any {
	return map[string]any{
		"description": description,
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": map[string]any{
					"type": "text",
					"text": text,
				},
			},
		},
	}
}

// maxPromptArgument bounds a prompt argument: a path, symbol or file name is
// short, and the text is placed in a message the model reads as instructions.
const maxPromptArgument = 512

// checkPromptArgument refuses an argument that could not be a path or a
// symbol: one that is long, or holds a control character such as a newline,
// which would let the value start lines of its own in the prompt.
func checkPromptArgument(name, value string) error {
	if len(value) > maxPromptArgument {
		return fmt.Errorf("prompt argument %q is longer than %d bytes", name, maxPromptArgument)
	}

	for _, r := range value {
		if r < 0x20 || r == 0x7f || r == 0x85 || r == 0x2028 || r == 0x2029 {
			return fmt.Errorf("prompt argument %q holds a control character", name)
		}
	}

	return nil
}
