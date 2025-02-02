package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwtly10/litlua"
	"github.com/jwtly10/litlua/internal/cli"
	"github.com/jwtly10/litlua/internal/transformer"
)

const usage = `LitLua CLI
The LitLua CLI provides manual transformation of LitLua markdown files into Lua

Usage:
  litlua [flags] <input-file>

Examples:
  # Transform a single file with default settings
  $ litlua example.litlua.md

  # Transform cwd directory with default settings
  $ litlua .

  # Enable debug logging while transforming
  $ litlua -debug example.litlua.md

  # Print version information
  $ litlua -version

Flags:
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage)
		flag.PrintDefaults()
	}
	var (
		debug   = flag.Bool("debug", false, "Enable debug logging")
		version = flag.Bool("version", false, "Print version information")
	)

	flag.Parse()

	if *version {
		fmt.Printf("litlua version %s\n", litlua.VERSION)
		os.Exit(0)
	}

	if *debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})))
	}

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	opts := transformer.TransformOptions{
		WriterMode: litlua.ModePretty,
	}

	processor := cli.NewProcessor(opts)

	absPath, err := filepath.Abs(args[0])
	if err != nil {
		fmt.Printf("❌ Failed to resolve absolute path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n🚀 Compilation is running:\n"+
		"  📄 Path     : %s\n",
		absPath)

	results, err := processor.ProcessPath(args[0])
	if err != nil {
		fmt.Printf("❌ Compilation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nCompilation Results:")
	fmt.Printf("%-70s %-30s\n", "Source", "Output")
	fmt.Println(strings.Repeat("-", 110))

	for _, result := range results {
		fmt.Printf("%-70s %-30s \n",
			result.Path,
			result.OutPath,
		)
	}

	fmt.Println(strings.Repeat("-", 110))

	fmt.Printf("\n✨ Compilation complete! Processed %d files\n", len(results))
}
