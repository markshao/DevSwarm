package cmd

import (
	"strings"
	"testing"

	"orion/internal/types"

	"github.com/fatih/color"
)

func TestFormatStatus(t *testing.T) {
	// Disable color for testing to get plain text output
	color.NoColor = true

	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "StatusWorking",
			status:   string(types.StatusWorking),
			expected: "WORKING",
		},
		{
			name:     "StatusReadyToPush",
			status:   string(types.StatusReadyToPush),
			expected: "READY_TO_PUSH",
		},
		{
			name:     "StatusFail",
			status:   string(types.StatusFail),
			expected: "FAIL",
		},
		{
			name:     "StatusPushed",
			status:   string(types.StatusPushed),
			expected: "PUSHED",
		},
		{
			name:     "EmptyStatus",
			status:   "",
			expected: "WORKING",
		},
		{
			name:     "UnknownStatus",
			status:   "UNKNOWN",
			expected: "WORKING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)
			// Strip ANSI color codes for comparison
			result = stripANSI(result)
			if result != tt.expected {
				t.Errorf("formatStatus(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestFormatStatusWithColor(t *testing.T) {
	// Enable color for this test
	color.NoColor = false

	tests := []struct {
		name           string
		status         string
		expectedColor  string // Expected ANSI color code prefix
		shouldContain  string // Expected text content
	}{
		{
			name:          "StatusWorking_Yellow",
			status:        string(types.StatusWorking),
			shouldContain: "WORKING",
		},
		{
			name:          "StatusReadyToPush_Green",
			status:        string(types.StatusReadyToPush),
			shouldContain: "READY_TO_PUSH",
		},
		{
			name:          "StatusFail_Red",
			status:        string(types.StatusFail),
			shouldContain: "FAIL",
		},
		{
			name:          "StatusPushed_HiBlack",
			status:        string(types.StatusPushed),
			shouldContain: "PUSHED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)
			// Verify the result contains the expected text
			if !strings.Contains(result, tt.shouldContain) {
				t.Errorf("formatStatus(%q) should contain %q, got %q", tt.status, tt.shouldContain, result)
			}
			// Verify the result has ANSI codes when color is enabled
			if !hasANSICodes(result) {
				t.Errorf("formatStatus(%q) should have ANSI color codes, got %q", tt.status, result)
			}
		})
	}
}

func TestFormatStatusAllNodeStatusTypes(t *testing.T) {
	color.NoColor = true

	// Test all defined NodeStatus types
	allStatuses := []types.NodeStatus{
		types.StatusWorking,
		types.StatusReadyToPush,
		types.StatusFail,
		types.StatusPushed,
	}

	expectedOutputs := []string{
		"WORKING",
		"READY_TO_PUSH",
		"FAIL",
		"PUSHED",
	}

	for i, status := range allStatuses {
		t.Run(string(status), func(t *testing.T) {
			result := formatStatus(string(status))
			result = stripANSI(result)
			if result != expectedOutputs[i] {
				t.Errorf("formatStatus(%q) = %q, want %q", status, result, expectedOutputs[i])
			}
		})
	}
}

func TestFormatStatusDefaultCase(t *testing.T) {
	color.NoColor = true

	// Test various invalid status values that should default to WORKING
	invalidStatuses := []string{
		"",
		"UNKNOWN",
		"INVALID_STATUS",
		"working", // lowercase should still default
		"ReadyToPush", // wrong case should default
		"random-text",
	}

	for _, status := range invalidStatuses {
		t.Run(status, func(t *testing.T) {
			result := formatStatus(status)
			result = stripANSI(result)
			if status == string(types.StatusWorking) {
				// This is actually valid, skip
				return
			}
			if result != "WORKING" {
				t.Errorf("formatStatus(%q) should default to WORKING, got %q", status, result)
			}
		})
	}
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	result := ""
	i := 0
	for i < len(s) {
		if i < len(s)-1 && s[i] == '\x1b' && s[i+1] == '[' {
			// Skip until we find 'm'
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			i = j + 1
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}

// hasANSICodes checks if a string contains ANSI escape codes
func hasANSICodes(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '\x1b' && s[i+1] == '[' {
			return true
		}
	}
	return false
}
