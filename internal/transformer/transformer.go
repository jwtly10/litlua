package transformer

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jwtly10/litlua"
)

type TransformOptions struct {
	// The mode for the writer instance
	WriterMode litlua.WriteMode
	// If true, no backup will be created
	NoBackup bool
	// If true, pragma output is required for transformation, otherwise transform will error
	RequirePragmaOutput bool

	// By default, output files are .litlua.lua (safe) otherwise .lua
	NoLitLuaOutputExt bool
}

var InputExt = ".litlua.md"

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

	outputExt string

	opts TransformOptions
}

// NewTransformer creates a new Transformer instance with the specified options [TransformOptions]
func NewTransformer(opts TransformOptions) *Transformer {
	t := &Transformer{
		parser: litlua.NewParser(),
		writer: litlua.NewWriter(opts.WriterMode),
		backup: litlua.NewBackupManager(),
		opts:   opts,
	}

	if !opts.NoLitLuaOutputExt {
		t.outputExt = ".litlua.lua"
	} else {
		t.outputExt = ".lua"
	}

	return t
}

type MarkdownSource struct {
	Content  io.Reader
	Metadata litlua.MetaData
}

// Transform handles standard transformation (using pragmas/default paths)
func (t *Transformer) Transform(input MarkdownSource) (string, error) {
	if t.opts.WriterMode == litlua.ModeShadow {
		return "", fmt.Errorf("cannot use Transform() for shadow mode, use TransformToPath() instead")
	}

	if !strings.HasSuffix(input.Metadata.AbsSource, InputExt) {
		return "", fmt.Errorf("source file must be %s", InputExt)
	}

	return t.transform(input, "")
}

// TransformToPath forces output to a specific path (for lsp shadow files)
func (t *Transformer) TransformToPath(input MarkdownSource, outputPath string) (string, error) {
	if t.opts.WriterMode != litlua.ModeShadow {
		return "", fmt.Errorf("TransformToPath() can only be used with shadow mode")
	}
	if outputPath == "" {
		return "", fmt.Errorf("output path is required for shadow transformation")
	}

	return t.transform(input, outputPath)
}

func (t *Transformer) transform(input MarkdownSource, forcedPath string) (string, error) {
	slog.Debug("transforming document", "path", input.Metadata.AbsSource)
	if input.Metadata.AbsSource == "" {
		return "", fmt.Errorf("abs source metadata is required for transformation")
	}

	doc, err := t.parser.ParseMarkdownDoc(input.Content, input.Metadata)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}

	baseName := filepath.Base(input.Metadata.AbsSource)
	outputBaseName := filepath.Base(doc.Pragmas.Output)
	if baseName == outputBaseName {
		return "", fmt.Errorf("output file cannot have the same name as the input file")
	}

	var absTransformPath string
	if forcedPath != "" {
		absTransformPath = forcedPath
	} else if t.opts.RequirePragmaOutput {
		if doc.Pragmas.Output == "" {
			return "", fmt.Errorf("pragma key 'output' is required for transformation")
		}

		absTransformPath = filepath.Join(filepath.Dir(input.Metadata.AbsSource), t.CleanPragmaOutputExt(doc.Pragmas))
	} else {
		absTransformPath, err = t.resolveTransformToAbsPath(input.Metadata.AbsSource, doc.Pragmas)
		if err != nil {
			return "", fmt.Errorf("resolve output path error: %w", err)
		}
	}

	// Only support creating backups for pretty mode
	var bkPath string
	if t.opts.WriterMode == litlua.ModePretty {
		// If we are not using the litlua extension, we should create a backup, to ensure safety
		// we give the user the option to disable this
		if t.opts.NoLitLuaOutputExt && !t.opts.NoBackup {
			bkPath, err = t.backup.CreateBackupOf(absTransformPath)
			if err != nil {
				return "", fmt.Errorf("backup error: %w", err)
			}
		}
	}

	if bkPath != "" {
		slog.Info("file already existed. Created backup", "backup", bkPath, "original", input.Metadata.AbsSource)
	}

	if err := os.MkdirAll(filepath.Dir(absTransformPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	out, err := os.Create(absTransformPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
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

// CleanPragmaOutputExt uses all pragmas to correctly determine the output path
func (t *Transformer) CleanPragmaOutputExt(pragma litlua.Pragma) string {
	if pragma.Output != "" && pragma.Force {
		return pragma.Output
	}

	clean := strings.TrimSuffix(pragma.Output, ".lua") // remove .lua extension if present
	clean = strings.TrimSuffix(clean, ".litlua")       // remove .litlua extension if present

	return clean + t.outputExt
}

// CleanShadowOutputExt removes the .litlua.md extension from src files and replaces it with .lua
//
// Used by the shadow lsp tranformer to turn a md file like test.litlua.md
// to test.lua in tmp dir for LSP proxying
func (t *Transformer) CleanShadowOutputExt(output string) string {
	return strings.TrimSuffix(output, InputExt) + t.outputExt
}

// resolveOutputPath generate the abs transformed path from the abs src path
func (t *Transformer) resolveTransformToAbsPath(absSrcPath string, pragma litlua.Pragma) (string, error) {
	if pragma.Output == "" {
		// If there is no pragma output, we trim the extension of the file where applicable

		// If extenion is input, trim and add output extension
		if strings.HasSuffix(absSrcPath, InputExt) {
			return strings.TrimSuffix(absSrcPath, InputExt) + t.outputExt, nil
		}

		// If extension is just .md (backwards compat), trim and add ext
		if strings.HasSuffix(absSrcPath, ".md") {
			return strings.TrimSuffix(absSrcPath, ".md") + t.outputExt, nil
		}

		// Else we just append
		return absSrcPath + t.outputExt, nil
	}

	mdDir := filepath.Dir(absSrcPath)
	return filepath.Join(mdDir, t.CleanPragmaOutputExt(pragma)), nil
}
