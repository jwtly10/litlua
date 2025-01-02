package lsp

import (
	"github.com/jwtly10/litlua"
	"gotest.tools/v3/golden"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestCanWriteToLspFile(t *testing.T) {
	slog.SetDefault(
		slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))

	tests := []struct {
		name       string
		document   litlua.Document
		goldenFile string
	}{
		{
			name:       "can write document struct to lsp shadow file",
			goldenFile: "lsp_shadow.golden.lua",
			// This document is the parser output of testdata/parser/basic_valid.md
			document: litlua.Document{
				Metadata: litlua.MetaData{},
				Pragmas: litlua.Pragma{
					Output: "init.lua",
					Debug:  true,
				},
				Blocks: []litlua.CodeBlock{
					{
						Code: "print(\"Hello World\")",
						Position: litlua.Position{
							StartLine: 10,
							EndLine:   11,
						},
					},
					{
						Code: "print(\"Goodbye World\")\n",
						Position: litlua.Position{
							StartLine: 15,
							EndLine:   17,
						},
					},
					{
						Code: "print(\"Goodbye World\")\n-- This is a multiline lua src",
						Position: litlua.Position{
							StartLine: 20,
							EndLine:   22,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder
			lspWriter := NewWriter()

			if err := lspWriter.Write(&tt.document, &output); err != nil {
				t.Fatalf("Error writing document: %v", err)
			}

			slog.Info(output.String())

			golden.Assert(t, output.String(), tt.goldenFile)
		})
	}
}
