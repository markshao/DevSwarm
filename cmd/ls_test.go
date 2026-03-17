package cmd

import (
	"testing"

	"github.com/fatih/color"
)

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		wantFunc func() string
	}{
		{
			name:   "StatusWorking returns yellow WORKING",
			status: "WORKING",
			wantFunc: func() string {
				return color.YellowString("WORKING")
			},
		},
		{
			name:   "StatusReadyToPush returns green READY_TO_PUSH",
			status: "READY_TO_PUSH",
			wantFunc: func() string {
				return color.GreenString("READY_TO_PUSH")
			},
		},
		{
			name:   "StatusFail returns red FAIL",
			status: "FAIL",
			wantFunc: func() string {
				return color.RedString("FAIL")
			},
		},
		{
			name:   "StatusPushed returns hi-black PUSHED",
			status: "PUSHED",
			wantFunc: func() string {
				return color.HiBlackString("PUSHED")
			},
		},
		{
			name:   "unknown status defaults to yellow WORKING",
			status: "UNKNOWN",
			wantFunc: func() string {
				return color.YellowString("WORKING")
			},
		},
		{
			name:   "empty status defaults to yellow WORKING",
			status: "",
			wantFunc: func() string {
				return color.YellowString("WORKING")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStatus(tt.status)
			want := tt.wantFunc()
			if got != want {
				t.Errorf("formatStatus(%q) = %q, want %q", tt.status, got, want)
			}
		})
	}
}
