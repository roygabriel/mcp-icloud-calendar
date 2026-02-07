package middleware

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type requestIDKey struct{}

// RequestIDMiddleware returns middleware that generates a UUID per tool call
// and adds it to the context and slog fields.
func RequestIDMiddleware() server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			reqID := uuid.New().String()
			ctx = context.WithValue(ctx, requestIDKey{}, reqID)
			slog.InfoContext(ctx, "tool call started",
				"request_id", reqID,
				"tool", req.Params.Name,
			)
			result, err := next(ctx, req)
			status := "success"
			if err != nil || (result != nil && result.IsError) {
				status = "error"
			}
			slog.InfoContext(ctx, "tool call completed",
				"request_id", reqID,
				"tool", req.Params.Name,
				"status", status,
			)
			return result, err
		}
	}
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}
