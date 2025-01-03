package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jwtly10/litlua"
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

	inFile := args[0]

	if debug {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}

	if inFile == "" {
		fmt.Println("Please provide an input file with -in")
		os.Exit(1)
	}

	ext := strings.ToLower(filepath.Ext(inFile))
	if ext != ".md" {
		fmt.Printf("Error: Invalid file extension %q. Supported extensions are: .md\n", ext)
		os.Exit(1)
	}

	absPath, err := filepath.Abs(inFile)
	if err != nil {
		fmt.Printf("Error resolving absolute path: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Open(inFile)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Printf("\n🔍 LitLua is running! Source: %s\n\n", filepath.Base(inFile))

	parser := litlua.NewParser()
	doc, err := parser.ParseMarkdownDoc(f, litlua.MetaData{
		Source: absPath,
	})
	if err != nil {
		fmt.Printf("Error parsing source file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📝 Found %d Lua blocks to process\n", len(doc.Blocks))

	outPath, err := litlua.ResolveOutputPath(inFile, doc.Pragmas)
	if err != nil {
		fmt.Printf("Error resolving output path: %v\n", err)
		os.Exit(1)
	}

	backupMgr := litlua.NewBackupManager(outPath)
	backupPath, err := backupMgr.CreateBackup()
	if err != nil {
		fmt.Printf("Error creating backup: %v\n", err)
		os.Exit(1)
	}

	if backupPath != "" {
		fmt.Printf("💾 Created backup of existing file to %v\n", litlua.MustAbs(backupPath))
	}

	out, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	now := time.Now()
	writer := litlua.NewWriter(litlua.ModePretty)
	if err := writer.Write(doc, out, litlua.VERSION, now); err != nil {
		fmt.Printf("Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✨ Successfully wrote output to %s\n", litlua.MustAbs(outPath))
}
