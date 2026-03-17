package cmd

import (
	"strings"
	"testing"

	"orion/internal/types"
)

// TestFormatStatus tests the formatStatus function for different node statuses
func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		wantText string
	}{
		{
			name:      "Working status",
			status:    string(types.StatusWorking),
			wantText:  "WORKING",
		},
		{
			name:      "ReadyToPush status",
			status:    string(types.StatusReadyToPush),
			wantText:  "READY_TO_PUSH",
		},
		{
			name:      "Fail status",
			status:    string(types.StatusFail),
			wantText:  "FAIL",
		},
		{
			name:      "Pushed status",
			status:    string(types.StatusPushed),
			wantText:  "PUSHED",
		},
		{
			name:      "Empty status (legacy)",
			status:    "",
			wantText:  "WORKING",
		},
		{
			name:      "Unknown status",
			status:    "UNKNOWN",
			wantText:  "WORKING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStatus(tt.status)

			// Check that the status text is present (ignoring ANSI color codes)
			// Strip ANSI codes for comparison
			stripped := stripANSI(got)
			if stripped != tt.wantText {
				t.Errorf("formatStatus(%q) = %q, want %q", tt.status, stripped, tt.wantText)
			}
		})
	}
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
		} else if inEscape && r == 'm' {
			inEscape = false
		} else if !inEscape {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// TestFormatStatusColorConsistency tests that the same status always produces the same color
func TestFormatStatusColorConsistency(t *testing.T) {
	statuses := []string{
		string(types.StatusWorking),
		string(types.StatusReadyToPush),
		string(types.StatusFail),
		string(types.StatusPushed),
	}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			// Call multiple times and ensure consistency
			result1 := formatStatus(status)
			result2 := formatStatus(status)
			result3 := formatStatus(status)

			if result1 != result2 || result2 != result3 {
				t.Errorf("formatStatus(%q) is not consistent: %q vs %q vs %q", status, result1, result2, result3)
			}
		})
	}
}

// TestFormatStatusWithAllNodeStatuses tests formatStatus with all defined NodeStatus constants
func TestFormatStatusWithAllNodeStatuses(t *testing.T) {
	allStatuses := []types.NodeStatus{
		types.StatusWorking,
		types.StatusReadyToPush,
		types.StatusFail,
		types.StatusPushed,
	}

	for _, status := range allStatuses {
		t.Run(string(status), func(t *testing.T) {
			result := formatStatus(string(status))
			stripped := stripANSI(result)

			// The stripped result should match the status string
			if stripped != string(status) {
				t.Errorf("formatStatus(%q) stripped = %q, want %q", status, stripped, string(status))
			}
		})
	}
}
