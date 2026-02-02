package caldav

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
)

const (
	iCloudBaseURL = "https://caldav.icloud.com"
	timeout       = 30 * time.Second
)

// Client wraps the CalDAV client with iCloud-specific functionality
type Client struct {
	client          *caldav.Client
	calendarHomeSet string
}

// Calendar represents a calendar with its metadata
type Calendar struct {
	Path        string
	Name        string
	Description string
	Color       string
}

// Event represents a calendar event
type Event struct {
	ID          string    `json:"id"`
	Path        string    `json:"path"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Recurrence  string    `json:"recurrence,omitempty"`
	Timezone    string    `json:"timezone"`
}

// NewClient creates a new CalDAV client configured for iCloud
func NewClient(email, password string) (*Client, error) {
	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	// Create basic auth HTTP client
	authClient := webdav.HTTPClientWithBasicAuth(httpClient, email, password)

	// Create CalDAV client
	caldavClient, err := caldav.NewClient(authClient, iCloudBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create CalDAV client: %w", err)
	}

	return &Client{
		client: caldavClient,
	}, nil
}

// DiscoverCalendarHomeSet discovers the user's calendar home set
func (c *Client) DiscoverCalendarHomeSet(ctx context.Context) (string, error) {
	if c.calendarHomeSet != "" {
		return c.calendarHomeSet, nil
	}

	// Find the current user principal
	principal, err := c.client.FindCurrentUserPrincipal(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to find user principal: %w", err)
	}

	// Find the calendar home set for this principal
	homeSet, err := c.client.FindCalendarHomeSet(ctx, principal)
	if err != nil {
		return "", fmt.Errorf("failed to find calendar home set: %w", err)
	}

	c.calendarHomeSet = homeSet
	return homeSet, nil
}

// ListCalendars lists all available calendars
func (c *Client) ListCalendars(ctx context.Context) ([]Calendar, error) {
	homeSet, err := c.DiscoverCalendarHomeSet(ctx)
	if err != nil {
		return nil, err
	}

	caldavCals, err := c.client.FindCalendars(ctx, homeSet)
	if err != nil {
		return nil, fmt.Errorf("failed to find calendars: %w", err)
	}

	calendars := make([]Calendar, 0, len(caldavCals))
	for _, cal := range caldavCals {
		calendars = append(calendars, Calendar{
			Path:        cal.Path,
			Name:        cal.Name,
			Description: cal.Description,
		})
	}

	return calendars, nil
}

// SearchEvents searches for events in a calendar with optional date filters
func (c *Client) SearchEvents(ctx context.Context, calendarPath string, startTime, endTime *time.Time) ([]Event, error) {
	// Build the query
	query := &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name: "VCALENDAR",
			Comps: []caldav.CalendarCompRequest{
				{
					Name:     "VEVENT",
					AllProps: true,
				},
			},
		},
	}

	// Add time range filter if provided
	if startTime != nil || endTime != nil {
		compFilter := caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{
				{
					Name: "VEVENT",
				},
			},
		}

		if startTime != nil {
			compFilter.Comps[0].Start = *startTime
		}
		if endTime != nil {
			compFilter.Comps[0].End = *endTime
		}

		query.CompFilter = compFilter
	}

	calendarObjects, err := c.client.QueryCalendar(ctx, calendarPath, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query calendar: %w", err)
	}

	events := make([]Event, 0, len(calendarObjects))
	for _, obj := range calendarObjects {
		event, err := c.parseCalendarObject(&obj)
		if err != nil {
			// Log but continue processing other events
			continue
		}
		events = append(events, *event)
	}

	return events, nil
}

// CreateEvent creates a new event in the specified calendar
func (c *Client) CreateEvent(ctx context.Context, calendarPath string, event *Event) (string, error) {
	// Create iCalendar object
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//mcp-icloud-calendar//EN")

	// Create event component
	vevent := ical.NewEvent()
	
	// Generate UID if not provided
	uid := event.ID
	if uid == "" {
		uid = fmt.Sprintf("%d@mcp-icloud-calendar", time.Now().UnixNano())
	}
	vevent.Props.SetText(ical.PropUID, uid)
	
	vevent.Props.SetText(ical.PropSummary, event.Title)
	
	if event.Description != "" {
		vevent.Props.SetText(ical.PropDescription, event.Description)
	}
	
	if event.Location != "" {
		vevent.Props.SetText(ical.PropLocation, event.Location)
	}
	
	vevent.Props.SetDateTime(ical.PropDateTimeStart, event.StartTime)
	vevent.Props.SetDateTime(ical.PropDateTimeEnd, event.EndTime)
	vevent.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())

	cal.Children = append(cal.Children, vevent.Component)

	// Create the event path
	eventPath := fmt.Sprintf("%s/%s.ics", strings.TrimSuffix(calendarPath, "/"), uid)

	// Put the calendar object
	_, err := c.client.PutCalendarObject(ctx, eventPath, cal)
	if err != nil {
		return "", fmt.Errorf("failed to create event: %w", err)
	}

	return uid, nil
}

// UpdateEvent updates an existing event
func (c *Client) UpdateEvent(ctx context.Context, eventPath string, event *Event) error {
	// Get the existing event
	existingObj, err := c.client.GetCalendarObject(ctx, eventPath)
	if err != nil {
		return fmt.Errorf("failed to get existing event: %w", err)
	}

	// Find the VEVENT component
	var vevent *ical.Event
	for _, child := range existingObj.Data.Children {
		if child.Name == ical.CompEvent {
			vevent = ical.NewEvent()
			vevent.Component = child
			break
		}
	}

	if vevent == nil {
		return fmt.Errorf("no VEVENT component found in calendar object")
	}

	// Update properties
	if event.Title != "" {
		vevent.Props.SetText(ical.PropSummary, event.Title)
	}
	
	if event.Description != "" {
		vevent.Props.SetText(ical.PropDescription, event.Description)
	}
	
	if event.Location != "" {
		vevent.Props.SetText(ical.PropLocation, event.Location)
	}
	
	if !event.StartTime.IsZero() {
		vevent.Props.SetDateTime(ical.PropDateTimeStart, event.StartTime)
	}
	
	if !event.EndTime.IsZero() {
		vevent.Props.SetDateTime(ical.PropDateTimeEnd, event.EndTime)
	}

	// Update timestamp
	vevent.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())

	// Put the updated calendar object
	_, err = c.client.PutCalendarObject(ctx, eventPath, existingObj.Data)
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

// DeleteEvent deletes an event by its path
func (c *Client) DeleteEvent(ctx context.Context, eventPath string) error {
	err := c.client.RemoveAll(ctx, eventPath)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	return nil
}

// GetEventPath constructs the full path to an event
func (c *Client) GetEventPath(calendarPath, eventID string) string {
	calPath := strings.TrimSuffix(calendarPath, "/")
	if !strings.HasSuffix(eventID, ".ics") {
		eventID = eventID + ".ics"
	}
	return fmt.Sprintf("%s/%s", calPath, eventID)
}

// parseCalendarObject converts a CalDAV calendar object to our Event struct
func (c *Client) parseCalendarObject(obj *caldav.CalendarObject) (*Event, error) {
	// Find the VEVENT component
	var vevent *ical.Event
	for _, child := range obj.Data.Children {
		if child.Name == ical.CompEvent {
			vevent = ical.NewEvent()
			vevent.Component = child
			break
		}
	}

	if vevent == nil {
		return nil, fmt.Errorf("no VEVENT component found")
	}

	event := &Event{
		Path: obj.Path,
	}

	// Extract UID
	if uid := vevent.Props.Get(ical.PropUID); uid != nil {
		event.ID = uid.Value
	}

	// Extract summary (title)
	if summary := vevent.Props.Get(ical.PropSummary); summary != nil {
		event.Title = summary.Value
	}

	// Extract description
	if desc := vevent.Props.Get(ical.PropDescription); desc != nil {
		event.Description = desc.Value
	}

	// Extract location
	if loc := vevent.Props.Get(ical.PropLocation); loc != nil {
		event.Location = loc.Value
	}

	// Extract start time
	if dtstart := vevent.Props.Get(ical.PropDateTimeStart); dtstart != nil {
		startTime, err := dtstart.DateTime(time.UTC)
		if err == nil {
			event.StartTime = startTime
			if tzid := dtstart.Params.Get(ical.PropTimezoneID); tzid != "" {
				event.Timezone = tzid
			}
		}
	}

	// Extract end time
	if dtend := vevent.Props.Get(ical.PropDateTimeEnd); dtend != nil {
		endTime, err := dtend.DateTime(time.UTC)
		if err == nil {
			event.EndTime = endTime
		}
	}

	// Extract recurrence rule
	if rrule := vevent.Props.Get(ical.PropRecurrenceRule); rrule != nil {
		event.Recurrence = rrule.Value
	}

	if event.Timezone == "" {
		event.Timezone = "UTC"
	}

	return event, nil
}
