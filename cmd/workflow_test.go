package cmd

import (
	"testing"

	"orion/internal/workflow"
)

func TestGetTriggerDisplay(t *testing.T) {
	tests := []struct {
		name string
		run  workflow.Run
		want string
	}{
		{
			name: "manual trigger",
			run: workflow.Run{
				ID:       "run-001",
				Workflow: "default",
				Trigger:  "manual",
			},
			want: "manual",
		},
		{
			name: "commit trigger with hash",
			run: workflow.Run{
				ID:          "run-002",
				Workflow:    "default",
				Trigger:     "commit",
				TriggerData: "abc123def",
			},
			want: "commit",
		},
		{
			name: "push trigger",
			run: workflow.Run{
				ID:          "run-003",
				Workflow:    "code-review",
				Trigger:     "push",
				TriggerData: "main",
			},
			want: "push",
		},
		{
			name: "empty trigger",
			run: workflow.Run{
				ID:       "run-004",
				Workflow: "default",
				Trigger:  "",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTriggerDisplay(tt.run)
			if got != tt.want {
				t.Errorf("getTriggerDisplay() = %q, want %q", got, tt.want)
			}
		})
	}
}
