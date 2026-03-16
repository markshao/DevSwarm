package tmux

import (
	"os"
	"testing"
)

// TestSessionExists verifies SessionExists returns false for non-existent sessions
func TestSessionExists(t *testing.T) {
	// Use a unique session name that shouldn't exist
	sessionName := "orion-test-nonexistent-" + t.Name()

	// Clean up in case it exists from a previous failed test
	_ = KillSession(sessionName)

	if SessionExists(sessionName) {
		t.Errorf("SessionExists(%q) returned true for non-existent session", sessionName)
	}
}

// TestNewSessionAndKillSession verifies creating and killing a tmux session
func TestNewSessionAndKillSession(t *testing.T) {
	sessionName := "orion-test-session-" + t.Name()

	// Create a temp directory for the session
	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	// Verify session exists
	if !SessionExists(sessionName) {
		t.Errorf("Session should exist after NewSession")
	}

	// Kill session
	if err := KillSession(sessionName); err != nil {
		t.Errorf("KillSession failed: %v", err)
	}

	// Verify session no longer exists
	if SessionExists(sessionName) {
		t.Errorf("Session should not exist after KillSession")
	}
}

// TestSendKeys verifies sending keys to a tmux session
func TestSendKeys(t *testing.T) {
	sessionName := "orion-test-sendkeys-" + t.Name()

	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	defer KillSession(sessionName)

	// Send a simple command (echo)
	if err := SendKeys(sessionName, "echo hello"); err != nil {
		t.Errorf("SendKeys failed: %v", err)
	}
}

// TestIsInsideTmux verifies IsInsideTmux returns expected value
func TestIsInsideTmux(t *testing.T) {
	// This test depends on whether we're running inside tmux
	// We just verify the function doesn't panic and returns a boolean
	inside := IsInsideTmux()
	t.Logf("Running inside tmux: %v", inside)
}

// TestGetCurrentSessionName verifies GetCurrentSessionName behavior
func TestGetCurrentSessionName(t *testing.T) {
	// This test depends on whether we're running inside tmux
	sessionName, err := GetCurrentSessionName()

	if IsInsideTmux() {
		if err != nil {
			t.Errorf("GetCurrentSessionName failed when inside tmux: %v", err)
		}
		if sessionName == "" {
			t.Error("GetCurrentSessionName returned empty string when inside tmux")
		}
	} else {
		if err == nil {
			t.Error("GetCurrentSessionName should return error when not inside tmux")
		}
		if sessionName != "" {
			t.Error("GetCurrentSessionName should return empty string when not inside tmux")
		}
	}
}

// TestEnsureAndAttach verifies session creation and attachment logic
func TestEnsureAndAttach(t *testing.T) {
	sessionName := "orion-test-ensure-" + t.Name()

	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// First call should create the session
	// Note: EnsureAndAttach calls AttachSession which replaces the process,
	// so we can't test the full flow in a unit test.
	// Instead, we test that the session gets created.

	// Create session manually for testing
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	defer KillSession(sessionName)

	// Verify session was created
	if !SessionExists(sessionName) {
		t.Error("Session should exist after EnsureAndAttach setup")
	}
}

// TestSwitchClient verifies switching tmux client
func TestSwitchClient(t *testing.T) {
	sessionName := "orion-test-switch-" + t.Name()

	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	defer KillSession(sessionName)

	// SwitchClient requires being inside tmux, so we just verify it returns
	// an appropriate error when not inside tmux
	if !IsInsideTmux() {
		err := SwitchClient(sessionName)
		if err == nil {
			t.Error("SwitchClient should return error when not inside tmux")
		}
	}
}

// TestKillSessionNonExistent verifies KillSession handles non-existent sessions gracefully
func TestKillSessionNonExistent(t *testing.T) {
	sessionName := "orion-test-nonexistent-kill-" + t.Name()

	// Should not return error for non-existent session
	if err := KillSession(sessionName); err != nil {
		t.Errorf("KillSession should not return error for non-existent session: %v", err)
	}
}
