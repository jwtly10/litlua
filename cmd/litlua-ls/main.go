package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jwtly10/litlua"
	"github.com/jwtly10/litlua/internal/lsp/server"
	"github.com/sourcegraph/jsonrpc2"
)

type stdRWC struct{}

func (stdRWC) Read(p []byte) (int, error)  { return os.Stdin.Read(p) }
func (stdRWC) Write(p []byte) (int, error) { return os.Stdout.Write(p) }
func (stdRWC) Close() error                { return nil }

// getLogFile returns a log file for the lsp server to write to.
//
// During development (-debug flag) uses persistent log for easy access.
func getLogFile(debug bool) (*os.File, error) {
	if debug {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		logDir := filepath.Join(homeDir, ".litlua")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}
		return os.OpenFile(filepath.Join(logDir, "litlua-ls.log"),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}

	return os.CreateTemp("", "litlua-ls-*.log")
}

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Parse()

	logFile, err := getLogFile(debug)
	if err != nil {
		slog.Error("failed to setup logging", "error", err)
		os.Exit(1)
	}
	defer logFile.Close()

	var handler slog.Handler
	if debug {
		handler = slog.NewTextHandler(io.MultiWriter(os.Stderr, logFile), &slog.HandlerOptions{
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

	slog.Info("starting litlua-ls", "logfile", logFile.Name())

	parser := litlua.NewParser()
	lspWriter := litlua.NewWriter(litlua.ModeShadow)

	ctx := context.Background()

	// TODO: Properly configure options
	o := server.Options{}

	s, err := server.NewServer(parser, lspWriter, o)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		return
	}

	<-jsonrpc2.NewConn(
		ctx,
		jsonrpc2.NewBufferedStream(stdRWC{}, jsonrpc2.VSCodeObjectCodec{}),
		jsonrpc2.HandlerWithError(s.Handle),
	).DisconnectNotify()
}
