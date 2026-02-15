package middleware

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ConcurrencyMiddleware returns middleware that limits concurrent tool executions
// using a buffered channel as a semaphore.
func ConcurrencyMiddleware(max int) server.ToolHandlerMiddleware {
	sem := make(chan struct{}, max)
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				return next(ctx, req)
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
}
