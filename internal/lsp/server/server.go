package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	iLsp "github.com/jwtly10/litlua/internal/lsp"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
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

type Options struct {
	// Custom path to lua-language-server
	LuaLsPath string
	// Custom path to where intermediate LSP shadow files are stored
	ShadowRoot string
}

func (o *Options) Validate() error {
	if o.LuaLsPath != "" {
		if _, err := os.Stat(o.LuaLsPath); err != nil {
			return fmt.Errorf("lua-language-server path is invalid: %w", err)
		}
	}

	if o.ShadowRoot != "" {
		if _, err := os.Stat(o.ShadowRoot); err != nil {
			return fmt.Errorf("shadow root path is invalid: %w", err)
		}
	}

	return nil
}

func (o *Options) OverrideDocOpts(opts *iLsp.DocumentServiceOptions) error {
	if err := o.Validate(); err != nil {
		return fmt.Errorf("invalid server options: %w", err)
	}

	if o.ShadowRoot != "" {
		opts.ShadowRoot = o.ShadowRoot
	}

	return opts.Validate()
}

type Server struct {
	conn *jsonrpc2.Conn
	// lua lsp interface
	LuaLS *LuaLS

	// tracking for method request counts
	trackRequestCount sync.Map

	// abstraction for transpiling operations
	docService *iLsp.DocumentService

	// Mutex for the debounceTimer map
	mu            sync.Mutex
	debounceTimer map[string]*time.Timer
}

func NewServer(opts Options) (*Server, error) {
	// We load defaults
	docOpts := iLsp.DefaultDocumentServiceOptions
	if err := opts.OverrideDocOpts(&docOpts); err != nil {
		return nil, fmt.Errorf("failed to set options: %w", err)
	}

	dService, err := iLsp.NewDocumentService(docOpts)
	if err != nil {
		return nil, err
	}

	s := &Server{
		docService:    dService,
		debounceTimer: make(map[string]*time.Timer),
	}

	l, err := NewLuaLs(s, opts.LuaLsPath)
	if err != nil {
		return nil, err
	}

	s.LuaLS = l
	return s, nil
}

func (s *Server) Start() error {
	if err := s.LuaLS.Start(); err != nil {
		return fmt.Errorf("failed to start lua-ls: %w", err)
	}

	return nil
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

	switch req.Method {
	case "initialize":
		slog.Info("initializing lsp server")

		var initParams lsp.InitializeParams
		if err := json.Unmarshal(*req.Params, &initParams); err != nil {
			return nil, err
		}

		initParams.RootPath = s.docService.ShadowRoot()
		initParams.RootURI = lsp.DocumentURI("file://" + s.docService.ShadowRoot())

		result, err := s.LuaLS.ForwardRequest(req.Method, initParams)
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
		return s.LuaLS.ForwardRequest(req.Method, params)
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

		if !strings.HasSuffix(string(params.TextDocument.URI), ".litlua.md") {
			slog.Info("ignoring non-litlua file", "uri", params.TextDocument.URI)
			return nil, &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInvalidRequest,
				Message: "litlua-ls only supports .litlua.md files",
			}
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

		return s.LuaLS.ForwardRequest("textDocument/didOpen", newParams)
	case "textDocument/didChange":
		var params lsp.DidChangeTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		// I have implemented debouncing because testing in VSCode
		// showed that the didChange event is sent on every character change
		// so even though neovim does not do this, we handle it just in case
		// as it can cause the entire LSP (and editor) to hang
		s.handleDebouncedChange(params)

		return nil, nil
	case "textDocument/didSave":
		var params lsp.DidSaveTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		// TODO: Add debouncing
		slog.Info("Compiling final output on save", "uri", params.TextDocument.URI)

		shadowURI, exists := s.docService.ShadowURI(string(params.TextDocument.URI))
		if !exists {
			return nil, fmt.Errorf("no shadow file found for %s", params.TextDocument.URI)
		}

		originalURI, exists := s.docService.OriginalURI(shadowURI)
		if !exists {
			return nil, fmt.Errorf("no original file found for %s", shadowURI)
		}

		originalPath, err := s.docService.URIToPath(lsp.DocumentURI(originalURI))
		if err != nil {
			return nil, fmt.Errorf("failed to get original path: %w", err)
		}

		slog.Debug("Original path", "path", originalPath)

		content, err := os.ReadFile(originalPath)
		if err != nil {
			return nil, err
		}

		transformedPath, err := s.docService.TransformFinalDoc(string(content), originalPath)
		if err != nil {
			return nil, fmt.Errorf("failed to transform final doc: %w", err)
		}

		slog.Info("Compiled final output", "path", transformedPath)

		return nil, nil
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
		result, err := s.LuaLS.ForwardRequest(req.Method, params)
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
		return s.LuaLS.ForwardRequest(req.Method, params)

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
		return s.LuaLS.ForwardRequest(req.Method, params)

	// There are some methods we want to ignore, as they are not implemented
	// and cause overheard when proxying to the lua-language-server
	case "$/cancelRequest", "textDocument/documentHighlight", "textDocument/documentSymbol", "textDocument/foldingRange",
		"textDocument/documentColor", "textDocument/codeLens", "textDocument/codeAction", "textDocument/semanticTokens/range",
		"textDocument/semanticTokens/full", "$/setTrace":
		return nil, nil
	// Anything else is not specifically implemented through LitLua.
	// We just proxy the request to the lua-language-server and accept partial support
	// for anything undocumented as supported
	default:
		//slog.Warn("unknown method", "method", req.Method)
		return s.LuaLS.ForwardRequest(req.Method, req.Params)
	}

}

func (s *Server) handleDebouncedChange(params lsp.DidChangeTextDocumentParams) {
	uri := string(params.TextDocument.URI)

	s.mu.Lock()
	if timer, exists := s.debounceTimer[uri]; exists {
		timer.Stop()
	}

	s.debounceTimer[uri] = time.AfterFunc(200*time.Millisecond, func() {
		if len(params.ContentChanges) > 0 {
			newContent := params.ContentChanges[0].Text

			shadowURI, err := s.docService.TransformShadowDoc(newContent, params.TextDocument.URI)
			if err != nil {
				slog.Error("failed to transform shadow doc", "error", err)
				return
			}

			fsPath, err := s.docService.URIToPath(lsp.DocumentURI(shadowURI))
			if err != nil {
				slog.Error("failed to get fs path", "error", err)
				return
			}

			content, err := os.ReadFile(fsPath)
			if err != nil {
				slog.Error("failed to read shadow file", "error", err)
				return
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

			s.LuaLS.ForwardRequest("textDocument/didChange", newParams)
		}
	})
	s.mu.Unlock()
}

func (s *Server) SendDiagnostics(ctx context.Context, params lsp.PublishDiagnosticsParams) error {
	return s.conn.Notify(ctx, "textDocument/publishDiagnostics", params)
}

func (s *Server) getShadowToOriginalURI(shadowURI string) (string, bool) {
	// On macOS a temp dir with /var is symlinked to /private/var
	// which fails a lookup since they don't match what we store in the map

	path := strings.TrimPrefix(shadowURI, "file:///")
	path = strings.TrimPrefix(path, "private/")

	normalizedURI := "file:///" + path

	slog.Debug("URI normalization",
		"original", shadowURI,
		"normalized", normalizedURI)

	return s.docService.OriginalURI(normalizedURI)
}

func (s *Server) printDebugStats() {
	slog.Debug("request counts")
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
