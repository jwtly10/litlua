package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type lspServer interface {
	SendDiagnostics(ctx context.Context, params lsp.PublishDiagnosticsParams) error
}

type LuaLS struct {
	conn *jsonrpc2.Conn
	cmd  *exec.Cmd

	server lspServer
}

func NewLuaLs(server lspServer) (*LuaLS, error) {
	luaPath, err := findLuaLS()
	if err != nil {
		return nil, fmt.Errorf("lua-language-server not found: %w", err)
	}
	cmd := exec.Command(luaPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	rw := rwc{r: stdout, w: stdin}
	stream := jsonrpc2.NewBufferedStream(rw, jsonrpc2.VSCodeObjectCodec{})

	l := &LuaLS{
		cmd:    cmd,
		server: server,
	}

	l.conn = jsonrpc2.NewConn(
		context.Background(),
		stream,
		jsonrpc2.HandlerWithError(l.HandleResponse),
		jsonrpc2.OnRecv(func(req *jsonrpc2.Request, _ *jsonrpc2.Response) {
			if req != nil {
				// Additional debugging if for some reason we are not even handling the request
				//slog.Debug("raw notification from lua-ls", "method", req.Method)
			}
		}),
	)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start lua-language-server: %w", err)
	}

	go func() {
		err := cmd.Wait()
		if err != nil {
			slog.Error("lua-language-server exited", "error", err)
		}
	}()

	slog.Debug("lua-language-server started", "path", luaPath)

	return l, nil
}

// HandleResponse handles responses from the lua-language-server
// and forwards them to the proxy
func (l *LuaLS) HandleResponse(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
	slog.Debug("received notification from lua-ls", "method", req.Method)
	switch req.Method {
	case "textDocument/publishDiagnostics":
		var params lsp.PublishDiagnosticsParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		// Get the markdown file this diagnostic is for
		originalURI, exists := l.server.(*Server).getShadowToOriginalURI(string(params.URI))
		if !exists {
			return nil, fmt.Errorf("no mapping for shadow URI: %s", params.URI)
		}

		slog.Debug("forwarding diagnostics",
			"shadow_uri", params.URI,
			"original_uri", originalURI,
			"diagnostic_count", len(params.Diagnostics))

		params.URI = lsp.DocumentURI(originalURI)
		return nil, l.server.SendDiagnostics(ctx, params)
	}
	return nil, nil
}

// ForwardRequest forwards a request from proxy to the lua-language-server
func (l *LuaLS) ForwardRequest(method string, params interface{}) (interface{}, error) {
	var result interface{}
	slog.Info("sending request to lua-ls", "method", method)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := l.conn.Call(ctx, method, params, &result)
	return result, err
}

// findLuaLS attempts to find the lua-language-server binary
// based on common installation paths
func findLuaLS() (string, error) {
	commonPaths := []string{
		"/opt/homebrew/bin/lua-language-server",
		"/usr/local/bin/lua-language-server",
		"/usr/bin/lua-language-server",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return exec.LookPath("lua-language-server")
}
