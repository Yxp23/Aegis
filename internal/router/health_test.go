package router

import (
	"fmt"
	"testing"
)

func TestHealthTrackerMarksProviderUnhealthy(t *testing.T) {
	tracker := NewHealthTracker(3)

	if !tracker.IsHealthy("openai") {
		t.Fatal("expected new provider to be healthy")
	}

	tracker.Record("openai", fmt.Errorf("failure"))
	tracker.Record("openai", fmt.Errorf("failure"))

	if !tracker.IsHealthy("openai") {
		t.Fatal("expected provider to remain healthy before threshold")
	}

	tracker.Record("openai", fmt.Errorf("failure"))

	if tracker.IsHealthy("openai") {
		t.Fatal("expected provider to be unhealthy after threshold")
	}

	tracker.Record("openai", nil)

	if !tracker.IsHealthy("openai") {
		t.Fatal("expected successful request to restore health")
	}
}
