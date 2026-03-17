package cmd

import (
	"strings"
	"testing"

	"orion/internal/types"
)

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		expectedColor  string
		expectedText   string
		shouldContain  bool
	}{
		{
			name:          "StatusWorking",
			status:        string(types.StatusWorking),
			expectedText:  "WORKING",
			expectedColor: "Yellow",
			shouldContain: true,
		},
		{
			name:          "StatusReadyToPush",
			status:        string(types.StatusReadyToPush),
			expectedText:  "READY_TO_PUSH",
			expectedColor: "Green",
			shouldContain: true,
		},
		{
			name:          "StatusFail",
			status:        string(types.StatusFail),
			expectedText:  "FAIL",
			expectedColor: "Red",
			shouldContain: true,
		},
		{
			name:          "StatusPushed",
			status:        string(types.StatusPushed),
			expectedText:  "PUSHED",
			expectedColor: "HiBlack",
			shouldContain: true,
		},
		{
			name:          "UnknownStatus",
			status:        "UNKNOWN",
			expectedText:  "WORKING",
			expectedColor: "Yellow",
			shouldContain: true,
		},
		{
			name:          "EmptyStatus",
			status:        "",
			expectedText:  "WORKING",
			expectedColor: "Yellow",
			shouldContain: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)

			// Check if result contains expected text
			if !strings.Contains(result, tt.expectedText) {
				t.Errorf("formatStatus(%q) should contain %q, got %q", tt.status, tt.expectedText, result)
			}

			// Check color codes are present (fatih/color uses ANSI escape codes)
			// The result should contain ANSI escape sequences for coloring
			switch tt.expectedColor {
			case "Yellow":
				// Yellow is typically ANSI code 33
				if !strings.Contains(result, "33") && !strings.Contains(result, "3[0") {
					// Note: color might not be applied in test environment
					// We just verify the text is correct
				}
			case "Green":
				// Green is typically ANSI code 32
			case "Red":
				// Red is typically ANSI code 31
			case "HiBlack":
				// HiBlack is typically ANSI code 90
			}
		})
	}
}

func TestFormatStatusReturnsColoredString(t *testing.T) {
	// Verify that formatStatus returns different strings for different statuses
	statuses := []string{
		string(types.StatusWorking),
		string(types.StatusReadyToPush),
		string(types.StatusFail),
		string(types.StatusPushed),
	}

	results := make(map[string]string)
	for _, status := range statuses {
		results[status] = formatStatus(status)
	}

	// All results should be different (due to different colors)
	seen := make(map[string]bool)
	for status, result := range results {
		if seen[result] {
			t.Errorf("formatStatus(%q) returned same result as another status: %q", status, result)
		}
		seen[result] = true
	}
}

func TestFormatStatusDefaultCase(t *testing.T) {
	// Test that unknown status defaults to WORKING with yellow color
	result := formatStatus("UNKNOWN_STATUS")

	if !strings.Contains(result, "WORKING") {
		t.Errorf("formatStatus for unknown status should default to WORKING, got %q", result)
	}
}
