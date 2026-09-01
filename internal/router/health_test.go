package router

import (
	"fmt"
	"testing"
	"time"
)

func TestHealthTrackerMarksProviderUnhealthy(t *testing.T) {
	tracker := NewHealthTracker(3, 30*time.Second)

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

func TestHealthTrackerAllowsSingleRecoveryProbe(t *testing.T) {
	tracker := NewHealthTracker(3, 10*time.Millisecond)

	for range 3 {
		tracker.Record("openai", fmt.Errorf("failure"))
	}

	time.Sleep(20 * time.Millisecond)

	if !tracker.ShouldAllow("openai") {
		t.Fatal("expected first recovery probe to be allowed")
	}

	if tracker.ShouldAllow("openai") {
		t.Fatal("expected second recovery probe to be blocked")
	}

	tracker.Record("openai", nil)

	if !tracker.ShouldAllow("openai") {
		t.Fatal("expected healthy provider to be allowed")
	}
}
