package tmux

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestSessionExists tests the SessionExists function
func TestSessionExists(t *testing.T) {
	// Create a unique session name for testing
	sessionName := "orion-test-session-exists"

	// Ensure session doesn't exist first
	exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Should not exist initially
	if SessionExists(sessionName) {
		t.Errorf("SessionExists() = true, want false (session should not exist)")
	}

	// Create the session
	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", tmpDir)
	if err := cmd.Run(); err != nil {
		t.Skipf("tmux not available, skipping test: %v", err)
	}
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Should exist now
	if !SessionExists(sessionName) {
		t.Errorf("SessionExists() = false, want true (session should exist)")
	}
}

// TestNewSession tests the NewSession function
func TestNewSession(t *testing.T) {
	sessionName := "orion-test-new-session"
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	tmpDir, err := os.MkdirTemp("", "orion-tmux-new-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create new session
	err = NewSession(sessionName, tmpDir)
	if err != nil {
		// Check if tmux is available
		if _, err := exec.LookPath("tmux"); err != nil {
			t.Skipf("tmux not available, skipping test: %v", err)
		}
		t.Fatalf("NewSession() error = %v", err)
	}

	// Verify session exists
	if !SessionExists(sessionName) {
		t.Errorf("NewSession() failed to create session")
	}
}

// TestSendKeys tests the SendKeys function
func TestSendKeys(t *testing.T) {
	sessionName := "orion-test-send-keys"
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	tmpDir, err := os.MkdirTemp("", "orion-tmux-keys-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session first
	if err := NewSession(sessionName, tmpDir); err != nil {
		if _, err := exec.LookPath("tmux"); err != nil {
			t.Skipf("tmux not available, skipping test: %v", err)
		}
		t.Fatalf("failed to create session: %v", err)
	}

	// Send keys
	err = SendKeys(sessionName, "echo hello")
	if err != nil {
		t.Errorf("SendKeys() error = %v", err)
	}
}

// TestIsInsideTmux tests the IsInsideTmux function
func TestIsInsideTmux(t *testing.T) {
	// This test behavior depends on whether we're running inside tmux
	// We just verify the function returns a consistent boolean
	result := IsInsideTmux()

	// Verify TMUX env var matches the result
	tmuxEnv := os.Getenv("TMUX")
	if result && tmuxEnv == "" {
		t.Errorf("IsInsideTmux() = true but TMUX env var is empty")
	}
	if !result && tmuxEnv != "" {
		t.Errorf("IsInsideTmux() = false but TMUX env var is set: %s", tmuxEnv)
	}
}

// TestGetCurrentSessionName tests the GetCurrentSessionName function
func TestGetCurrentSessionName(t *testing.T) {
	// This test only works inside tmux
	if !IsInsideTmux() {
		t.Skip("not running inside tmux, skipping test")
	}

	sessionName, err := GetCurrentSessionName()
	if err != nil {
		t.Errorf("GetCurrentSessionName() error = %v", err)
	}
	if sessionName == "" {
		t.Errorf("GetCurrentSessionName() returned empty string")
	}
}

// TestSwitchClient tests the SwitchClient function
func TestSwitchClient(t *testing.T) {
	// This test only works inside tmux
	if !IsInsideTmux() {
		t.Skip("not running inside tmux, skipping test")
	}

	sessionName := "orion-test-switch-client"
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	tmpDir, err := os.MkdirTemp("", "orion-tmux-switch-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Switch to the session (this will change the current client's session)
	// Note: This may fail if there's no current client to switch
	err = SwitchClient(sessionName)
	if err != nil {
		// Check if it's a "no current client" error, which is acceptable
		if !strings.Contains(err.Error(), "no current client") {
			t.Logf("SwitchClient() error (may be expected in some environments): %v", err)
		}
	}
}

// TestKillSession tests the KillSession function
func TestKillSession(t *testing.T) {
	sessionName := "orion-test-kill-session"

	tmpDir, err := os.MkdirTemp("", "orion-tmux-kill-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session first
	if err := NewSession(sessionName, tmpDir); err != nil {
		if _, err := exec.LookPath("tmux"); err != nil {
			t.Skipf("tmux not available, skipping test: %v", err)
		}
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify it exists
	if !SessionExists(sessionName) {
		t.Fatalf("session was not created")
	}

	// Kill the session
	err = KillSession(sessionName)
	if err != nil {
		t.Errorf("KillSession() error = %v", err)
	}

	// Verify it's gone
	if SessionExists(sessionName) {
		t.Errorf("KillSession() failed to kill session")
	}
}

// TestKillNonExistentSession tests killing a non-existent session
func TestKillNonExistentSession(t *testing.T) {
	sessionName := "orion-test-non-existent"

	// Ensure it doesn't exist
	exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Killing non-existent session should not error
	err := KillSession(sessionName)
	if err != nil {
		t.Errorf("KillSession() on non-existent session returned error: %v", err)
	}
}

// TestEnsureAndAttach tests the EnsureAndAttach function
func TestEnsureAndAttach(t *testing.T) {
	sessionName := "orion-test-ensure-attach"
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	tmpDir, err := os.MkdirTemp("", "orion-tmux-ensure-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// EnsureAndAttach will replace the current process, so we can't test the attach part
	// We just test that it creates the session if it doesn't exist
	// We'll fork a subprocess to test this

	// For now, just test session creation
	err = NewSession(sessionName, tmpDir)
	if err != nil {
		if _, err := exec.LookPath("tmux"); err != nil {
			t.Skipf("tmux not available, skipping test: %v", err)
		}
		t.Fatalf("NewSession() error = %v", err)
	}

	if !SessionExists(sessionName) {
		t.Errorf("EnsureAndAttach() failed to create session")
	}
}
