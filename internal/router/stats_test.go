package router

import (
	"testing"
	"time"
)

func TestProviderStatsDerivedMetrics(t *testing.T) {
	stats := ProviderStats{
		Requests:     4,
		Errors:       1,
		TotalLatency: 800 * time.Millisecond,
	}

	if got := stats.AverageLatency(); got != 200*time.Millisecond {
		t.Fatalf("expected 200ms average latency, got %v", got)
	}

	if got := stats.ErrorRate(); got != 0.25 {
		t.Fatalf("expected 0.25 error rate, got %f", got)
	}
}
