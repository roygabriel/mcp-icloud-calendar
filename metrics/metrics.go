// Package metrics provides Prometheus metrics and middleware for tracking
// MCP tool calls and CalDAV request performance.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ToolCallsTotal counts tool invocations by tool name and status.
	ToolCallsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mcp_tool_calls_total",
		Help: "Total number of MCP tool calls",
	}, []string{"tool", "status"})

	// ToolCallDuration tracks tool call latency.
	ToolCallDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mcp_tool_call_duration_seconds",
		Help:    "Duration of MCP tool calls",
		Buckets: prometheus.DefBuckets,
	}, []string{"tool"})

	// CalDAVRequestsTotal counts CalDAV operations by operation and status.
	CalDAVRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "caldav_requests_total",
		Help: "Total number of CalDAV requests",
	}, []string{"operation", "status"})

	// CalDAVRequestDuration tracks CalDAV operation latency.
	CalDAVRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "caldav_request_duration_seconds",
		Help:    "Duration of CalDAV requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})
)
