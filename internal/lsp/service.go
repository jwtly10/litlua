package lsp

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jwtly10/litlua"
	"github.com/jwtly10/litlua/internal/transformer"
	"github.com/sourcegraph/go-lsp"
)

type DocumentServiceOptions struct {
	ShadowTransformerOpts transformer.TransformOptions
	FinalTransformerOpts  transformer.TransformOptions

	// Root directory for shadow files
	ShadowRoot string
}

var DefaultDocumentServiceOptions = DocumentServiceOptions{
	ShadowRoot: filepath.Join(os.TempDir(), "litlua-workspace"),
	ShadowTransformerOpts: transformer.TransformOptions{
		WriterMode:          litlua.ModeShadow,
		NoBackup:            true,
		RequirePragmaOutput: false,
		NoLitLuaOutputExt:   false,
	},
	FinalTransformerOpts: transformer.TransformOptions{
		WriterMode:          litlua.ModePretty,
		NoBackup:            false,
		RequirePragmaOutput: true,
		NoLitLuaOutputExt:   false,
	},
}

func (o DocumentServiceOptions) Validate() error {
	if o.ShadowRoot == "" {
		return fmt.Errorf("shadow root directory is required")
	}

	return nil
}

// DocumentService handles all document transformations and path mappings
type DocumentService struct {
	// Maps shadow URIs to original URIs, which include a mirror of the source file structure
	//
	// shadow_file = file:///tmp/Users/personal/Projects/litlua/lsp_example.md.lua
	// original    = file:///Users/personal/Projects/litlua/testdata/lsp_example.md
	shadowMap         map[string]string
	shadowTransformer *transformer.Transformer
	// The root directory for shadow files eg /tmp/litlua
	shadowRoot string

	// The transformer used for 'final' transformation
	finalTransformer *transformer.Transformer
}

func NewDocumentService(opts DocumentServiceOptions) (*DocumentService, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid document service options: %w", err)
	}

	d := &DocumentService{
		shadowTransformer: transformer.NewTransformer(opts.ShadowTransformerOpts),
		shadowRoot:        opts.ShadowRoot,
		shadowMap:         make(map[string]string),
		finalTransformer:  transformer.NewTransformer(opts.FinalTransformerOpts),
	}

	// Cleanup shadow files on GC finalization
	runtime.SetFinalizer(d, func(d *DocumentService) {
		if err := d.CleanupShadowFiles(); err != nil {
			slog.Error("failed to cleanup shadow files", "error", err)
		}
	})

	return d, nil
}

// TransformShadowDoc transforms the document for LSP proxying and returns the shadow URI
func (s *DocumentService) TransformShadowDoc(text string, documentURI lsp.DocumentURI) (shadowURI string, err error) {
	fsPath, err := s.URIToPath(documentURI)
	if err != nil {
		return "", fmt.Errorf("invalid document URI: %w", err)
	}

	// Create shadow file path
	// in the shadow root directory, with the same directory structure as the original file, but with transformer configured output extension
	shadowPath := filepath.Join(s.shadowRoot, filepath.Dir(fsPath)+s.shadowTransformer.CleanShadowOutputExt(filepath.Base(fsPath)))
	if err := os.MkdirAll(filepath.Dir(shadowPath), 0755); err != nil {
		return "", err
	}

	source := transformer.MarkdownSource{
		Content: strings.NewReader(text),
		Metadata: litlua.MetaData{
			AbsSource: fsPath,
		},
	}

	transformedPath, err := s.shadowTransformer.TransformToPath(source, shadowPath)
	if err != nil {
		return "", fmt.Errorf("transform error: %w", err)
	}

	shadowURI = s.PathToURI(transformedPath)
	originalURI := string(documentURI)
	s.shadowMap[shadowURI] = originalURI

	slog.Debug("transformed document",
		"original", originalURI,
		"transformed", transformedPath,
		"shadow", shadowURI,
	)

	return shadowURI, nil
}

// TransformFinalDoc transforms a document for final 'compilation' output, returning the absolute path of the output file
func (s *DocumentService) TransformFinalDoc(text string, sourcePath string) (string, error) {
	source := transformer.MarkdownSource{
		Content: strings.NewReader(text),
		Metadata: litlua.MetaData{
			AbsSource: sourcePath,
		},
	}

	transformedPath, err := s.finalTransformer.Transform(source)
	if err != nil {
		return "", fmt.Errorf("transform error: %w", err)
	}

	return transformedPath, nil
}

// ShadowRoot returns the root directory for shadow files
func (s *DocumentService) ShadowRoot() string {
	return s.shadowRoot
}

// OriginalURI returns the original document URI for a shadow file
func (s *DocumentService) OriginalURI(shadowURI string) (string, bool) {
	uri, exists := s.shadowMap[shadowURI]
	return uri, exists
}

// ShadowURI returns the shadow URI for an original document URI
func (s *DocumentService) ShadowURI(originalURI string) (string, bool) {
	for shadow, original := range s.shadowMap {
		if original == originalURI {
			return shadow, true
		}
	}
	return "", false
}

// URIToPath converts an LSP URI to a filesystem path
func (s *DocumentService) URIToPath(uri lsp.DocumentURI) (string, error) {
	u, err := url.Parse(string(uri))
	if err != nil {
		return "", err
	}
	return u.Path, nil
}

// PathToURI converts a filesystem path to an LSP URI
func (s *DocumentService) PathToURI(path string) string {
	return "file://" + path
}

// CleanupShadowFiles removes all shadow files
func (s *DocumentService) CleanupShadowFiles() error {
	if s.shadowRoot != DefaultDocumentServiceOptions.ShadowRoot {
		slog.Info("skipping shadow file cleanup due to user specified", "path", s.shadowRoot)
		return nil
	}

	return filepath.WalkDir(s.shadowRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Warn("error accessing path", "path", path, "error", err)
			return nil
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), "litlua.lua") {
			if err := os.Remove(path); err != nil {
				slog.Warn("failed to remove shadow file", "path", path, "error", err)
			} else {
				slog.Debug("removed shadow file", "path", path)
			}
		}
		return nil
	})
}
