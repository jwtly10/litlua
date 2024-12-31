package main

import (
	"flag"
	"fmt"
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
	}

	if inFile == "" {
		fmt.Println("Please provide an input file with -in")
		os.Exit(1)
	}

	f, err := os.Open(inFile)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	parser := litlua.NewParser()
	doc, err := parser.ParseMarkdownDoc(f, litlua.MetaData{
		Source: inFile,
	})
	if err != nil {
		fmt.Printf("Error parsing document: %v\n", err)
		os.Exit(1)
	}

	outPath := doc.Pragmas.Output
	if outPath == "" {
		outPath = strings.TrimSuffix(inFile, filepath.Ext(inFile)) + ".lua"
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

	fmt.Printf("Wrote %s to %s\n", inFile, outPath)
}
