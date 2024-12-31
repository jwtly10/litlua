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
	var inFile string
	var debug bool
	flag.StringVar(&inFile, "in", "", "Input markdown file")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Parse()

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

	fmt.Printf("🔍 Processing %s\n", filepath.Base(inFile))

	parser := litlua.NewParser()
	doc, err := parser.ParseMarkdownDoc(f, litlua.MetaData{
		Source: absPath,
	})
	if err != nil {
		fmt.Printf("Error parsing document: %v\n", err)
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
		fmt.Printf("💾 Created backup of existing file to %v\n", backupPath)
	}

	out, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	now := time.Now()
	writer := litlua.NewWriter(out)
	if err := writer.Write(doc, now); err != nil {
		fmt.Printf("Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✨ Successfully wrote output to %s\n", outPath)

}
