package metrics

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestToolCallMiddleware_Success(t *testing.T) {
	inner := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	}

	mw := ToolCallMiddleware()
	handler := mw(inner)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "test_tool"},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success result")
	}
}

func TestToolCallMiddleware_Error(t *testing.T) {
	inner := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, fmt.Errorf("internal error")
	}

	mw := ToolCallMiddleware()
	handler := mw(inner)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "test_tool"},
	}

	_, err := handler(context.Background(), req)
	if err == nil {
		t.Fatal("expected error to be propagated")
	}
}

func TestToolCallMiddleware_ErrorResult(t *testing.T) {
	inner := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("tool failed"), nil
	}

	mw := ToolCallMiddleware()
	handler := mw(inner)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "search_events"},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestToolCallMiddleware_NilResult(t *testing.T) {
	inner := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, nil
	}

	mw := ToolCallMiddleware()
	handler := mw(inner)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "test_tool"},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result")
	}
}

func TestToolCallMiddleware_RecordsMetrics(t *testing.T) {
	callCount := 0
	inner := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		callCount++
		return mcp.NewToolResultText("ok"), nil
	}

	mw := ToolCallMiddleware()
	handler := mw(inner)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "create_event"},
	}

	for i := 0; i < 3; i++ {
		_, err := handler(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}
