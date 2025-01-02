package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
	"github.com/jwtly10/litlua"
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

type Server struct {
	conn *jsonrpc2.Conn

	documents map[string]*litlua.Document

	// shadowRoot is the root directory for shadow files
	// e.g. /tmp/litlua
	// this is to hide the intermediate compiled lua files from the user
	shadowRoot string

	//shadowMap holds state of the mapping of the shadow file to the original markdown file
	//
	// e.g.
	// shadow_file = file:///var/folders/8r/rkbmn5qd68jfvl9t6sn0szym0000gn/T/litlua-94e0a4e1-ae7d-4970-b94d-3b537c37d103/lsp_example.md.lua
	//
	// original    = file:///Users/personal/Projects/litlua/testdata/parser/basic_valid.md
	shadowMap map[string]string // [shadowUrl]SourceUrl
	parser    *litlua.Parser
	writer    *iLsp.Writer

	processor *iLsp.DocumentProcessor

	luaLS *LuaLS
}

type Options struct {
	LuaLsPath string
	ShadowDir string
}

func NewServer(parser *litlua.Parser, writer *iLsp.Writer, options Options) (*Server, error) {
	s := &Server{
		documents: make(map[string]*litlua.Document),
		shadowMap: make(map[string]string),
		parser:    parser,
		writer:    writer,

		processor: iLsp.NewDocumentProcessor(parser, writer),
	}

	tmpDir := filepath.Join(os.TempDir(), "litlua-"+uuid.New().String())
	s.shadowRoot = tmpDir

	// Just make sure we clean up the shadow files
	runtime.SetFinalizer(s, func(s *Server) {
		if err := os.RemoveAll(s.shadowRoot); err != nil {
			slog.Error("failed to cleanup shadow files", "error", err)
		}
	})

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

	switch req.Method {
	case "initialize":
		slog.Info("initializing lsp server")

		var initParams lsp.InitializeParams
		if err := json.Unmarshal(*req.Params, &initParams); err != nil {
			return nil, err
		}

		initParams.RootPath = s.shadowRoot
		initParams.RootURI = lsp.DocumentURI("file://" + s.shadowRoot)

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
		if err := os.RemoveAll(s.shadowRoot); err != nil {
			slog.Error("failed to remove shadow workspace", "error", err)
		}
		return nil, nil
	case "exit":
		slog.Info("exiting")
		// here we can handle cleaning up of the tmp files
		os.Exit(0)
		return nil, nil

	// Biz logic
	case "textDocument/didOpen":
		// The file is transpiled on open, so LSP diagnostics are shown initially
		var params lsp.DidOpenTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		// fileURI wraps the URI of the file the LSP req are triggered by in an url.URL object
		// .String() gives us the full URI as required by the LSP spec
		// .Path gives us access to the raw 'absolute' path of the URI without file://
		// so we can easily do file system operations
		fileURI, err := url.Parse(string(params.TextDocument.URI))
		if err != nil {
			return nil, err
		}

		doc, shadowPath, err := s.processor.ProcessDocument(params.TextDocument.Text, fileURI.Path, s.shadowRoot)
		if err != nil {
			return nil, err
		}

		// TODO: is this the best way to track the doc? Or should we just compile when needed?
		s.documents[fileURI.String()] = doc

		shadowURI := "file://" + shadowPath
		s.shadowMap[shadowURI] = fileURI.String()

		slog.Debug("transpiled document to shadow file", "shadow_path", shadowPath, "shadow_uri", shadowURI)

		content, err := os.ReadFile(shadowPath)
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

		fileURI, err := url.Parse(string(params.TextDocument.URI))
		if err != nil {
			return nil, err
		}

		if len(params.ContentChanges) > 0 {
			newContent := params.ContentChanges[0].Text
			doc, shadowPath, err := s.processor.ProcessDocument(newContent, fileURI.Path, s.shadowRoot)
			if err != nil {
				return nil, err
			}
			s.documents[fileURI.String()] = doc

			content, err := os.ReadFile(shadowPath)
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
			newParams.TextDocument.URI = lsp.DocumentURI("file://" + shadowPath)

			return s.luaLS.ForwardRequest("textDocument/didChange", newParams)
		}

		// No content changes
		return s.luaLS.ForwardRequest("textDocument/didChange", params)

	case "textDocument/definition":
		var params lsp.TextDocumentPositionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		fileURI, err := url.Parse(string(params.TextDocument.URI))
		if err != nil {
			return nil, err
		}

		shadowURI := "file://" + iLsp.GetRawShadowPathFromMd(fileURI.Path, s.shadowRoot)
		slog.Debug(shadowURI)
		if _, exists := s.shadowMap[shadowURI]; !exists {
			slog.Debug("text document URI", "path", string(params.TextDocument.URI))
			slog.Debug("shadow uri", "path", shadowURI)
			slog.Debug("state", "shadowMap", s.shadowMap)
			return nil, fmt.Errorf("no shadow file found for %s (%s)", params.TextDocument.URI, shadowURI)
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
				if originalURI := s.getShadowToOriginalURI(string(locationLinks[i].TargetURI)); originalURI != "" {
					slog.Debug("found original URI", "original_uri", originalURI)
					locationLinks[i].TargetURI = lsp.DocumentURI(originalURI)
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

		fileURI, err := url.Parse(string(params.TextDocument.URI))
		if err != nil {
			return nil, err
		}

		shadowURI := "file://" + iLsp.GetRawShadowPathFromMd(fileURI.Path, s.shadowRoot)
		if _, exists := s.shadowMap[shadowURI]; !exists {
			return nil, fmt.Errorf("no shadow file found for %s", params.TextDocument.URI)
		}

		params.TextDocument.URI = lsp.DocumentURI(shadowURI)
		return s.luaLS.ForwardRequest(req.Method, params)

	case "textDocument/completion":
		var params lsp.CompletionParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		fileURI, err := url.Parse(string(params.TextDocument.URI))
		if err != nil {
			return nil, err
		}

		shadowURI := "file://" + iLsp.GetRawShadowPathFromMd(fileURI.Path, s.shadowRoot)
		if _, exists := s.shadowMap[shadowURI]; !exists {
			return nil, fmt.Errorf("no shadow file found for %s", params.TextDocument.URI)
		}

		params.TextDocument.URI = lsp.DocumentURI(shadowURI)
		return s.luaLS.ForwardRequest(req.Method, params)

	// These are the LSP methods that don't require markdown-specific handling
	// or we want to not throw errors when not implemented
	case "window/logMessage", "window/showMessage", "textDocument/publishDiagnostics",
		"textDocument/didClose", "textDocument/semanticTokens/full", "textDocument/semanticTokens/range",
		"textDocument/documentColor", "textDocument/documentSymbol", "textDocument/codeLens",
		"textDocument/foldingRange", "textDocument/codeAction", "textDocument/documentHighlight", "completionItem/resolve",
		"textDocument/signatureHelp":
		return s.luaLS.ForwardRequest(req.Method, req.Params)

	default:
		slog.Warn("unknown method", "method", req.Method)
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: req.Method + " not found"}
	}

}

func (s *Server) SendDiagnostics(ctx context.Context, params lsp.PublishDiagnosticsParams) error {
	return s.conn.Notify(ctx, "textDocument/publishDiagnostics", params)
}

func (s *Server) getShadowToOriginalURI(shadowURI string) string {
	return s.shadowMap[shadowURI]
}

// https://github.com/sourcegraph/go-lsp does not support the LocationLink Type
// so we have implemented it here for lua-language-server
//
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#locationLink
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
