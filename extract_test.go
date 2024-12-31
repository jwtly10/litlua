package litlua

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"
)

func TestExtractLuaFromMarkdown(t *testing.T) {
	tests := []struct {
		name      string
		inFile    string
		metadata  MetaData
		fixedTime time.Time
	}{
		{
			name:   "basic neovim config",
			inFile: "basic",
			metadata: MetaData{
				Source: "testdata/basic.md",
			},
			fixedTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := os.ReadFile(fmt.Sprintf("testdata/%s.md", tt.inFile))
			require.NoError(t, err)

			parser := NewParser()
			doc, err := parser.ParseMarkdownDoc(bytes.NewReader(input), tt.metadata)
			require.NoError(t, err)

			var buf bytes.Buffer
			writer := NewWriter(&buf)
			err = writer.Write(doc, tt.fixedTime)
			require.NoError(t, err)

			golden.Assert(t, buf.String(), fmt.Sprintf("%s.golden.lua", tt.inFile))
		})
	}
}
