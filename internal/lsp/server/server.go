package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jwtly10/litlua"
	iLsp "github.com/jwtly10/litlua/internal/lsp"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
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
	// shadow_file = file:///Users/personal/Projects/litlua/testdata/parser/basic_valid.md.lua
	//
	// original    = file:///Users/personal/Projects/litlua/testdata/parser/basic_valid.md
	shadowMap map[string]string // [shadowUrl]SourceUrl
	parser    *litlua.Parser
	writer    *iLsp.Writer

	processor *iLsp.DocumentProcessor

	luaLS *LuaLS
}

func NewServer(parser *litlua.Parser, writer *iLsp.Writer) (*Server, error) {
	s := &Server{
		documents: make(map[string]*litlua.Document),
		shadowMap: make(map[string]string),
		parser:    parser,
		writer:    writer,

		processor: iLsp.NewDocumentProcessor(parser, writer),
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

	switch req.Method {
	case "initialize":
		slog.Info("initializing lsp server")

		var initParams lsp.InitializeParams
		if err := json.Unmarshal(*req.Params, &initParams); err != nil {
			return nil, err
		}

		s.shadowRoot = filepath.Join(os.TempDir(), "litlua")
		// At this point we know s.shadowRoot is set
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
		sr, err := s.ShadowRoot()
		if err != nil {
			return nil, err
		}
		if err := os.RemoveAll(sr); err != nil {
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

		sr, err := s.ShadowRoot()
		if err != nil {
			return nil, err
		}

		doc, shadowPath, err := s.processor.ProcessDocument(
			params.TextDocument.Text,
			fileURI.Path,
			sr,
		)
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

		// Process content changes
		if len(params.ContentChanges) > 0 {
			sr, err := s.ShadowRoot()
			if err != nil {
				return nil, err
			}
			newContent := params.ContentChanges[0].Text
			doc, shadowPath, err := s.processor.ProcessDocument(newContent, fileURI.Path, sr)
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

	case "textDocument/didClose":
		var params lsp.DidCloseTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return s.luaLS.ForwardRequest(req.Method, params)

	case "window/logMessage", "window/showMessage", "textDocument/publishDiagnostics":
		slog.Debug("received notification from lua-ls", "method", req.Method, "params", req.Params)

		return s.luaLS.ForwardRequest(req.Method, req.Params)

	case "textDocument/semanticTokens/full":
		var params lsp.SemanticHighlightingTokens
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return s.luaLS.ForwardRequest(req.Method, params)

	default:
		slog.Warn("unknown method", "method", req.Method)
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: req.Method + " not found"}
	}

}

func (s *Server) SendDiagnostics(ctx context.Context, params lsp.PublishDiagnosticsParams) error {
	return s.conn.Notify(ctx, "textDocument/publishDiagnostics", params)
}

func (s *Server) ShadowRoot() (string, error) {
	if s.shadowRoot == "" {
		return "", fmt.Errorf("shadow root not set. This should have been set during initialization")
	}

	return s.shadowRoot, nil
}

func (s *Server) getShadowToOriginalURI(shadowURI string) string {
	return s.shadowMap[shadowURI]
}
