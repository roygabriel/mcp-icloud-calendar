package metrics

import (
	"testing"
)

func TestMetricsRegistered(t *testing.T) {
	// Verify prometheus metrics are registered and usable
	if ToolCallsTotal == nil {
		t.Fatal("ToolCallsTotal not registered")
	}
	if ToolCallDuration == nil {
		t.Fatal("ToolCallDuration not registered")
	}
	if CalDAVRequestsTotal == nil {
		t.Fatal("CalDAVRequestsTotal not registered")
	}
	if CalDAVRequestDuration == nil {
		t.Fatal("CalDAVRequestDuration not registered")
	}

	// Verify they can be used without panicking
	ToolCallsTotal.WithLabelValues("test_tool", "success").Inc()
	ToolCallDuration.WithLabelValues("test_tool").Observe(0.1)
	CalDAVRequestsTotal.WithLabelValues("search", "success").Inc()
	CalDAVRequestDuration.WithLabelValues("search").Observe(0.05)
}
