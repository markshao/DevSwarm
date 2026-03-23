package notification

import (
	"testing"
	"time"
)

func TestHeuristicClassifyWaitingInput(t *testing.T) {
	screen := `
Would you like to run the following command?
1. Yes, proceed
2. No
Press enter to confirm or esc to cancel
`

	classification := HeuristicClassify(screen, 2*time.Second, 20*time.Second)
	if classification.State != StateWaitingInput {
		t.Fatalf("expected waiting_input, got %s", classification.State)
	}
}

func TestHeuristicClassifyStableQuietOutput(t *testing.T) {
	screen := `
• Completed changes successfully.
• Summary written.
`

	classification := HeuristicClassify(screen, 30*time.Second, 20*time.Second)
	if classification.State != StateQuietCandidate {
		t.Fatalf("expected quiet_candidate, got %s", classification.State)
	}
}

func TestHeuristicClassifyRecentOutput(t *testing.T) {
	screen := `
• Working...
`

	classification := HeuristicClassify(screen, 5*time.Second, 20*time.Second)
	if classification.State != StateRunning {
		t.Fatalf("expected running, got %s", classification.State)
	}
}
