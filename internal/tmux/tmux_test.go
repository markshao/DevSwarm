package tmux

import (
	"os"
	"os/exec"
	"testing"
)

// TestSessionExists tests the SessionExists function
func TestSessionExists(t *testing.T) {
	// Create a unique session name for testing
	sessionName := "orion-test-session-" + t.Name()

	// Ensure session doesn't exist first
	if SessionExists(sessionName) {
		exec.Command("tmux", "kill-session", "-t", sessionName).Run()
	}

	// Should not exist initially
	if SessionExists(sessionName) {
		t.Errorf("Session %s should not exist initially", sessionName)
	}

	// Create a session
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		t.Skipf("tmux not available, skipping test: %v", err)
	}
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Should exist now
	if !SessionExists(sessionName) {
		t.Errorf("Session %s should exist after creation", sessionName)
	}
}

// TestNewSession tests creating a new tmux session
func TestNewSession(t *testing.T) {
	sessionName := "orion-test-new-session-" + t.Name()
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create session
	err = NewSession(sessionName, cwd)
	if err != nil {
		// Check if tmux is available
		if _, err := exec.LookPath("tmux"); err != nil {
			t.Skipf("tmux not available, skipping test: %v", err)
		}
		t.Fatalf("NewSession failed: %v", err)
	}

	// Verify session exists
	if !SessionExists(sessionName) {
		t.Errorf("Session %s was not created", sessionName)
	}
}

// TestNewSessionWithInvalidPath tests creating a session with invalid path
func TestNewSessionWithInvalidPath(t *testing.T) {
	sessionName := "orion-test-invalid-path-" + t.Name()
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Try to create session with non-existent directory
	invalidPath := "/non/existent/path/that/does/not/exist"
	err := NewSession(sessionName, invalidPath)

	// Note: tmux may still create the session even with invalid path depending on version
	// We just verify the behavior - it may or may not fail
	// The important thing is that it doesn't crash
	if err != nil {
		// If it fails, that's expected
		t.Logf("NewSession failed as expected with invalid path: %v", err)
	}
}

// TestSendKeys tests sending keys to a tmux session
func TestSendKeys(t *testing.T) {
	sessionName := "orion-test-send-keys-" + t.Name()

	// Create session
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		t.Skipf("tmux not available, skipping test: %v", err)
	}
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Send keys
	err := SendKeys(sessionName, "echo hello")
	if err != nil {
		t.Errorf("SendKeys failed: %v", err)
	}

	// Send keys to non-existent session should fail
	err = SendKeys("non-existent-session", "echo hello")
	if err == nil {
		t.Error("SendKeys should fail for non-existent session")
	}
}

// TestIsInsideTmux tests the IsInsideTmux function
func TestIsInsideTmux(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer func() {
		if originalTmux == "" {
			os.Unsetenv("TMUX")
		} else {
			os.Setenv("TMUX", originalTmux)
		}
	}()

	// Test when TMUX is not set
	os.Unsetenv("TMUX")
	if IsInsideTmux() {
		t.Error("IsInsideTmux should return false when TMUX env var is not set")
	}

	// Test when TMUX is set
	os.Setenv("TMUX", "/tmp/tmux-123/default,123,0")
	if !IsInsideTmux() {
		t.Error("IsInsideTmux should return true when TMUX env var is set")
	}
}

// TestGetCurrentSessionName tests getting current session name
func TestGetCurrentSessionName(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer func() {
		if originalTmux == "" {
			os.Unsetenv("TMUX")
		} else {
			os.Setenv("TMUX", originalTmux)
		}
	}()

	// Test when not inside tmux
	os.Unsetenv("TMUX")
	_, err := GetCurrentSessionName()
	if err == nil {
		t.Error("GetCurrentSessionName should fail when not inside tmux")
	}

	// Note: Testing inside tmux requires actual tmux environment, skip for now
}

// TestSwitchClient tests switching tmux client
func TestSwitchClient(t *testing.T) {
	// Save original TMUX env var
	originalTmux := os.Getenv("TMUX")
	defer func() {
		if originalTmux == "" {
			os.Unsetenv("TMUX")
		} else {
			os.Setenv("TMUX", originalTmux)
		}
	}()

	// Test when not inside tmux - should fail
	os.Unsetenv("TMUX")
	err := SwitchClient("some-session")
	if err == nil {
		t.Error("SwitchClient should fail when not inside tmux")
	}
}

// TestKillSession tests killing a tmux session
func TestKillSession(t *testing.T) {
	sessionName := "orion-test-kill-session-" + t.Name()

	// Create session
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		t.Skipf("tmux not available, skipping test: %v", err)
	}

	// Kill session
	err := KillSession(sessionName)
	if err != nil {
		t.Errorf("KillSession failed: %v", err)
	}

	// Verify session is gone
	if SessionExists(sessionName) {
		t.Errorf("Session %s should be killed", sessionName)
	}

	// Killing non-existent session should not error
	err = KillSession("non-existent-session-xyz")
	if err != nil {
		t.Errorf("KillSession should not error for non-existent session, got: %v", err)
	}
}

// TestEnsureAndAttach tests ensuring and attaching to a session
func TestEnsureAndAttach(t *testing.T) {
	sessionName := "orion-test-ensure-attach-" + t.Name()
	defer exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Test creating new session (but don't actually attach as it would replace process)
	// We just test the creation part
	if !SessionExists(sessionName) {
		err = NewSession(sessionName, cwd)
		if err != nil {
			if _, err := exec.LookPath("tmux"); err != nil {
				t.Skipf("tmux not available, skipping test: %v", err)
			}
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	// Verify session was created
	if !SessionExists(sessionName) {
		t.Errorf("Session %s should be created", sessionName)
	}

	// Note: We don't test the actual attach as it would replace the current process
}

// TestSessionLifecycle tests the full lifecycle of a tmux session
func TestSessionLifecycle(t *testing.T) {
	sessionName := "orion-test-lifecycle-" + t.Name()

	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skipf("tmux not available, skipping test: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// 1. Create session
	err = NewSession(sessionName, cwd)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	// 2. Verify exists
	if !SessionExists(sessionName) {
		t.Fatal("Session should exist after creation")
	}

	// 3. Send keys
	err = SendKeys(sessionName, "echo test")
	if err != nil {
		t.Errorf("SendKeys failed: %v", err)
	}

	// 4. Kill session
	err = KillSession(sessionName)
	if err != nil {
		t.Errorf("KillSession failed: %v", err)
	}

	// 5. Verify gone
	if SessionExists(sessionName) {
		t.Error("Session should be killed")
	}
}
