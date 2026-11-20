package server

import (
	"encoding/json/v2"
	"fmt"
	"time"

	cflog "github.com/cfmleditor/clif/internal/log"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// traceLSPRequest records the handler's outcome while the request still owns
// its parameter bytes. Record only identifying metadata from incoming params,
// since document synchronization can carry the entire editor buffer.
func (s *Server) traceLSPRequest(req *jsonrpc2.Request, result any, err error, elapsed time.Duration) {
	debug := cflog.LocalDebug(s.log)
	if debug == nil {
		return
	}

	fields := []any{
		cflog.String("method", req.Method()),
		cflog.Duration("elapsed", elapsed),
	}
	if doc := docOf(req.Params()); doc != "" {
		fields = append(fields, cflog.String("doc", doc))
	}

	if req.Method() == protocol.MethodWorkspaceExecuteCommand {
		var p struct {
			Command string `json:"command"`
		}

		if json.Unmarshal(req.Params(), &p) == nil {
			fields = append(fields, cflog.String("command", p.Command))
		}
	}

	if !req.IsCall() {
		debug("LSP notification handled", append(fields, cflog.Err(err))...)

		return
	}

	fields = append(fields, cflog.String("id", fmt.Sprintf("%q", req.ID())))
	if err != nil {
		debug("LSP response", append(fields, cflog.Err(err))...)

		return
	}

	debug("LSP response", append(fields, traceJSON("result", result))...)
}

func (s *Server) traceLSPNotification(method string, params any, err error) {
	if debug := cflog.LocalDebug(s.log); debug != nil {
		debug("LSP notification to client", cflog.String("method", method),
			traceJSON("params", params), cflog.Err(err))
	}
}

func traceLSPClientCall(debug func(string, ...any), method string, id jsonrpc2.ID, params, result any, err error) {
	fields := []any{
		cflog.String("method", method),
		cflog.String("id", fmt.Sprintf("%q", id)),
		traceJSON("params", params),
	}
	if err != nil {
		debug("LSP client call", append(fields, cflog.Err(err))...)

		return
	}

	if raw, ok := result.(*jsonrpc2.RawMessage); ok {
		result = *raw
	}

	debug("LSP client call", append(fields, traceJSON("result", result))...)
}

// traceJSON uses the connection's codec, including its handling of null and
// already encoded results. A logging error should never change the reply.
func traceJSON(key string, value any) cflog.Field {
	data, err := jsonrpc2.DefaultCodec.Marshal(value)
	if err != nil {
		return cflog.String(key+"EncodingError", err.Error())
	}

	return cflog.String(key, string(data))
}
