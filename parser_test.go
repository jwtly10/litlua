package litlua

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanParseMarkdownDoc(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name     string
		srcFile  string
		document Document
	}{
		{
			name:    "test parse basic markdown doc",
			srcFile: "testdata/valid_markdown_doc.md",
			document: Document{
				Metadata: MetaData{
					Source: "testdata/valid_markdown_doc.md",
				},
				Pragmas: Pragma{
					Output: "init.lua",
					Debug:  true,
				},
				Blocks: []CodeBlock{
					{
						Code:   "print(\"Hello World\")\n",
						Source: "testdata/valid_markdown_doc.md",
					},
					{
						Code:   "print(\"Goodbye World\")\n\n",
						Source: "testdata/valid_markdown_doc.md",
					},
				},
			},
		},
		{
			name:    "test parse basic markdown doc with bad pragmas",
			srcFile: "testdata/valid_markdown_doc_invalid_pragma.md",
			document: Document{
				Metadata: MetaData{
					Source: "testdata/valid_markdown_doc_invalid_pragma.md",
				},
				Pragmas: Pragma{},
				Blocks: []CodeBlock{
					{
						Code:   "print(\"Hello World\")\n",
						Source: "testdata/valid_markdown_doc_invalid_pragma.md",
					},
					{
						Code:   "print(\"Goodbye World\")\n\n",
						Source: "testdata/valid_markdown_doc_invalid_pragma.md",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(tc.srcFile)
			if err != nil {
				t.Errorf("Could not open test source file: %v", err)
			}

			d, err := parser.ParseMarkdownDoc(f, MetaData{
				tc.srcFile,
			})
			if err != nil {
				t.Errorf("Could not parse document: %v", err)
			}

			for i := 0; i < len(d.Blocks); i++ {
				require.Equal(t, tc.document.Blocks[i].Code, d.Blocks[i].Code)
			}

			require.Equal(t, tc.document.Pragmas, d.Pragmas)
			require.Equal(t, tc.document.Metadata, d.Metadata)
		})
	}
}

func TestCanExtractPragmaFromLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected Pragma
		wantErr  bool
	}{
		{
			name: "test basic output pragma",
			line: "<!-- @pragma output: init.lua -->",
			expected: Pragma{
				Output: "init.lua",
			},
		},
		{
			name:     "test ignores invalid pragma",
			line:     "<!-- @pragma invalid: something -->",
			expected: Pragma{},
		},
		{
			name:     "test ignores malformed comment",
			line:     "@pragma output: init.lua",
			expected: Pragma{},
		},
		{
			name:     "test ignores malformed comment if duplicated",
			line:     "<!-- @pragma output: something --><!-- @pragma output: something -->",
			expected: Pragma{},
		},
		{
			name:     "test ignores malformed comment start",
			line:     "@pragma output: init.lua -->",
			expected: Pragma{},
		},
		{
			name:     "test ignores malformed comment end",
			line:     "<!-- @pragma output: init.lua",
			expected: Pragma{},
		},
		{
			name:     "test error when invalid pragma value",
			line:     "<!-- @pragma debug: invalid -->",
			expected: Pragma{},
			wantErr:  true,
		},
	}

	parser := NewParser()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got Pragma
			e := parser.extractPragmaFromLine(&got, tc.line)
			if tc.wantErr {
				require.Error(t, e)
				return
			}
			require.Equal(t, tc.expected, got)
		})
	}
}
