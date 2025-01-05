package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jwtly10/litlua"
	"github.com/jwtly10/litlua/internal/lsp/server"
	"github.com/sourcegraph/jsonrpc2"
)

type stdRWC struct{}

func (stdRWC) Read(p []byte) (int, error)  { return os.Stdin.Read(p) }
func (stdRWC) Write(p []byte) (int, error) { return os.Stdout.Write(p) }
func (stdRWC) Close() error                { return nil }

const usage = `LitLua Language Server

The LitLua Language Server provides LSP features for LitLua files, including:
  - Code completion
  - Go to definition
  - Hover information
  - Diagnostics

Usage:
  litlua-ls [flags]

Flags:
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Start the server with default settings\n")
		fmt.Fprintf(os.Stderr, "  $ litlua-ls\n\n")
		fmt.Fprintf(os.Stderr, "  # Start with custom lua-language-server path\n")
		fmt.Fprintf(os.Stderr, "  $ litlua-ls -luals=/usr/local/bin/lua-language-server\n\n")
		fmt.Fprintf(os.Stderr, "  # Enable debug logging\n")
		fmt.Fprintf(os.Stderr, "  $ litlua-ls -debug\n")
	}

	var (
		debug      = flag.Bool("debug", false, "Enable debug logging")
		lualsPath  = flag.String("luals", "", "Custom path to lua-language-server")
		shadowRoot = flag.String("shadow-root", "", "Custom path to shadow root directory (for LSP intermediate files)")
		version    = flag.Bool("version", false, "Print version information")
	)

	flag.Parse()

	if *version {
		fmt.Printf("litlua-ls version %s\n", litlua.VERSION)
		os.Exit(0)
	}

	flag.Parse()

	var handler slog.Handler
	if *debug {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("starting litlua-ls with opts", "version", litlua.VERSION, "debug", *debug, "custom-luals", *lualsPath, "custom-shadow-root", *shadowRoot)

	ctx := context.Background()

	opts := server.Options{
		LuaLsPath:  *lualsPath,  // Will use default if empty
		ShadowRoot: *shadowRoot, // Will use default if empty
	}

	s, err := server.NewServer(opts)
	if err != nil {
		slog.Error("failed to create lsp server", "error", err)
		return
	}

	if err := s.Start(); err != nil {
		slog.Error("failed to start lsp server", "error", err)
		os.Exit(1)
	}

	<-jsonrpc2.NewConn(
		ctx,
		jsonrpc2.NewBufferedStream(stdRWC{}, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.HandlerWithError(s.Handle),
	).DisconnectNotify()
}
