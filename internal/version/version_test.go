package version

import (
	"strings"
	"testing"
)

// TestVersionVariables tests that version variables are properly initialized
func TestVersionVariables(t *testing.T) {
	// Test Version is set
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// Test Version has expected format (semver-like)
	if !strings.HasPrefix(Version, "v") {
		t.Errorf("Version should start with 'v', got %q", Version)
	}

	// Test Commit is set
	if Commit == "" {
		t.Error("Commit should not be empty")
	}

	// Test Date is set
	if Date == "" {
		t.Error("Date should not be empty")
	}
}

// TestVersionInfoFormat tests the format of version information
func TestVersionInfoFormat(t *testing.T) {
	// Verify version string format
	// Expected: v1.0.0-alpha.7 or similar semver format
	parts := strings.Split(Version, ".")
	if len(parts) < 3 {
		t.Errorf("Version should have at least 3 parts (major.minor.patch), got %q", Version)
	}

	// First part should start with 'v' followed by a number
	major := parts[0]
	if !strings.HasPrefix(major, "v") {
		t.Errorf("Version major part should start with 'v', got %q", major)
	}
}
