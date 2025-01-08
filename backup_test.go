package litlua

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupManager(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		want    string
		wantErr bool
	}{
		{
			name: "no_existing_file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return filepath.Join(dir, "nonexistent.lua")
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "existing_file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "config.lua")
				must(t, os.WriteFile(path, []byte("content"), 0644))
				return path
			},
			want:    ".bak",
			wantErr: false,
		},
		{
			name: "permission_denied",
			setup: func(t *testing.T) string {
				// Sets up the directory to be read-only
				// when the backup logic tries to create a backup file

				dir := t.TempDir()
				must(t, os.Chmod(dir, 0555))

				// Restore permissions after test
				t.Cleanup(func() {
					os.Chmod(dir, 0755)
				})

				path := filepath.Join(dir, "config.lua")

				must(t, os.Chmod(dir, 0755))
				must(t, os.WriteFile(path, []byte("content"), 0644))
				must(t, os.Chmod(dir, 0555))

				return path
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			bm := NewBackupManager()

			got, err := bm.CreateBackupOf(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateBackupOf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.want == "" && got != "" {
					t.Errorf("CreateBackupOf() = %v, want empty string", got)
				} else if tt.want != "" && !strings.HasSuffix(got, tt.want) {
					t.Errorf("CreateBackupOf() = %v, want suffix %v", got, tt.want)
				}

				// Verify backup content matches original if backup was created
				if got != "" {
					original, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					backup, err := os.ReadFile(got)
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(original, backup) {
						t.Error("Backup content doesn't match original")
					}
				}
			}
		})
	}
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
