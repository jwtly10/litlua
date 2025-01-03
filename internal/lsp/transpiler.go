package lsp

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jwtly10/litlua"
)

type DocumentProcessor struct {
	parser *litlua.Parser
	writer *litlua.Writer
}

func NewDocumentProcessor(parser *litlua.Parser, writer *litlua.Writer) *DocumentProcessor {
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

	shadowPath, err := createNewRawShadowPath(filePath, shadowRoot)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate shadow path: %w", err)
	}

	f, err := os.Create(shadowPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create shadow file: %w", err)
	}
	defer f.Close()

	if err := dp.writer.Write(doc, f, litlua.VERSION, time.Now()); err != nil {
		return nil, "", fmt.Errorf("failed to write shadow file: %w", err)
	}

	return doc, shadowPath, nil
}

// createNewRawShadowPath generates a new shadow path for a given markdown file,
// it will create the necessary directories if they do not exist
// note 'raw' means an absolute path without schema
func createNewRawShadowPath(rawFilePath, shadowRoot string) (string, error) {
	p := GetRawShadowPathFromMd(rawFilePath, shadowRoot)

	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return "", err
	}

	slog.Debug("generated shadow path", "path", p)

	return p, nil
}

// getShadowPathFromMd returns the shadow path for a given markdown file
func GetRawShadowPathFromMd(rawFilePath, shadowRoot string) string {
	return filepath.Join(
		shadowRoot,
		// Mirror the real file path
		rawFilePath,
		"..",
		filepath.Base(rawFilePath)+".lua",
	)
}
