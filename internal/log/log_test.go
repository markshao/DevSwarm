package log

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestInit tests the Init function
func TestInit(t *testing.T) {
	// Create a temp directory to simulate home
	tmpDir, err := os.MkdirTemp("", "orion-log-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save original home
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Set temp dir as home
	os.Setenv("HOME", tmpDir)

	// Close any existing log file
	Close()

	// Initialize logger
	err = Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify log file was created
	expectedLogPath := filepath.Join(tmpDir, ".orion.log")
	if _, err := os.Stat(expectedLogPath); os.IsNotExist(err) {
		t.Errorf("Log file not created at %s", expectedLogPath)
	}

	// Clean up
	Close()
}

// TestInitWithInvalidHome tests Init with invalid home directory
func TestInitWithInvalidHome(t *testing.T) {
	// Save original home
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Set invalid home
	os.Setenv("HOME", "/non/existent/path/that/cannot/exist")

	// Close any existing log file
	Close()

	// Init should fail
	err := Init()
	if err == nil {
		t.Error("Init should fail with invalid home directory")
	}

	// Restore
	Close()
}

// TestError tests the Error function
func TestError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-log-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)
	Close()

	// Initialize
	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	// Log an error
	testErrorMsg := "test error message"
	Error("test error: %s", testErrorMsg)

	// Flush and read log file
	logFile.Close()

	logPath := filepath.Join(tmpDir, ".orion.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)

	// Verify error was logged
	if !strings.Contains(contentStr, "ERROR") {
		t.Error("Log should contain ERROR level")
	}
	if !strings.Contains(contentStr, testErrorMsg) {
		t.Errorf("Log should contain error message: %s", testErrorMsg)
	}
	if !strings.Contains(contentStr, "test error:") {
		t.Error("Log should contain formatted message")
	}
}

// TestInfo tests the Info function
func TestInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-log-info-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)
	Close()

	// Initialize
	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	// Log an info message
	testInfoMsg := "test info message"
	Info("test info: %s", testInfoMsg)

	// Flush and read log file
	logFile.Close()

	logPath := filepath.Join(tmpDir, ".orion.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)

	// Verify info was logged
	if !strings.Contains(contentStr, "INFO") {
		t.Error("Log should contain INFO level")
	}
	if !strings.Contains(contentStr, testInfoMsg) {
		t.Errorf("Log should contain info message: %s", testInfoMsg)
	}
	if !strings.Contains(contentStr, "test info:") {
		t.Error("Log should contain formatted message")
	}
}

// TestErrorWithoutInit tests Error function without initialization
func TestErrorWithoutInit(t *testing.T) {
	// Ensure log file is nil
	Close()

	// This should not panic
	Error("this should not be logged")

	// Test passes if no panic
}

// TestInfoWithoutInit tests Info function without initialization
func TestInfoWithoutInit(t *testing.T) {
	// Ensure log file is nil
	Close()

	// This should not panic
	Info("this should not be logged")

	// Test passes if no panic
}

// TestClose tests the Close function
func TestClose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-log-close-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)
	Close()

	// Initialize
	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Close
	Close()

	// Close again should not panic
	Close()

	// Test passes if no panic
}

// TestMultipleLogEntries tests multiple log entries
func TestMultipleLogEntries(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-log-multi-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)
	Close()

	// Initialize
	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Log multiple entries
	Error("error 1")
	Info("info 1")
	Error("error 2: %s", "with detail")
	Info("info 2: %d", 42)

	// Flush
	Close()

	// Read and verify
	logPath := filepath.Join(tmpDir, ".orion.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	// Should have 4 lines
	if len(lines) != 4 {
		t.Errorf("Expected 4 log lines, got %d", len(lines))
	}

	// Verify content
	contentStr := string(content)
	if !strings.Contains(contentStr, "error 1") {
		t.Error("Should contain 'error 1'")
	}
	if !strings.Contains(contentStr, "info 1") {
		t.Error("Should contain 'info 1'")
	}
	if !strings.Contains(contentStr, "error 2: with detail") {
		t.Error("Should contain 'error 2: with detail'")
	}
	if !strings.Contains(contentStr, "info 2: 42") {
		t.Error("Should contain 'info 2: 42'")
	}
}

// TestLogTimestamp tests that log entries have timestamps
func TestLogTimestamp(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-log-timestamp-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)
	Close()

	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	Info("test message")
	Close()

	logPath := filepath.Join(tmpDir, ".orion.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)

	// Verify timestamp format [YYYY-MM-DDTHH:MM:SS]
	if !strings.Contains(contentStr, "[") || !strings.Contains(contentStr, "]") {
		t.Error("Log entry should contain timestamp in brackets")
	}

	// Verify it contains current year
	currentYear := time.Now().Format("2006")
	if !strings.Contains(contentStr, currentYear) {
		t.Errorf("Log timestamp should contain current year: %s", currentYear)
	}
}

// TestLogFormat tests the log format
func TestLogFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-log-format-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)
	Close()

	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	testMsg := "format test"
	Error("%s", testMsg)
	Close()

	logPath := filepath.Join(tmpDir, ".orion.log")
	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		// Expected format: [TIMESTAMP] ERROR: message
		if !strings.HasPrefix(line, "[") {
			t.Errorf("Log line should start with '[', got: %s", line)
		}
		if !strings.Contains(line, "] ERROR:") {
			t.Errorf("Log line should contain '] ERROR:', got: %s", line)
		}
		if !strings.HasSuffix(line, testMsg) {
			t.Errorf("Log line should end with message")
		}
	}
}

// TestConcurrentLogging tests concurrent log writes (basic test)
func TestConcurrentLogging(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-log-concurrent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", tmpDir)
	Close()

	if err := Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Log multiple times concurrently
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

	Close()

	// Read and count lines
	logPath := filepath.Join(tmpDir, ".orion.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 10 {
		t.Errorf("Expected 10 log lines, got %d", len(lines))
	}
}
