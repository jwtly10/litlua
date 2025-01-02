package lsp

import (
	"fmt"
	"github.com/jwtly10/litlua"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type DocumentProcessor struct {
	parser *litlua.Parser
	writer *Writer
}

func NewDocumentProcessor(parser *litlua.Parser, writer *Writer) *DocumentProcessor {
	return &DocumentProcessor{
		parser: parser,
		writer: writer,
	}
}

// ProcessDocument parsed a Markdown document and transpiles to shadow lua file
func (dp *DocumentProcessor) ProcessDocument(content, filePath, shadowRoot string) (*litlua.Document, string, error) {
	slog.Debug("processing document", "path", filePath)
	doc, err := dp.parser.ParseMarkdownDoc(
		strings.NewReader(content),
		litlua.MetaData{Source: filePath},
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse markdown: %w", err)
	}

	shadowPath, err := generateShadowPath(filePath, shadowRoot)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate shadow path: %w", err)
	}

	if err := dp.writer.WriteToPath(doc, shadowPath); err != nil {
		return nil, "", fmt.Errorf("failed to write shadow file: %w", err)
	}

	return doc, shadowPath, nil
}

func generateShadowPath(rawFilePath, shadowRoot string) (string, error) {
	p := filepath.Join(
		shadowRoot,
		// Mirror the real file path
		rawFilePath,
		"..",
		filepath.Base(rawFilePath)+".lua",
	)

	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return "", err
	}

	slog.Debug("generated shadow path", "path", p)

	return p, nil
}
