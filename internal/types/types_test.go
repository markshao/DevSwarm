package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNodeStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected string
	}{
		{"StatusWorking", StatusWorking, "WORKING"},
		{"StatusReadyToPush", StatusReadyToPush, "READY_TO_PUSH"},
		{"StatusFail", StatusFail, "FAIL"},
		{"StatusPushed", StatusPushed, "PUSHED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, tt.status)
			}
		})
	}
}

func TestNodeStatusJSONSerialization(t *testing.T) {
	tests := []struct {
		name         string
		status       NodeStatus
		expectedJSON string
	}{
		{"StatusWorking", StatusWorking, `"WORKING"`},
		{"StatusReadyToPush", StatusReadyToPush, `"READY_TO_PUSH"`},
		{"StatusFail", StatusFail, `"FAIL"`},
		{"StatusPushed", StatusPushed, `"PUSHED"`},
		{"EmptyStatus", "", `""`}, // Empty status marshals to empty string
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshal
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if string(data) != tt.expectedJSON {
				t.Errorf("Marshal: expected %s, got %s", tt.expectedJSON, string(data))
			}

			// Test unmarshal
			var unmarshaled NodeStatus
			if err := json.Unmarshal([]byte(tt.expectedJSON), &unmarshaled); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if unmarshaled != tt.status {
				t.Errorf("Unmarshal: expected %v, got %v", tt.status, unmarshaled)
			}
		})
	}
}

func TestNodeWithStatusJSONSerialization(t *testing.T) {
	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		BaseBranch:    "main",
		ShadowBranch:  "orion-shadow/test-node/feature/test",
		WorktreePath:  "/tmp/test-worktree",
		Label:         "test",
		CreatedBy:     "user",
		Status:        StatusReadyToPush,
		AppliedRuns:   []string{"run-1", "run-2"},
		CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	// Marshal
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal back
	var unmarshaled Node
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify fields
	if unmarshaled.Name != node.Name {
		t.Errorf("Name mismatch: expected %s, got %s", node.Name, unmarshaled.Name)
	}
	if unmarshaled.Status != node.Status {
		t.Errorf("Status mismatch: expected %v, got %v", node.Status, unmarshaled.Status)
	}
	if len(unmarshaled.AppliedRuns) != len(node.AppliedRuns) {
		t.Errorf("AppliedRuns length mismatch: expected %d, got %d", len(node.AppliedRuns), len(unmarshaled.AppliedRuns))
	}
}

func TestNodeWithEmptyStatusJSONSerialization(t *testing.T) {
	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		Status:        "", // Empty status
		CreatedAt:     time.Now(),
	}

	// Marshal
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify status is omitted when empty (due to omitempty)
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}

	if _, exists := result["status"]; exists {
		t.Errorf("Expected status to be omitted when empty, but it was present in JSON: %s", string(data))
	}
}

func TestNodeStatusComparison(t *testing.T) {
	tests := []struct {
		name     string
		status1  NodeStatus
		status2  NodeStatus
		expected bool
	}{
		{"SameStatus", StatusWorking, StatusWorking, true},
		{"DifferentStatus", StatusWorking, StatusReadyToPush, false},
		{"EmptyStatus", "", "", true},
		{"EmptyVsWorking", "", StatusWorking, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status1 == tt.status2
			if result != tt.expected {
				t.Errorf("Expected %v == %v to be %v", tt.status1, tt.status2, tt.expected)
			}
		})
	}
}

func TestNodeStatusFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected NodeStatus
	}{
		{"WorkingString", "WORKING", StatusWorking},
		{"ReadyToPushString", "READY_TO_PUSH", StatusReadyToPush},
		{"FailString", "FAIL", StatusFail},
		{"PushedString", "PUSHED", StatusPushed},
		{"UnknownString", "UNKNOWN", "UNKNOWN"}, // Unknown status should just be the string
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NodeStatus(tt.input)
			if result != tt.expected {
				t.Errorf("Expected NodeStatus(%q) to be %v, got %v", tt.input, tt.expected, result)
			}
		})
	}
}
