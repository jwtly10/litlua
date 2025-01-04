package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jwtly10/litlua"
	iLsp "github.com/jwtly10/litlua/internal/lsp"
	"github.com/jwtly10/litlua/internal/transformer"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type rwc struct {
	r io.ReadCloser
	w io.WriteCloser
}

func (rw rwc) Read(p []byte) (int, error)  { return rw.r.Read(p) }
func (rw rwc) Write(p []byte) (int, error) { return rw.w.Write(p) }
func (rw rwc) Close() error {
	rerr := rw.r.Close()
	werr := rw.w.Close()
	if rerr != nil {
		return rerr
	}
	return werr
}

type Server struct {
	conn *jsonrpc2.Conn
	// lua lsp interface
	luaLS *LuaLS
	// tracks canceled request IDs
	cancelMap sync.Map

	// tracking for method request counts
	trackRequestCount sync.Map

	// abstraction for transpiling operations
	docService *iLsp.DocumentService
}

type Options struct {
	LuaLsPath  string
	DocService iLsp.DocumentServiceOptions
}

var DefaultServerOptions = Options{
	LuaLsPath: filepath.Join(os.TempDir(), "litlua-workspace"),
	DocService: iLsp.DocumentServiceOptions{
		ShadowRoot: filepath.Join(os.TempDir(), "litlua-workspace"),
		ShadowTransformerOpts: transformer.TransformOptions{
			WriterMode:          litlua.ModeShadow,
			NoBackup:            true,
			RequirePragmaOutput: false,
		},
		FinalTransformerOpts: transformer.TransformOptions{
			WriterMode:          litlua.ModePretty,
			NoBackup:            false,
			RequirePragmaOutput: true,
		},
	},
}

func NewServer(options Options) (*Server, error) {
	dService, err := iLsp.NewDocumentService(options.DocService)
	if err != nil {
		return nil, err
	}

	s := &Server{
		docService: dService,
	}

	l, err := NewLuaLs(s)
	if err != nil {
		return nil, err
	}

	s.luaLS = l
	return s, nil
}

