package caldav

import (
	"context"
	"testing"
	"time"
)

func TestRateLimitedClient_AllowsRequests(t *testing.T) {
	mock := &MockClient{
		Events: []Event{{ID: "e1", Title: "Test"}},
	}
	rl := NewRateLimitedClient(mock, 100, 10) // high limit for test

	events, err := rl.SearchEvents(context.Background(), "/cal", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestRateLimitedClient_BlocksOnCancel(t *testing.T) {
	mock := &MockClient{}
	rl := NewRateLimitedClient(mock, 0.001, 0) // extremely low limit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := rl.SearchEvents(ctx, "/cal", nil, nil)
	if err == nil {
		t.Fatal("expected error from rate limit with cancelled context")
	}
}
