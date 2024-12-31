package litlua

import (
	"path/filepath"
	"testing"
)

func TestResolveOutputPath(t *testing.T) {
	tests := []struct {
		name    string
		mdPath  string
		pragma  Pragma
		want    string
		wantErr bool
	}{
		{
			name:   "no_pragma_simple",
			mdPath: "config.md",
			pragma: Pragma{},
			want:   "config.lua",
		},
		{
			name:   "no_pragma_with_path",
			mdPath: "/home/user/nvim/config.md",
			pragma: Pragma{},
			want:   "/home/user/nvim/config.lua",
		},
		{
			name:   "with_pragma_relative",
			mdPath: "config.md",
			pragma: Pragma{
				Output: "init.lua",
			},
			want: "init.lua",
		},
		{
			name:   "with_pragma_and_path",
			mdPath: "/home/user/nvim/config.md",
			pragma: Pragma{
				Output: "init.lua",
			},
			want: "/home/user/nvim/init.lua",
		},
		{
			name:   "different_extension",
			mdPath: "config.luadoc",
			pragma: Pragma{},
			want:   "config.lua",
		},
		{
			name:   "nested_path_no_pragma",
			mdPath: "configs/nvim/init.md",
			pragma: Pragma{},
			want:   "configs/nvim/init.lua",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveOutputPath(tt.mdPath, tt.pragma)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveOutputPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Use filepath.Clean to normalize paths for comparison
			if filepath.Clean(got) != filepath.Clean(tt.want) {
				t.Errorf("ResolveOutputPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveOutputPath1(t *testing.T) {
	type args struct {
		mdPath string
		pragma Pragma
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveOutputPath(tt.args.mdPath, tt.args.pragma)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveOutputPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveOutputPath() got = %v, want %v", got, tt.want)
			}
		})
	}
}
