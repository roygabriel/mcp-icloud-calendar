package caldav

import (
	"testing"
	"time"
)

func TestExpandRecurrence_Daily(t *testing.T) {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	event := Event{
		ID:         "e1",
		Title:      "Daily standup",
		StartTime:  start,
		EndTime:    start.Add(30 * time.Minute),
		Recurrence: "FREQ=DAILY;COUNT=3",
	}

	rangeStart := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)

	events, err := ExpandRecurrence(event, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 occurrences, got %d", len(events))
	}

	// Check first and last occurrence
	if !events[0].StartTime.Equal(start) {
		t.Errorf("first occurrence = %v, want %v", events[0].StartTime, start)
	}
	expected3rd := start.Add(2 * 24 * time.Hour)
	if !events[2].StartTime.Equal(expected3rd) {
		t.Errorf("third occurrence = %v, want %v", events[2].StartTime, expected3rd)
	}

	// Check duration preserved
	for i, e := range events {
		duration := e.EndTime.Sub(e.StartTime)
		if duration != 30*time.Minute {
			t.Errorf("occurrence %d duration = %v, want 30m", i, duration)
		}
	}
}

func TestExpandRecurrence_NoRule(t *testing.T) {
	event := Event{
		ID:    "e1",
		Title: "One-time event",
	}

	events, err := ExpandRecurrence(event, time.Time{}, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("expected 1 event (no recurrence), got %d", len(events))
	}
}

func TestExpandRecurrence_Weekly(t *testing.T) {
	start := time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC)
	event := Event{
		ID:         "e2",
		Title:      "Weekly meeting",
		StartTime:  start,
		EndTime:    start.Add(time.Hour),
		Recurrence: "FREQ=WEEKLY;COUNT=4",
	}

	rangeStart := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	events, err := ExpandRecurrence(event, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 4 {
		t.Errorf("expected 4 occurrences, got %d", len(events))
	}
}
