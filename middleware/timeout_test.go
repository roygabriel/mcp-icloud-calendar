package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestTimeoutMiddleware_AppliesDeadline(t *testing.T) {
	mw := TimeoutMiddleware(100 * time.Millisecond)

	handler := mw(func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("expected deadline to be set")
		}
		if time.Until(deadline) > 200*time.Millisecond {
			t.Errorf("deadline too far in the future: %v", time.Until(deadline))
		}
		return mcp.NewToolResultText("ok"), nil
	})

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTimeoutMiddleware_TimesOutSlowHandler(t *testing.T) {
	mw := TimeoutMiddleware(50 * time.Millisecond)

	handler := mw(func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		select {
		case <-time.After(5 * time.Second):
			return mcp.NewToolResultText("ok"), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	_, err := handler(context.Background(), mcp.CallToolRequest{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestTimeoutMiddleware_FastHandlerSucceeds(t *testing.T) {
	mw := TimeoutMiddleware(1 * time.Second)

	handler := mw(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("fast"), nil
	})

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
