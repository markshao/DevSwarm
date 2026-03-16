package version

import (
	"strings"
	"testing"
)

// TestVersionNotEmpty verifies Version is not empty
func TestVersionNotEmpty(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

// TestVersionFormat verifies Version follows semantic versioning pattern
func TestVersionFormat(t *testing.T) {
	// Version should start with 'v' followed by numbers
	if !strings.HasPrefix(Version, "v") {
		t.Errorf("Version %q should start with 'v'", Version)
	}
}

// TestCommitNotEmpty verifies Commit is not empty
func TestCommitNotEmpty(t *testing.T) {
	// In development, Commit might be "none", but should not be empty
	if Commit == "" {
		t.Error("Commit should not be empty")
	}
}

// TestCommitFormat verifies Commit format when not "none"
func TestCommitFormat(t *testing.T) {
	// If commit is not "none" or "unknown", it should be a valid git hash (hex string)
	if Commit != "none" && Commit != "unknown" {
		// Git commit hashes are typically 40 characters (full) or 7+ (short)
		if len(Commit) < 7 {
			t.Errorf("Commit %q seems too short to be a valid git hash", Commit)
		}
	}
}

// TestDateNotEmpty verifies Date is not empty
func TestDateNotEmpty(t *testing.T) {
	// In development, Date might be "unknown", but should not be empty
	if Date == "" {
		t.Error("Date should not be empty")
	}
}

// TestDateFormat verifies Date format when not "unknown"
func TestDateFormat(t *testing.T) {
	// If date is not "unknown", it should be a valid date string
	if Date != "unknown" {
		// Basic check: should contain some date-like content
		if len(Date) < 8 {
			t.Errorf("Date %q seems too short to be a valid date", Date)
		}
	}
}
