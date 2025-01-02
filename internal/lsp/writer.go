package lsp

import (
	"fmt"
	"github.com/jwtly10/litlua"
	"io"
	"log/slog"
	"os"
	"strings"
)

type Writer struct {
}

func NewWriter() *Writer {
	return &Writer{}
}

// Write generates a shadow Lua file preserving original line numbers
func (w *Writer) Write(doc *litlua.Document, output io.Writer) error {
	var lines []string

	maxLine := doc.Blocks[len(doc.Blocks)-1].Position.EndLine
	lines = make([]string, maxLine)

	for i := range lines {
		lines[i] = ""
	}

	slog.Debug("writing document to LSP shadow file", "blocks", len(doc.Blocks), "last_line", maxLine, "source", doc.Metadata.Source)

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
		if _, err := fmt.Fprintln(output, line); err != nil {
			return fmt.Errorf("writing line: %w", err)
		}
	}

	return nil
}

func (w *Writer) WriteToPath(doc *litlua.Document, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	return w.Write(doc, f)
}
