package transformer

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

type testDir struct {
	path string
	t    *testing.T
}

func setupFiles(t *testing.T, srcDir string, destDir *testDir) {
	t.Helper()

	entries, err := os.ReadDir(srcDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "input.md" || entry.Name() == "expected.lua" || entry.Name() == "expected.shadow.lua" {
			continue
		}

		content, err := os.ReadFile(filepath.Join(srcDir, entry.Name()))
		require.NoError(t, err)

		destDir.createFile(entry.Name(), string(content))
	}
}

func newTestDir(t *testing.T) *testDir {
	t.Helper()

	dir, err := os.MkdirTemp("", "transformer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	return &testDir{
		path: dir,
		t:    t,
	}
}

func (td *testDir) cleanup() {
	td.t.Helper()
	if err := os.RemoveAll(td.path); err != nil {
		td.t.Errorf("failed to cleanup test dir: %v", err)
	}
}

func (td *testDir) createFile(name, content string) string {
	td.t.Helper()

	path := filepath.Join(td.path, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		td.t.Fatalf("failed to create test file: %v", err)
	}
	return path
}
