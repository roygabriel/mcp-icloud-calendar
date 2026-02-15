package middleware

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestConcurrencyMiddleware_LimitsMaxConcurrent(t *testing.T) {
	const maxConc = 3
	mw := ConcurrencyMiddleware(maxConc)

	var running atomic.Int32
	var maxSeen atomic.Int32
	var wg sync.WaitGroup

	handler := mw(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cur := running.Add(1)
		for {
			old := maxSeen.Load()
			if cur <= old || maxSeen.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		running.Add(-1)
		return mcp.NewToolResultText("ok"), nil
	})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = handler(context.Background(), mcp.CallToolRequest{})
		}()
	}

	wg.Wait()

	if maxSeen.Load() > int32(maxConc) {
		t.Errorf("max concurrent = %d, want <= %d", maxSeen.Load(), maxConc)
	}
}

func TestConcurrencyMiddleware_ContextCancellation(t *testing.T) {
	// Fill the semaphore completely.
	mw := ConcurrencyMiddleware(1)
	blocking := make(chan struct{})

	handler := mw(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		<-blocking
		return mcp.NewToolResultText("ok"), nil
	})

	// Start one that holds the semaphore.
	go func() {
		_, _ = handler(context.Background(), mcp.CallToolRequest{})
	}()
	time.Sleep(20 * time.Millisecond) // let it acquire

	// Try with a cancelled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := handler(ctx, mcp.CallToolRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	close(blocking) // unblock the first goroutine
}

func TestConcurrencyMiddleware_AllRequestsComplete(t *testing.T) {
	mw := ConcurrencyMiddleware(2)
	var count atomic.Int32

	handler := mw(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		count.Add(1)
		time.Sleep(10 * time.Millisecond)
		return mcp.NewToolResultText("ok"), nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = handler(context.Background(), mcp.CallToolRequest{})
		}()
	}

	wg.Wait()

	if count.Load() != 6 {
		t.Errorf("completed = %d, want 6", count.Load())
	}
}

// Ensure ConcurrencyMiddleware satisfies the ToolHandlerMiddleware type.
var _ server.ToolHandlerMiddleware = ConcurrencyMiddleware(1)
