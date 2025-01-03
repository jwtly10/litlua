package litlua

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type WriteMode int

const (
	// ModePretty Writes with headers and formatting
	ModePretty WriteMode = iota
	// ModeShadow Writes preserving line positions for LSP
	ModeShadow
)

// Writer writes a parsed Markdown Document to the configured output writer
type Writer struct {
	mode WriteMode
}

// WriterMetadata contains metadata for file generation
type WriterMetadata struct {
	Version   string
	AbsSource string
	Generated string // Pre-formatted timestamp string
}

// NewWriter creates a new Writer with the specified write mode [WriteMode]
func NewWriter(mode WriteMode) *Writer {
	return &Writer{
		mode: mode,
	}
}

func (w *Writer) WriteContent(doc *Document, out io.Writer) error {
	switch w.mode {
	case ModePretty:
		return w.writePretty(doc, out)
	case ModeShadow:
		return w.writeShadow(doc, out)
	}
	return fmt.Errorf("invalid write mode")
}

func (w *Writer) WriteHeader(out io.Writer, metadata WriterMetadata) error {
	header := fmt.Sprintf(`-- Generated by LitLua (https://www.github.com/jwtly10/litlua) %s
-- Source: %s
-- Generated: %s

-- WARNING: This is an auto-generated file.
-- Do not modify this file directly as changes will be overwritten on next compilation.
-- Instead, modify the source markdown file and recompile.

`, metadata.Version, metadata.AbsSource, metadata.Generated)

	_, err := fmt.Fprint(out, header)
	return err
}

// writePretty writes a parsed Markdown Document to the configured output writer
func (w *Writer) writePretty(doc *Document, out io.Writer) error {
	for _, block := range doc.Blocks {
		if _, err := fmt.Fprintf(out, "%s\n", block.Code); err != nil {
			return fmt.Errorf("writing block: %w", err)
		}
	}

	slog.Debug("wrote document to output", "blocks", len(doc.Blocks), "source", doc.Metadata.AbsSource, "output", doc.Pragmas.Output)
	return nil
}

// writeShadow generates a shadow Lua file preserving original line numbers
func (w *Writer) writeShadow(doc *Document, out io.Writer) error {
	var lines []string

	maxLine := doc.Blocks[len(doc.Blocks)-1].Position.EndLine
	lines = make([]string, maxLine)

	for i := range lines {
		lines[i] = ""
	}

	slog.Debug("writing document to LSP shadow file", "blocks", len(doc.Blocks), "last_line", maxLine, "source", doc.Metadata.AbsSource)

	for _, block := range doc.Blocks {
		blockLines := strings.Split(block.Code, "\n")

		startLine := block.Position.StartLine

		for i, line := range blockLines {
			actualIndex := startLine + i - 1 // arrays are 0-indexed, but file lines are 1-indexed
			if lines[actualIndex] != "" {
				return fmt.Errorf("line %d already contains code", startLine+i)
			}

			slog.Debug("writing block line", "line", startLine+i, "code", line)
			lines[actualIndex] = line
		}
	}

	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return fmt.Errorf("writing line: %w", err)
		}
	}

	return nil

}
