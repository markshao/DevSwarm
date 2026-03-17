package cmd

import (
	"testing"

	"orion/internal/types"

	"github.com/fatih/color"
)

func TestFormatStatus(t *testing.T) {
	// Disable color output for consistent testing
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
			name:     "Empty status defaults to WORKING",
			status:   "",
			expected: "WORKING",
		},
		{
			name:     "Unknown status defaults to WORKING",
			status:   "UNKNOWN",
			expected: "WORKING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)
			if result != tt.expected {
				t.Errorf("formatStatus(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestFormatStatusWithColor(t *testing.T) {
	// Enable color output for this test
	color.NoColor = false
	defer func() { color.NoColor = true }()

	tests := []struct {
		name   string
		status string
	}{
		{
			name:   "StatusWorking",
			status: string(types.StatusWorking),
		},
		{
			name:   "StatusReadyToPush",
			status: string(types.StatusReadyToPush),
		},
		{
			name:   "StatusFail",
			status: string(types.StatusFail),
		},
		{
			name:   "StatusPushed",
			status: string(types.StatusPushed),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)
			// Just verify it returns a non-empty string with the status name
			if result == "" {
				t.Errorf("formatStatus(%q) returned empty string", tt.status)
			}
		})
	}
}
