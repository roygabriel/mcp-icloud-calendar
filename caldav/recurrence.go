package caldav

import (
	"fmt"
	"time"

	"github.com/teambition/rrule-go"
)

// ExpandRecurrence expands a recurrence rule into individual event occurrences
// within the given time range.
func ExpandRecurrence(event Event, rangeStart, rangeEnd time.Time) ([]Event, error) {
	if event.Recurrence == "" {
		return []Event{event}, nil
	}

	ruleStr := fmt.Sprintf("DTSTART:%s\nRRULE:%s",
		event.StartTime.UTC().Format("20060102T150405Z"),
		event.Recurrence,
	)

	rOption, err := rrule.StrToROption(ruleStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse recurrence rule: %w", err)
	}

	rule, err := rrule.NewRRule(*rOption)
	if err != nil {
		return nil, fmt.Errorf("failed to create recurrence rule: %w", err)
	}

	occurrences := rule.Between(rangeStart, rangeEnd, true)
	duration := event.EndTime.Sub(event.StartTime)

	events := make([]Event, 0, len(occurrences))
	for _, occ := range occurrences {
		e := Event{
			ID:          event.ID,
			Path:        event.Path,
			Title:       event.Title,
			Description: event.Description,
			Location:    event.Location,
			StartTime:   occ,
			EndTime:     occ.Add(duration),
			Recurrence:  event.Recurrence,
			Timezone:    event.Timezone,
			Attendees:   event.Attendees,
		}
		events = append(events, e)
	}

	return events, nil
}
