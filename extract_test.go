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

func TestCanHandleExtractingDataFromFiles(t *testing.T) {
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
				Source: "testdata/extract/basic.md",
			},
			fixedTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:   "partial broken src block",
			inFile: "partial_src",
			metadata: MetaData{
				Source: "testdata/extract/partial_src.md",
			},
			fixedTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := os.ReadFile(fmt.Sprintf("testdata/extract/%s.md", tt.inFile))
			require.NoError(t, err)

			parser := NewParser()
			doc, err := parser.ParseMarkdownDoc(bytes.NewReader(input), tt.metadata)
			require.NoError(t, err)

			var buf bytes.Buffer
			writer := NewWriter(ModePretty)
			err = writer.Write(doc, &buf, "v0.0.2", tt.fixedTime)
			require.NoError(t, err)

			golden.Assert(t, buf.String(), fmt.Sprintf("extract/%s.golden.lua", tt.inFile))
		})
	}
}
