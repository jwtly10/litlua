package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/jwtly10/litlua"
	"github.com/jwtly10/litlua/internal/transformer"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "A tool for processing Lua blocks in markdown files\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  litlua example.md\n")
		fmt.Fprintf(os.Stderr, "  litlua -debug example.md\n")
	}
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	setLoggingLevel(debug)

	inFile := args[0]

	absPath, err := parseInFile(inFile)
	if err != nil {
		fmt.Printf("Error parsing input file: %v\n", err)
		os.Exit(1)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	opts := transformer.TransformOptions{
		WriterMode: litlua.ModePretty,
	}
	t := transformer.NewTransformer(opts)
	src := transformer.MarkdownSource{
		Content: bytes.NewReader(content),
		Metadata: litlua.MetaData{
			AbsSource: absPath,
		},
	}

	fmt.Printf("\n🚀 Transforming markdown:\n"+
		"  📄 File     : %s\n"+
		"  ⚙️  Options  : %s\n\n",
		absPath,
		opts.Pretty())

	outPath, err := t.Transform(src)
	if err != nil {
		fmt.Printf("Error transforming file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✨ Successfully transformed to ->  %q\n", outPath)
}

func setLoggingLevel(debug bool) {
	if debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})))

	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}
}

func parseInFile(inFile string) (string, error) {
	if inFile == "" {
		return "", fmt.Errorf("no input file provided. Run `litlua` for usage")
	}

	ext := strings.ToLower(filepath.Ext(inFile))
	if ext != ".md" {
		return "", fmt.Errorf("invalid file extension %q. Supported extensions are: .md", ext)
	}

	absPath, err := filepath.Abs(inFile)
	if err != nil {
		return "", fmt.Errorf("error resolving absolute path: %w", err)
	}

	return absPath, nil
}
