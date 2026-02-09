package middleware

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestRequestIDMiddleware_AddsRequestID(t *testing.T) {
	var capturedCtx context.Context
	inner := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capturedCtx = ctx
		return mcp.NewToolResultText("ok"), nil
	}

	mw := RequestIDMiddleware()
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

	reqID := GetRequestID(capturedCtx)
	if reqID == "" {
		t.Fatal("expected non-empty request ID in context")
	}
}

func TestRequestIDMiddleware_PropagatesError(t *testing.T) {
	inner := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, context.DeadlineExceeded
	}

	mw := RequestIDMiddleware()
	handler := mw(inner)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "test_tool"},
	}

	_, err := handler(context.Background(), req)
	if err == nil {
		t.Fatal("expected error to be propagated")
	}
}

func TestRequestIDMiddleware_ErrorResult(t *testing.T) {
	inner := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("something failed"), nil
	}

	mw := RequestIDMiddleware()
	handler := mw(inner)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "test_tool"},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	id := GetRequestID(context.Background())
	if id != "" {
		t.Errorf("expected empty string for context without request ID, got %q", id)
	}
}

func TestRequestIDMiddleware_UniquePerCall(t *testing.T) {
	var ids []string
	inner := func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ids = append(ids, GetRequestID(ctx))
		return mcp.NewToolResultText("ok"), nil
	}

	mw := RequestIDMiddleware()
	handler := mw(inner)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "test_tool"},
	}

	for i := 0; i < 3; i++ {
		_, err := handler(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if len(ids) != 3 {
		t.Fatalf("expected 3 IDs, got %d", len(ids))
	}

	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate request ID: %q", id)
		}
		seen[id] = true
	}
}

func TestRequestIDMiddleware_NilResult(t *testing.T) {
	inner := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, nil
	}

	mw := RequestIDMiddleware()
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
