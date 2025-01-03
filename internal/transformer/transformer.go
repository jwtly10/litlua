package transformer

import (
	"fmt"
	"github.com/jwtly10/litlua"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type TransformOptions struct {
	// The mode for the writer instance
	WriterMode litlua.WriteMode
	// If true, no backup will be created
	NoBackup bool
	// If true, pragma output is required for transformation, otherwise transform will error
	RequirePragmaOutput bool
}

func (t *TransformOptions) Pretty() string {
	return fmt.Sprintf("mode=%s backup=%s require_output_pragma=%s",
		writerModeToString(t.WriterMode),
		boolToText(!t.NoBackup),
		boolToText(t.RequirePragmaOutput))
}

func writerModeToString(mode litlua.WriteMode) string {
	switch mode {
	case litlua.ModePretty:
		return "Pretty"
	case litlua.ModeShadow:
		return "Shadow"
	default:
		return fmt.Sprintf("Mode(%d)", mode)
	}
}

func boolToText(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

type Transformer struct {
	parser *litlua.Parser
	writer *litlua.Writer
	backup *litlua.BackupManager

	opts TransformOptions
}

// NewTransformer creates a new Transformer instance with the specified options [TransformOptions]
func NewTransformer(opts TransformOptions) *Transformer {
	return &Transformer{
		parser: litlua.NewParser(),
		writer: litlua.NewWriter(opts.WriterMode),
		backup: litlua.NewBackupManager(),
		opts:   opts,
	}
}

type MarkdownSource struct {
	Content  io.Reader
	Metadata litlua.MetaData
}

// Transform runs the Markdown to Lua transformation and returns the absolute path to the output file [TransformOptions]
func (t *Transformer) Transform(input MarkdownSource) (string, error) {
	slog.Debug("transforming document", "path", input.Metadata.AbsSource)
	if input.Metadata.AbsSource == "" {
		return "", fmt.Errorf("abs source metadata is required for transformation")
	}

	doc, err := t.parser.ParseMarkdownDoc(input.Content, input.Metadata)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}

	var absTransformPath string
	if t.opts.RequirePragmaOutput {
		if doc.Pragmas.Output == "" {
			return "", fmt.Errorf("pragma output is required for transformation")
		}
		// If pragma output is set, use it as the compile path, relative to the source file
		absTransformPath = filepath.Join(filepath.Dir(input.Metadata.AbsSource), doc.Pragmas.Output)
	} else {
		// Else resolve the compile path from the source file (src.md -> src.lua)
		absTransformPath, err = resolveTransformToAbsPath(input.Metadata.AbsSource, doc.Pragmas)
		if err != nil {
			return "", fmt.Errorf("resolve output path error: %w", err)
		}
	}

	var bkPath string
	if t.opts.WriterMode == litlua.ModeShadow || !t.opts.NoBackup {
		bkPath, err = t.backup.CreateBackupOf(absTransformPath)
		if err != nil {
			return "", fmt.Errorf("backup error: %w", err)
		}
	}

	if bkPath != "" {
		slog.Info("file already existed. Created backup", "backup", bkPath, "original", input.Metadata.AbsSource)
	}

	out, err := os.Create(absTransformPath)
	if err != nil {
		fmt.Printf("Error creating transform output file: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	if t.opts.WriterMode == litlua.ModePretty {
		metadata := litlua.WriterMetadata{
			Version:   litlua.VERSION,
			AbsSource: input.Metadata.AbsSource,
			Generated: time.Now().Format(time.RFC3339),
		}
		if err := t.writer.WriteHeader(out, metadata); err != nil {
			return "", fmt.Errorf("write header error: %w", err)
		}
	}

	if err := t.writer.WriteContent(doc, out); err != nil {
		return "", fmt.Errorf("write error: %w", err)
	}

	return absTransformPath, nil
}

// resolveOutputPath generate the abs transformed path from the abs src path
func resolveTransformToAbsPath(absSrcPath string, pragma litlua.Pragma) (string, error) {
	if pragma.Output == "" {
		return strings.TrimSuffix(absSrcPath, filepath.Ext(absSrcPath)) + ".lua", nil
	}

	mdDir := filepath.Dir(absSrcPath)
	return filepath.Join(mdDir, pragma.Output), nil
}
