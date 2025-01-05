package cli

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/jwtly10/litlua"
	"github.com/jwtly10/litlua/internal/transformer"
)

const (
	maxFiles      = 100
	maxDepth      = 5
	maxWorkers    = 4
	fileExtension = ".litlua.md"
)

type TranspileResult struct {
	Path     string
	OutPath  string
	Duration time.Duration
}

type ProcessResult struct {
	Path    string
	OutPath string
	Error   error
}

type Processor struct {
	transformer *transformer.Transformer
	opts        transformer.TransformOptions
}

func NewProcessor(opts transformer.TransformOptions) *Processor {
	return &Processor{
		transformer: transformer.NewTransformer(opts),
		opts:        opts,
	}
}

func (p *Processor) ProcessPath(path string) ([]TranspileResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("error accessing path: %w", err)
	}

	if info.IsDir() {
		return p.processDirectory(path)
	}

	result := p.processFile(path)
	if result.Error != nil {
		return nil, result.Error
	}

	return []TranspileResult{{
		Path:    result.Path,
		OutPath: result.OutPath,
	}}, nil
}

// findFiles walks the directory tree starting at root and returns a list of parsable files
//
// If a .git directory is found, it will be used to load .gitignore patterns.
func (p *Processor) findFiles(root string) ([]string, error) {
	var files []string
	var patterns []gitignore.Pattern

	// If .git exists, set up gitignore patterns
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		// Add .git directory pattern
		patterns = append(patterns, gitignore.ParsePattern(".git/", nil))

		// Load .gitignore if it exists
		if data, err := os.ReadFile(filepath.Join(root, ".gitignore")); err == nil {
			for _, p := range strings.Split(string(data), "\n") {
				if p = strings.TrimSpace(p); p != "" && !strings.HasPrefix(p, "#") {
					patterns = append(patterns, gitignore.ParsePattern(p, nil))
				}
			}
		}
	}

	matcher := gitignore.NewMatcher(patterns)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		pathComponents := strings.Split(relPath, string(os.PathSeparator))

		if len(patterns) > 0 {
			if matcher.Match(pathComponents, info.IsDir()) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !info.IsDir() && strings.HasSuffix(path, fileExtension) {
			if len(files) >= maxFiles {
				return fmt.Errorf("max files limit reached (%d)", maxFiles)
			}
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no %s files found", fileExtension)
	}

	return files, nil
}

func (p *Processor) processDirectory(root string) ([]TranspileResult, error) {
	startTime := time.Now()
	slog.Debug("starting directory processing", "path", root)
	files, err := p.findFiles(root)
	if err != nil {
		return nil, err
	}

	slog.Debug("found files to process", "count", len(files), "duration", time.Since(startTime))

	jobs := make(chan string, len(files))
	results := make(chan ProcessResult, len(files))

	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				results <- p.processFile(path)
			}
		}()
	}

	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var errors []error
	var transpileResults []TranspileResult

	for result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("failed to process %s: %w", result.Path, result.Error))
			slog.Debug("failed to process file", "path", result.Path, "error", result.Error)
			continue
		}

		absRoot, _ := filepath.Abs(root)
		relSource, _ := filepath.Rel(absRoot, result.Path)
		relOut, _ := filepath.Rel(absRoot, result.OutPath)

		transpileResults = append(transpileResults, TranspileResult{
			Path:    relSource,
			OutPath: relOut,
		})

		slog.Debug("file transpiled",
			"source", relSource,
			"output", relOut,
		)
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("encountered %d errors during compilation. Please rerun with -debug to see trace", len(errors))
	}

	slog.Debug("compilation completed", "duration", time.Since(startTime), "processed", len(transpileResults))
	return transpileResults, nil
}

func (p *Processor) processFile(path string) ProcessResult {
	startTime := time.Now()
	var result ProcessResult

	absPath, err := filepath.Abs(path)
	if err != nil {
		result.Error = fmt.Errorf("failed to resolve absolute path: %w", err)
		return result
	}

	result.Path = absPath

	slog.Debug("processing file", "path", absPath)

	if !strings.HasSuffix(absPath, fileExtension) {
		result.Error = fmt.Errorf("invalid file extension, expected %s", fileExtension)
		return result
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		result.Error = fmt.Errorf("error reading file: %w", err)
		return result
	}

	src := transformer.MarkdownSource{
		Content: bytes.NewReader(content),
		Metadata: litlua.MetaData{
			AbsSource: absPath,
		},
	}

	outPath, err := p.transformer.Transform(src)
	if err != nil {
		result.Error = err
		return result
	}

	result.OutPath = outPath
	result.OutPath = outPath
	slog.Debug("file processed",
		"path", absPath,
		"duration", time.Since(startTime))

	return result
}
