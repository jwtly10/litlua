package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jwtly10/litlua/internal/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerOptions(t *testing.T) {
	// so our validation can check these paths are valid
	tempLuaPath := filepath.Join(t.TempDir(), "test-lua-ls")
	tempShadowRoot := filepath.Join(t.TempDir(), "test-shadow-root")
	err := os.MkdirAll(tempLuaPath, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(tempShadowRoot, 0755)
	require.NoError(t, err)

	tests := []struct {
		name        string
		opts        Options
		expectError bool
	}{
		{
			name: "valid options",
			opts: Options{
				LuaLsPath:  tempLuaPath,
				ShadowRoot: tempShadowRoot,
			},
			expectError: false,
		},
		{
			name: "invalid lua path",
			opts: Options{
				LuaLsPath:  "/nonexistent/path",
				ShadowRoot: tempShadowRoot,
			},
			expectError: true,
		},
		{
			name: "invalid shadow root",
			opts: Options{
				LuaLsPath:  tempLuaPath,
				ShadowRoot: "/nonexistent/path",
			},
			expectError: true,
		},
		{
			name:        "empty options - should use defaults",
			opts:        Options{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			server, err := NewServer(tt.opts)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, server)

			// are docservice options being set properly
			if tt.opts.ShadowRoot != "" {
				assert.Equal(t, tt.opts.ShadowRoot, server.docService.ShadowRoot())
			} else {
				assert.NotEmpty(t, server.docService.ShadowRoot())
			}

			// are luals options being set properly
			if tt.opts.LuaLsPath != "" {
				assert.Equal(t, tt.opts.LuaLsPath, server.LuaLS.Path)

			}
		})
	}
}

func TestOptionsOverride(t *testing.T) {
	tempShadowRoot := filepath.Join(t.TempDir(), "shadow-root")
	err := os.MkdirAll(tempShadowRoot, 0755)
	require.NoError(t, err)

	opts := Options{
		ShadowRoot: tempShadowRoot,
	}

	docOpts := lsp.DefaultDocumentServiceOptions
	err = opts.OverrideDocOpts(&docOpts)
	require.NoError(t, err)

	assert.Equal(t, tempShadowRoot, docOpts.ShadowRoot)

	// options remain dont change
	assert.Equal(t, lsp.DefaultDocumentServiceOptions.ShadowTransformerOpts.WriterMode,
		docOpts.ShadowTransformerOpts.WriterMode)
	assert.Equal(t, lsp.DefaultDocumentServiceOptions.FinalTransformerOpts.WriterMode,
		docOpts.FinalTransformerOpts.WriterMode)
}
