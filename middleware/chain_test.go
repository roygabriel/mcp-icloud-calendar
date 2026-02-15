package middleware

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestChain_ExecutionOrder(t *testing.T) {
	var order []string

	mkMiddleware := func(name string) server.ToolHandlerMiddleware {
		return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
			return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				order = append(order, name+"-before")
				result, err := next(ctx, req)
				order = append(order, name+"-after")
				return result, err
			}
		}
	}

	chained := Chain(mkMiddleware("outer"), mkMiddleware("inner"))
	handler := chained(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		order = append(order, "handler")
		return mcp.NewToolResultText("ok"), nil
	})

	_, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"outer-before", "inner-before", "handler", "inner-after", "outer-after"}
	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestChain_EmptyChain(t *testing.T) {
	chained := Chain()
	handler := chained(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("direct"), nil
	})

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestChain_ErrorPropagation(t *testing.T) {
	chained := Chain(func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return next(ctx, req)
		}
	})

	handler := chained(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, fmt.Errorf("handler error")
	})

	_, err := handler(context.Background(), mcp.CallToolRequest{})
	if err == nil || err.Error() != "handler error" {
		t.Errorf("expected 'handler error', got %v", err)
	}
}
