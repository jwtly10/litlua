package litlua

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var pragmaRegex = regexp.MustCompile(`^<!--\s*@pragma\s+(\w+)\s*:\s*([^>]+?)\s*-->$`)

type Parser struct {
	gm goldmark.Markdown
}

func NewParser() *Parser {
	return &Parser{
		gm: goldmark.New(),
	}
}

// ParseMarkdownDoc parses Markdown content into a document
//
// It pulls out compilation pragmas and lua code blocks from the content and returns a [Document]
func (p *Parser) ParseMarkdownDoc(r io.Reader, md MetaData) (*Document, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	doc := &Document{
		Metadata: md,
	}

	hasWalkedOtherNodes := false
	nodes := p.gm.Parser().Parse(text.NewReader(content))

	err = p.walkAst(nodes, content, &hasWalkedOtherNodes, doc)
	if err != nil {
		return nil, err
	}

	if len(doc.Blocks) == 0 {
		return nil, fmt.Errorf("no lua code blocks found in document")
	}

	return doc, nil
}

func getLineNumber(content []byte, byteOffset int) int {
	return bytes.Count(content[:byteOffset], []byte("\n")) + 1
}

// walkAst walks the AST of a markdown document and extracts pragmas and code blocks
// from the document
func (p *Parser) walkAst(doc ast.Node, content []byte, hasWalkedOtherNodes *bool, result *Document) error {
	return ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			// Entering is true BEFORE walking children, false after walking child
			// this way we only trigger the logic when entering a node,
			// and don-t re-trigger upon exiting
			return ast.WalkContinue, nil
		}

		if _, ok := n.(*ast.HTMLBlock); !ok {
			if _, isDoc := n.(*ast.Document); !isDoc {
				// Markdown files start with document node, so we can skip this if we see it first
				// Otherwise we should no longer try to parse pragmas in future comments
				// as we know they are not at the top of the file
				*hasWalkedOtherNodes = true
			} else {
			}
		}

		switch node := n.(type) {
		case *ast.HTMLBlock:
			if err := p.handleHTMLBlock(node, content, hasWalkedOtherNodes, result); err != nil {
				return ast.WalkStop, err
			}
		case *ast.FencedCodeBlock:
			if err := p.handleCodeBlock(node, content, result); err != nil {
				return ast.WalkStop, err
			}
		}

		return ast.WalkContinue, nil
	})
}

// handleHTMLBlock parses pragma values from HTML comments in markdown.
//
// # Only HTML comments at the top of the .litlua.md file are considered pragmas
//
// For example:
//
// [SOF]
//
// <!-- @pragma output: init.lua -->
//
// <!-- @pragma debug: true -->
//
// [EOF]
//
// will set the [Pragma] struct to have Output = "init.lua" and Debug = true
//
// [SOF]
//
// # Some title
//
// <!-- @pragma output: init.lua -->
//
// <!-- @pragma debug: true -->
//
// [EOF]
//
// will not set the [Pragma] struct as the comments are not at the top of the file
func (p *Parser) handleHTMLBlock(hb *ast.HTMLBlock, content []byte, hasWalkedOtherNodes *bool, doc *Document) error {
	slog.Debug("parsing html block", "hasWalkedOtherNodes", *hasWalkedOtherNodes)
	if !*hasWalkedOtherNodes && hb.HTMLBlockType == ast.HTMLBlockType2 {
		var buf bytes.Buffer
		l := hb.Lines().Len()
		for i := 0; i < l; i++ {
			line := hb.Lines().At(i)
			buf.Write(line.Value(content))
		}
		err := p.extractPragmaFromLine(&doc.Pragmas, buf.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) handleCodeBlock(cb *ast.FencedCodeBlock, content []byte, doc *Document) error {
	lang := string(cb.Language(content))
	if lang != "lua" {
		return nil
	}

	lines := cb.Lines()

	// If the code block is empty, we can skip it
	if lines.Len() == 0 {
		return nil
	}

	startLine := getLineNumber(content, lines.At(0).Start)
	endLine := getLineNumber(content, lines.At(lines.Len()-1).Stop)

	var buf bytes.Buffer
	l := lines.Len()
	for i := 0; i < l; i++ {
		line := lines.At(i)
		buf.Write(line.Value(content))
	}

	block := CodeBlock{
		// We trim the last \n since the md parsing always appends a newline, even when not needed
		Code:   buf.String(),
		Source: doc.Metadata.AbsSource,
		Position: Position{
			startLine,
			endLine,
		},
	}

	slog.Debug("parsed code block", "block", block)

	doc.Blocks = append(doc.Blocks, block)
	return nil
}

// extractPragmaFromLine parses pragma values from markdown comments
//
// A pragma line may look like this: <!-- @pragma output: init.lua -->
//
// In which case we will parse this as a keymap pair "output":"init.lua"
// and if the key maps to a valid value on the [Pragma] struct, set the value.
//
// If multiple lines contain the same key, the last one will be used.
//
// Will return an error if the value cannot be parsed
func (p *Parser) extractPragmaFromLine(pragma *Pragma, line string) error {
	line = strings.TrimSpace(line)
	slog.Debug("parsing pragma line", "line", line)

	matches := pragmaRegex.FindStringSubmatch(line)
	if len(matches) != 3 {
		slog.Debug("invalid pragma line", "line", line)
		return nil
	}

	key := matches[1]
	value := matches[2]

	slog.Debug("parsed pragma key value pair", "key", key, "value", value)

	switch key {
	case string(PragmaOutput):
		pragma.Output = value
	case string(PragmaDebug):
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("could not parse debug pragma value: %w", err)
		}
		pragma.Debug = b
	case string(PragmaForce):
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("could not parse force pragma value: %w", err)
		}
		pragma.Force = b
	default:
		slog.Debug("unknown pragma key", "key", key)
	}

	return nil
}
