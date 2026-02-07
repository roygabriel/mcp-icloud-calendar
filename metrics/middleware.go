package metrics

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolCallMiddleware returns middleware that records tool call metrics.
func ToolCallMiddleware() server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			start := time.Now()
			result, err := next(ctx, req)
			duration := time.Since(start).Seconds()

			toolName := req.Params.Name
			ToolCallDuration.WithLabelValues(toolName).Observe(duration)

			status := "success"
			if err != nil || (result != nil && result.IsError) {
				status = "error"
			}
			ToolCallsTotal.WithLabelValues(toolName, status).Inc()

			return result, err
		}
	}
}
