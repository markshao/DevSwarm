package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIsSubDir tests the isSubDir helper function
func TestIsSubDir(t *testing.T) {
	// Create a temp directory structure for testing
	tmpDir, err := os.MkdirTemp("", "orion-isSubDir-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories
	subDir := filepath.Join(tmpDir, "sub")
	nestedDir := filepath.Join(tmpDir, "sub", "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dirs: %v", err)
	}

	// Create a sibling directory
	siblingDir := filepath.Join(tmpDir, "sibling")
	if err := os.MkdirAll(siblingDir, 0755); err != nil {
		t.Fatalf("failed to create sibling dir: %v", err)
	}

	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{
			name:   "same directory",
			parent: tmpDir,
			child:  tmpDir,
			want:   true,
		},
		{
			name:   "direct subdirectory",
			parent: tmpDir,
			child:  subDir,
			want:   true,
		},
		{
			name:   "nested subdirectory",
			parent: tmpDir,
			child:  nestedDir,
			want:   true,
		},
		{
			name:   "sibling directory",
			parent: subDir,
			child:  siblingDir,
			want:   false,
		},
		{
			name:   "parent directory",
			parent: subDir,
			child:  tmpDir,
			want:   false,
		},
		{
			name:   "unrelated directory",
			parent: tmpDir,
			child:  "/some/other/path",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubDir(tt.parent, tt.child)
			if got != tt.want {
				t.Errorf("isSubDir(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
			}
		})
	}
}
