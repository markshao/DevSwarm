package cmd

import (
	"testing"

	"orion/internal/version"
)

// TestVersionCmdExists verifies version command is registered
func TestVersionCmdExists(t *testing.T) {
	if versionCmd == nil {
		t.Fatal("versionCmd should not be nil")
	}

	if versionCmd.Use != "version" {
		t.Errorf("versionCmd.Use = %q, want %q", versionCmd.Use, "version")
	}

	if versionCmd.Short == "" {
		t.Error("versionCmd.Short should not be empty")
	}
}

// TestVersionVariables verifies version variables are set
func TestVersionVariables(t *testing.T) {
	// Version should be set
	if version.Version == "" {
		t.Error("version.Version should not be empty")
	}

	// Version should start with 'v'
	if version.Version[0] != 'v' {
		t.Errorf("version.Version %q should start with 'v'", version.Version)
	}
}
