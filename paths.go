package litlua

import (
	"path/filepath"
	"strings"
)

// ResolveOutputPath determines the final output path from the input markdown source path
func ResolveOutputPath(mdPath string, pragma Pragma) (string, error) {
	if pragma.Output == "" {
		return strings.TrimSuffix(mdPath, filepath.Ext(mdPath)) + ".lua", nil
	}

	mdDir := filepath.Dir(mdPath)
	return filepath.Join(mdDir, pragma.Output), nil
}

func MustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return abs
}
