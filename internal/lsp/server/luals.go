package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
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

	Path string

	server lspServer
}

func NewLuaLs(server lspServer, luaLSPath string) (*LuaLS, error) {
	luaPath, err := findLuaLS(luaLSPath)
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

	rw := NewRWC(stdout, stdin)
	stream := jsonrpc2.NewBufferedStream(rw, jsonrpc2.VSCodeObjectCodec{})

	l := &LuaLS{
		cmd:    cmd,
		server: server,
		Path:   luaPath,
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

	return l, nil
}

func (l *LuaLS) Start() error {
	if err := l.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start lua-language-server: %w", err)
	}

	go func() {
		err := l.cmd.Wait()
		if err != nil {
			slog.Error("lua-language-server exited", "error", err)
		}
	}()

	slog.Info("lua-language-server started", "path", l.Path)

	return nil
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

// findLuaLS attempts to find the lua-language-server binary on the system
//
// It will try to find the binary based on the provided path, if not it will
// attempt common paths where the binary might be located.
func findLuaLS(tryFirst string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	commonPaths := []string{
		filepath.Join(homeDir, ".local/share/nvim/mason/bin/lua-language-server"), // Default mason path
		"/opt/homebrew/bin/lua-language-server",                                   // Homebrew (arm64)
		"/usr/local/bin/lua-language-server",                                      // Homebrew (x86_64)
		"/usr/bin/lua-language-server",                                            // Linux
	}

	if tryFirst != "" {
		commonPaths = append([]string{tryFirst}, commonPaths...)
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			slog.Info("valid path, will try to execute lua-language-server", "path", path)
			return path, nil
		} else {
			if path == tryFirst {
				slog.Error("custom path failed stat. Will try common locations.", "path", path, "error", err)
			}
		}
	}

	return exec.LookPath("lua-language-server")
}
