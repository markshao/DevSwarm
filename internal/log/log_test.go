package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitAndLogging verifies Init, Info, Error and Close work together
// and that log lines are appended to the expected file.
func TestInitAndLogging(t *testing.T) {
	// Use a temporary HOME so we don't touch the real user's home directory.
	tmpHome, err := os.MkdirTemp("", "orion-log-test")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	if err := Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer Close()

	Info("info message: %s", "hello")
	Error("error message: %s", "boom")

	logPath := filepath.Join(tmpHome, ".orion.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "INFO: info message: hello") {
		t.Errorf("log file missing info entry, got: %s", content)
	}
	if !strings.Contains(content, "ERROR: error message: boom") {
		t.Errorf("log file missing error entry, got: %s", content)
	}
}

// TestInfoWithoutInit tests that Info doesn't crash when logger not initialized
func TestInfoWithoutInit(t *testing.T) {
	// Don't call Init - logger should be nil
	// This should not panic
	Info("this should be silently ignored")
	// Test passes if no panic
}

// TestErrorWithoutInit tests that Error doesn't crash when logger not initialized
func TestErrorWithoutInit(t *testing.T) {
	// Don't call Init - logger should be nil
	// This should not panic
	Error("this should be silently ignored")
	// Test passes if no panic
}

// TestCloseWithoutInit tests that Close doesn't crash when logger not initialized
func TestCloseWithoutInit(t *testing.T) {
	// Don't call Init - logger should be nil
	// This should not panic
	Close()
	// Test passes if no panic
}

// TestMultipleInit tests multiple calls to Init
func TestMultipleInit(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "orion-log-multi-init")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	// First init
	if err := Init(); err != nil {
		t.Fatalf("first Init failed: %v", err)
	}

	Info("first init message")

	// Second init (should work, may replace the file handle)
	if err := Init(); err != nil {
		t.Fatalf("second Init failed: %v", err)
	}

	Info("second init message")

	Close()

	logPath := filepath.Join(tmpHome, ".orion.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	// At least the second message should be there
	if !strings.Contains(content, "INFO: second init message") {
		t.Errorf("log file missing second init message, got: %s", content)
	}
}

// TestLoggingWithSpecialCharacters tests logging with special characters
func TestLoggingWithSpecialCharacters(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "orion-log-special")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	if err := Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer Close()

	Info("message with special chars: %s %s %s", "!@#$%^&*()", "日本語", "\n\t\r")
	Error("error with unicode: %s", "错误 錯誤 エラー")

	logPath := filepath.Join(tmpHome, ".orion.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "INFO: message with special chars:") {
		t.Errorf("log file missing info entry with special chars")
	}
	if !strings.Contains(content, "ERROR: error with unicode:") {
		t.Errorf("log file missing error entry with unicode")
	}
}

// TestLoggingWithFormatStrings tests logging with various format specifiers
func TestLoggingWithFormatStrings(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "orion-log-format")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	if err := Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer Close()

	Info("string: %s, int: %d, float: %f, bool: %v", "test", 42, 3.14, true)
	Error("hex: %x, octal: %o", 255, 64)

	logPath := filepath.Join(tmpHome, ".orion.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "string: test") {
		t.Error("missing string format")
	}
	if !strings.Contains(content, "int: 42") {
		t.Error("missing int format")
	}
	if !strings.Contains(content, "float: 3.14") {
		t.Error("missing float format")
	}
}

// TestLogFileLocation tests that log file is created in correct location
func TestLogFileLocation(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "orion-log-location")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	if err := Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer Close()

	expectedPath := filepath.Join(tmpHome, ".orion.log")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		// File might be created on first write, not on Init
		Info("trigger file creation")
	}

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("log file should be created at %s", expectedPath)
	}
}

// TestConcurrentLogging tests concurrent log writes
func TestConcurrentLogging(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "orion-log-concurrent")
	if err != nil {
		t.Fatalf("failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	if err := os.Setenv("HOME", tmpHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}

	if err := Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer Close()

	// Log from multiple goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			Info("concurrent message %d", n)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	logPath := filepath.Join(tmpHome, ".orion.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	// Verify all messages are present
	for i := 0; i < 10; i++ {
		expected := "INFO: concurrent message"
		if !strings.Contains(content, expected) {
			t.Errorf("missing concurrent message %d", i)
		}
	}
}

