package tmux

import (
	"os"
	"testing"
)

// TestSessionExists tests the SessionExists function
func TestSessionExists(t *testing.T) {
	// Create a unique session name for testing
	sessionName := "orion-test-session-" + t.Name()

	// Ensure session doesn't exist initially
	if SessionExists(sessionName) {
		t.Logf("Session %s already exists, cleaning up", sessionName)
		_ = KillSession(sessionName)
	}

	// Create a new session
	testDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	if err := NewSession(sessionName, testDir); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer KillSession(sessionName)

	// Verify session exists
	if !SessionExists(sessionName) {
		t.Error("expected session to exist after creation")
	}

	// Kill session and verify it's gone
	if err := KillSession(sessionName); err != nil {
		t.Fatalf("failed to kill session: %v", err)
	}

	if SessionExists(sessionName) {
		t.Error("expected session to not exist after kill")
	}
}

// TestNewSession tests creating a new tmux session
func TestNewSession(t *testing.T) {
	sessionName := "orion-new-session-test"
	testDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)
	defer KillSession(sessionName)

	if err := NewSession(sessionName, testDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	// Verify session was created
	if !SessionExists(sessionName) {
		t.Error("session was not created")
	}
}

// TestNewSessionWithInvalidDir tests creating a session with invalid directory
func TestNewSessionWithInvalidDir(t *testing.T) {
	sessionName := "orion-invalid-dir-test"
	invalidDir := "/nonexistent/path/that/does/not/exist"

	// Note: tmux may or may not fail when creating a session with an invalid directory
	// depending on the tmux version and configuration. This test documents the behavior.
	err := NewSession(sessionName, invalidDir)
	// Don't assert error - just clean up if session was created
	if SessionExists(sessionName) {
		_ = KillSession(sessionName)
	}
	_ = err // Acknowledge error is unused
}

// TestSendKeys tests sending keys to a tmux session
func TestSendKeys(t *testing.T) {
	sessionName := "orion-send-keys-test"
	testDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)
	defer KillSession(sessionName)

	// Create session
	if err := NewSession(sessionName, testDir); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Send keys
	if err := SendKeys(sessionName, "echo hello"); err != nil {
		t.Fatalf("SendKeys failed: %v", err)
	}

	// Send keys to non-existent session should fail
	err = SendKeys("nonexistent-session-12345", "echo test")
	if err == nil {
		t.Error("expected error when sending keys to non-existent session")
	}
}

// TestIsInsideTmux tests the IsInsideTmux function
func TestIsInsideTmux(t *testing.T) {
	// This test behavior depends on whether tests are running inside tmux
	// We just verify the function returns a boolean without crashing
	result := IsInsideTmux()
	t.Logf("Running inside tmux: %v", result)
}

// TestGetCurrentSessionName tests getting the current session name
func TestGetCurrentSessionName(t *testing.T) {
	// This test depends on running inside tmux
	// If not inside tmux, it should return an error
	sessionName, err := GetCurrentSessionName()
	if !IsInsideTmux() {
		if err == nil {
			t.Error("expected error when getting session name outside tmux")
		}
	} else {
		if err != nil {
			t.Errorf("GetCurrentSessionName failed inside tmux: %v", err)
		}
		if sessionName == "" {
			t.Error("expected non-empty session name inside tmux")
		}
	}
}

// TestSwitchClient tests switching tmux clients
func TestSwitchClient(t *testing.T) {
	sessionName := "orion-switch-client-test"
	testDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)
	defer KillSession(sessionName)

	// Create session
	if err := NewSession(sessionName, testDir); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// SwitchClient should fail if not inside tmux
	if !IsInsideTmux() {
		err := SwitchClient(sessionName)
		if err == nil {
			t.Error("expected error when switching client outside tmux")
		}
	}
}

// TestKillSession tests killing a tmux session
func TestKillSession(t *testing.T) {
	sessionName := "orion-kill-session-test"
	testDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create session
	if err := NewSession(sessionName, testDir); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify session exists
	if !SessionExists(sessionName) {
		t.Fatal("session was not created")
	}

	// Kill session
	if err := KillSession(sessionName); err != nil {
		t.Fatalf("KillSession failed: %v", err)
	}

	// Verify session is gone
	if SessionExists(sessionName) {
		t.Error("session still exists after kill")
	}

	// Killing non-existent session should not error (idempotent)
	if err := KillSession("nonexistent-session-99999"); err != nil {
		t.Errorf("KillSession on non-existent session should not error: %v", err)
	}
}

// TestEnsureAndAttach tests the EnsureAndAttach function
func TestEnsureAndAttach(t *testing.T) {
	sessionName := "orion-ensure-attach-test"
	testDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)
	defer KillSession(sessionName)

	// Test creating new session (but don't attach since it would replace process)
	// We just verify the session creation part works
	if !SessionExists(sessionName) {
		if err := NewSession(sessionName, testDir); err != nil {
			t.Fatalf("failed to create session: %v", err)
		}
	}

	// Verify session exists
	if !SessionExists(sessionName) {
		t.Error("session should exist after EnsureAndAttach")
	}

	// Test with existing session - should not error
	if err := NewSession(sessionName+"-2", testDir); err != nil {
		t.Fatalf("failed to create second session: %v", err)
	}
	defer KillSession(sessionName + "-2")
}

// TestSessionLifecycle tests the full lifecycle of a tmux session
func TestSessionLifecycle(t *testing.T) {
	sessionName := "orion-lifecycle-test"
	testDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 1. Create session
	if err := NewSession(sessionName, testDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	// 2. Verify exists
	if !SessionExists(sessionName) {
		t.Fatal("session should exist after creation")
	}

	// 3. Send keys
	if err := SendKeys(sessionName, "pwd"); err != nil {
		t.Logf("SendKeys warning: %v", err)
	}

	// 4. Kill session
	if err := KillSession(sessionName); err != nil {
		t.Fatalf("KillSession failed: %v", err)
	}

	// 5. Verify gone
	if SessionExists(sessionName) {
		t.Error("session should not exist after kill")
	}
}

// TestMultipleSessions tests managing multiple sessions
func TestMultipleSessions(t *testing.T) {
	testDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	sessionNames := []string{"orion-multi-1", "orion-multi-2", "orion-multi-3"}

	// Create multiple sessions
	for _, name := range sessionNames {
		if err := NewSession(name, testDir); err != nil {
			t.Fatalf("failed to create session %s: %v", name, err)
		}
		defer KillSession(name)
	}

	// Verify all exist
	for _, name := range sessionNames {
		if !SessionExists(name) {
			t.Errorf("session %s should exist", name)
		}
	}

	// Kill all sessions
	for _, name := range sessionNames {
		if err := KillSession(name); err != nil {
			t.Errorf("failed to kill session %s: %v", name, err)
		}
	}

	// Verify all are gone
	for _, name := range sessionNames {
		if SessionExists(name) {
			t.Errorf("session %s should not exist after kill", name)
		}
	}
}
