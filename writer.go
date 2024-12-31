package litlua

import (
	"fmt"
	"io"
	"time"
)

type Writer struct {
	output io.Writer
}

func NewWriter(output io.Writer) *Writer {
	return &Writer{
		output: output,
	}
}

// Write writes a parsed Markdown Document to the configured output writer
func (w *Writer) Write(doc *Document, now time.Time) error {
	header := fmt.Sprintf(`-- Generated by LitLua (https://www.github.com/jwtly10/litlua) v0.0.1
-- Source: %s
-- Generated: %s

-- WARNING: This is an auto-generated file.
-- Do not modify this file directly as changes will be overwritten on next compilation.
-- Instead, modify the source markdown file and recompile.

`, doc.Metadata.Source, now.Format(time.RFC3339))

	if _, err := fmt.Fprint(w.output, header); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}

	for _, block := range doc.Blocks {
		if _, err := fmt.Fprintf(w.output, "%s\n", block.Code); err != nil {
			return fmt.Errorf("writing block: %w", err)
		}
	}

	return nil
}
