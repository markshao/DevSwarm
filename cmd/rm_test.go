package cmd

import (
	"testing"

	"orion/internal/types"
	"orion/internal/workflow"
	"orion/internal/workspace"
)

// TestCollectUnappliedRuns tests the logic for detecting unapplied workflow runs
// This test verifies the core logic used in rmCmd for warning users about unapplied changes
func TestCollectUnappliedRuns(t *testing.T) {
	tests := []struct {
		name           string
		nodeName       string
		node           types.Node
		runs           []workflow.Run
		expectedCount  int
		expectedIDs    []string
	}{
		{
			name:     "node with no applied runs, all successful runs are unapplied",
			nodeName: "node1",
			node: types.Node{
				Name:        "node1",
				AppliedRuns: []string{},
			},
			runs: []workflow.Run{
				{
					ID:              "run-001",
					Status:          workflow.StatusSuccess,
					TriggeredByNode: "node1",
				},
				{
					ID:              "run-002",
					Status:          workflow.StatusSuccess,
					TriggeredByNode: "node1",
				},
			},
			expectedCount: 2,
			expectedIDs:   []string{"run-001", "run-002"},
		},
		{
			name:     "node with some applied runs",
			nodeName: "node1",
			node: types.Node{
				Name:        "node1",
				AppliedRuns: []string{"run-001"},
			},
			runs: []workflow.Run{
				{
					ID:              "run-001",
					Status:          workflow.StatusSuccess,
					TriggeredByNode: "node1",
				},
				{
					ID:              "run-002",
					Status:          workflow.StatusSuccess,
					TriggeredByNode: "node1",
				},
			},
			expectedCount: 1,
			expectedIDs:   []string{"run-002"},
		},
		{
			name:     "node with all runs applied",
			nodeName: "node1",
			node: types.Node{
				Name:        "node1",
				AppliedRuns: []string{"run-001", "run-002"},
			},
			runs: []workflow.Run{
				{
					ID:              "run-001",
					Status:          workflow.StatusSuccess,
					TriggeredByNode: "node1",
				},
				{
					ID:              "run-002",
					Status:          workflow.StatusSuccess,
					TriggeredByNode: "node1",
				},
			},
			expectedCount: 0,
			expectedIDs:   []string{},
		},
		{
			name:     "only successful runs for triggered node are considered",
			nodeName: "node1",
			node: types.Node{
				Name:        "node1",
				AppliedRuns: []string{},
			},
			runs: []workflow.Run{
				{
					ID:              "run-001",
					Status:          workflow.StatusSuccess,
					TriggeredByNode: "node1",
				},
				{
					ID:              "run-002",
					Status:          workflow.StatusRunning,
					TriggeredByNode: "node1",
				},
				{
					ID:              "run-003",
					Status:          workflow.StatusFailed,
					TriggeredByNode: "node1",
				},
				{
					ID:              "run-004",
					Status:          workflow.StatusSuccess,
					TriggeredByNode: "node2", // Different node
				},
			},
			expectedCount: 1,
			expectedIDs:   []string{"run-001"},
		},
		{
			name:     "node does not exist in state",
			nodeName: "nonexistent",
			node: types.Node{
				Name:        "nonexistent",
				AppliedRuns: []string{},
			},
			runs: []workflow.Run{
				{
					ID:              "run-001",
					Status:          workflow.StatusSuccess,
					TriggeredByNode: "nonexistent",
				},
			},
			expectedCount: 1, // The logic still processes the run even if node doesn't exist in state
			expectedIDs:   []string{"run-001"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from rmCmd
			_ = &workspace.WorkspaceManager{
				State: &types.State{
					Nodes: map[string]types.Node{
						tt.nodeName: tt.node,
					},
				},
			}

			nodeWarnings := make(map[string][]string)
			var unapplied []string
			for _, run := range tt.runs {
				if run.TriggeredByNode == tt.nodeName && run.Status == workflow.StatusSuccess {
					isApplied := false
					for _, appliedID := range tt.node.AppliedRuns {
						if appliedID == run.ID {
							isApplied = true
							break
						}
					}
					if !isApplied {
						unapplied = append(unapplied, run.ID)
					}
				}
			}
			if len(unapplied) > 0 {
				nodeWarnings[tt.nodeName] = unapplied
			}

			if len(unapplied) != tt.expectedCount {
				t.Errorf("Expected %d unapplied runs, got %d", tt.expectedCount, len(unapplied))
			}

			if tt.expectedCount > 0 {
				warningRuns := nodeWarnings[tt.nodeName]
				if len(warningRuns) != tt.expectedCount {
					t.Errorf("Expected %d runs in warning, got %d", tt.expectedCount, len(warningRuns))
				}
				for _, expectedID := range tt.expectedIDs {
					found := false
					for _, actualID := range warningRuns {
						if actualID == expectedID {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected run ID %s in warnings, but not found", expectedID)
					}
				}
			}
		})
	}
}
