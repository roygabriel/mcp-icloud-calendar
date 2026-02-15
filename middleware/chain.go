package middleware

import (
	"github.com/mark3labs/mcp-go/server"
)

// Chain composes multiple middleware into a single middleware.
// The first middleware in the list is the outermost (executes first).
// An empty chain returns the handler unchanged.
func Chain(middlewares ...server.ToolHandlerMiddleware) server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
