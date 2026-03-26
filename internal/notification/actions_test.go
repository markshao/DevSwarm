package notification

import "testing"

func TestRouteReplyToNodeAllowsStaleWaitEvent(t *testing.T) {
	rootPath := t.TempDir()
	if err := UpdateRegistry(rootPath, func(registry *Registry) error {
		registry.Watchers["demo"] = &Watcher{
			NodeName:    "demo",
			PaneID:      "%dead",
			State:       StateWaitingInput,
			WaitEventID: 3,
		}
		return nil
	}); err != nil {
		t.Fatalf("seed registry: %v", err)
	}

	meta := CardActionMetadata{NodeName: "demo", WaitEvent: 2}
	err := RouteReplyToNode(rootPath, meta, "hello")
	if err == nil {
		t.Fatalf("expected tmux send failure in test env, got nil")
	}
	if err != nil && err.Error() == "stale wait_event_id for node demo" {
		t.Fatalf("should not reject stale wait_event anymore")
	}
}