func (s *Server) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if s.conn == nil {
		s.conn = conn
	}
	slog.Info("received request", "method", req.Method, "id", req.ID)
	reqCount, _ := s.trackRequestCount.LoadOrStore(req.Method, 1)
	if count, ok := reqCount.(int); ok {
		s.trackRequestCount.Store(req.Method, count+1)
	}

	if _, ok := s.cancelMap.Load(req.ID.String()); ok {
		slog.Debug("request was canceled", "id", req.ID)
		s.cancelMap.Delete(req.ID.String())
		return nil, nil
	}

	switch req.Method {
	case "initialize":
		slog.Info("initializing lsp server")

		var initParams lsp.InitializeParams
		if err := json.Unmarshal(*req.Params, &initParams); err != nil {
			return nil, err
		}

		initParams.RootPath = s.docService.ShadowRoot()
		initParams.RootURI = lsp.DocumentURI("file://" + s.docService.ShadowRoot())

		result, err := s.luaLS.ForwardRequest(req.Method, initParams)
		if err != nil {
			return nil, err
		}

		response := result.(map[string]interface{})
		if caps, ok := response["capabilities"].(map[string]interface{}); ok {
			caps["textDocumentSync"] = 1 // Full sync
			caps["publishDiagnostics"] = true
		}

		return response, nil

	case "initialized":
		slog.Info("server initialized")
		var params lsp.InitializeResult
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return s.luaLS.ForwardRequest(req.Method, params)
	case "shutdown":
		slog.Info("shutting down")

		if err := s.docService.CleanupShadowFiles(); err != nil {
			slog.Error("failed to remove shadow workspace", "error", err)
		}

		s.printDebugStats()

		return nil, nil
	case "exit":
		slog.Info("exiting")

		os.Exit(0)
		return nil, nil

	// Biz logic
	case "textDocument/didOpen":
		// The file is transpiled on open, so LSP diagnostics are shown initially
		var params lsp.DidOpenTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		shadowURI, err := s.docService.TransformShadowDoc(params.TextDocument.Text, params.TextDocument.URI)
		if err != nil {
			return nil, err
		}

		fsPath, err := s.docService.URIToPath(lsp.DocumentURI(shadowURI))
		if err != nil {
			return nil, err
		}

		content, err := os.ReadFile(fsPath)
		if err != nil {
			return nil, err
		}

		newParams := lsp.DidOpenTextDocumentParams{
			TextDocument: lsp.TextDocumentItem{
				URI:        lsp.DocumentURI(shadowURI),
				Text:       string(content),
				LanguageID: "lua",
				Version:    params.TextDocument.Version,
			},
		}

		slog.Debug("forwarding didOpen to lua-ls", "params", newParams)

		return s.luaLS.ForwardRequest("textDocument/didOpen", newParams)
	case "textDocument/didChange":
		var params lsp.DidChangeTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		if len(params.ContentChanges) > 0 {
			newContent := params.ContentChanges[0].Text

			shadowURI, err := s.docService.TransformShadowDoc(newContent, params.TextDocument.URI)
			if err != nil {
				return nil, err
			}

			fsPath, err := s.docService.URIToPath(lsp.DocumentURI(shadowURI))
			if err != nil {
				return nil, err
			}

			content, err := os.ReadFile(fsPath)
			if err != nil {
				return nil, err
			}

			newParams := lsp.DidChangeTextDocumentParams{
				TextDocument: params.TextDocument,
				ContentChanges: []lsp.TextDocumentContentChangeEvent{
					{
						Text: string(content),
					},
				},
			}
			newParams.TextDocument.URI = lsp.DocumentURI(shadowURI)

			return s.luaLS.ForwardRequest("textDocument/didChange", newParams)
		}

		// No content changes
		return s.luaLS.ForwardRequest("textDocument/didChange", params)
	case "textDocument/didSave":
		var params lsp.DidSaveTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		return s.luaLS.ForwardRequest(req.Method, params)
	case "textDocument/definition":
		var params lsp.TextDocumentPositionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		shadowURI, exists := s.docService.ShadowURI(string(params.TextDocument.URI))
		if !exists {
			return nil, fmt.Errorf("no shadow file found for %s", params.TextDocument.URI)
		}

		params.TextDocument.URI = lsp.DocumentURI(shadowURI)
		result, err := s.luaLS.ForwardRequest(req.Method, params)
		if err != nil {
			return nil, err
		}

		resultBytes, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}

		var locationLinks []LocationLink
		if err := json.Unmarshal(resultBytes, &locationLinks); err == nil {
			for i := range locationLinks {
				slog.Debug("locations[i].URI", "locations[i].URI", locationLinks[i].TargetURI)
				originalURI, exists := s.getShadowToOriginalURI(string(locationLinks[i].TargetURI))
				if exists {
					locationLinks[i].TargetURI = lsp.DocumentURI(originalURI)
				} else {
					slog.Debug("unable to find original URI for shadow URI", "shadowURI", locationLinks[i].TargetURI)
				}
			}
			slog.Debug("received location link definition response", "result", result)
			return locationLinks, nil
		}

		slog.Debug("received definition response that we could not parse", "result", result)
		return nil, fmt.Errorf("unable to parse definition response")

	case "textDocument/hover":
		var params lsp.TextDocumentPositionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		shadowURI, exists := s.docService.ShadowURI(string(params.TextDocument.URI))
		if !exists {
			return nil, fmt.Errorf("no shadow file found for %s", params.TextDocument.URI)
		}

		params.TextDocument.URI = lsp.DocumentURI(shadowURI)
		return s.luaLS.ForwardRequest(req.Method, params)

	case "textDocument/completion":
		var params lsp.CompletionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		shadowURI, exists := s.docService.ShadowURI(string(params.TextDocument.URI))
		if !exists {
			return nil, fmt.Errorf("no shadow file found for %s", params.TextDocument.URI)
		}

		params.TextDocument.URI = lsp.DocumentURI(shadowURI)
		return s.luaLS.ForwardRequest(req.Method, params)
	case "$/cancelRequest":
		var params lsp.CancelParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		slog.Debug("canceling request", "id", params.ID)
		s.cancelMap.Store(params.ID.String(), struct{}{})
		return nil, nil

	// Anything else is not specifically implemented through LitLua.
	// We just proxy the request to the lua-language-server and accept partial support
	// for anything undocumented as supported
	default:
		//slog.Warn("unknown method", "method", req.Method)
		return s.luaLS.ForwardRequest(req.Method, req.Params)
	}

}

func (s *Server) SendDiagnostics(ctx context.Context, params lsp.PublishDiagnosticsParams) error {
	return s.conn.Notify(ctx, "textDocument/publishDiagnostics", params)
}

func (s *Server) getShadowToOriginalURI(shadowURI string) (string, bool) {
	return s.docService.OriginalURI(shadowURI)
}

func (s *Server) printDebugStats() {
	s.trackRequestCount.Range(func(key, value interface{}) bool {
		msg := fmt.Sprintf("Method: %-30s Count: %d", key.(string), value.(int))
		slog.Debug(msg)
		return true
	})
}

// LocationLink is an implementation of the LocationLink LSP type
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#locationLink
//
// https://github.com/sourcegraph/go-lsp does not support the LocationLink Type
// so we have implemented it here for lua-language-server
type LocationLink struct {
	// Span of the origin of this link.
	// Used as the underlined span for mouse interaction.
	// Optional - defaults to the word range at the mouse position.
	OriginSelectionRange *lsp.Range `json:"originSelectionRange,omitempty"`

	// The target resource identifier of this link.
	TargetURI lsp.DocumentURI `json:"targetUri"`

	// The full target range including surrounding content like comments
	TargetRange lsp.Range `json:"targetRange"`

	// The precise range that should be selected when following the link
	// Must be contained within TargetRange
	TargetSelectionRange lsp.Range `json:"targetSelectionRange"`
}
