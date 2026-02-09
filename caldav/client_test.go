package caldav

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emersion/go-ical"
	extcaldav "github.com/emersion/go-webdav/caldav"
)

// mockBackend implements the backend interface for testing client.go methods.
type mockBackend struct {
	principal    string
	principalErr error

	homeSet    string
	homeSetErr error

	calendars  []extcaldav.Calendar
	findCalErr error

	queryResult []extcaldav.CalendarObject
	queryErr    error

	putResult *extcaldav.CalendarObject
	putErr    error

	getResult *extcaldav.CalendarObject
	getErr    error

	removeErr error

	// tracking
	lastPutPath    string
	lastGetPath    string
	lastRemovePath string
}

func (m *mockBackend) FindCurrentUserPrincipal(_ context.Context) (string, error) {
	return m.principal, m.principalErr
}

func (m *mockBackend) FindCalendarHomeSet(_ context.Context, _ string) (string, error) {
	return m.homeSet, m.homeSetErr
}

func (m *mockBackend) FindCalendars(_ context.Context, _ string) ([]extcaldav.Calendar, error) {
	return m.calendars, m.findCalErr
}

func (m *mockBackend) QueryCalendar(_ context.Context, _ string, _ *extcaldav.CalendarQuery) ([]extcaldav.CalendarObject, error) {
	return m.queryResult, m.queryErr
}

func (m *mockBackend) PutCalendarObject(_ context.Context, path string, _ *ical.Calendar) (*extcaldav.CalendarObject, error) {
	m.lastPutPath = path
	return m.putResult, m.putErr
}

func (m *mockBackend) GetCalendarObject(_ context.Context, path string) (*extcaldav.CalendarObject, error) {
	m.lastGetPath = path
	return m.getResult, m.getErr
}

func (m *mockBackend) RemoveAll(_ context.Context, path string) error {
	m.lastRemovePath = path
	return m.removeErr
}

func TestDefaultClientOptions(t *testing.T) {
	opts := DefaultClientOptions()
	if opts.MaxConnsPerHost != 10 {
		t.Errorf("MaxConnsPerHost = %d, want 10", opts.MaxConnsPerHost)
	}
	if opts.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", opts.Timeout)
	}
}

func TestGetEventPath(t *testing.T) {
	c := &Client{}

	tests := []struct {
		name         string
		calendarPath string
		eventID      string
		want         string
	}{
		{"basic", "/calendars/user/cal1", "event123", "/calendars/user/cal1/event123.ics"},
		{"trailing slash on calendar", "/calendars/user/cal1/", "event123", "/calendars/user/cal1/event123.ics"},
		{"event already has .ics", "/calendars/user/cal1", "event123.ics", "/calendars/user/cal1/event123.ics"},
		{"trailing slash and .ics", "/calendars/user/cal1/", "event123.ics", "/calendars/user/cal1/event123.ics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.GetEventPath(tt.calendarPath, tt.eventID)
			if got != tt.want {
				t.Errorf("GetEventPath(%q, %q) = %q, want %q", tt.calendarPath, tt.eventID, got, tt.want)
			}
		})
	}
}

func TestDiscoverCalendarHomeSet_Success(t *testing.T) {
	mb := &mockBackend{
		principal: "/principals/user/",
		homeSet:   "/calendars/user/",
	}
	c := NewClientWithBackend(mb)

	homeSet, err := c.DiscoverCalendarHomeSet(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if homeSet != "/calendars/user/" {
		t.Errorf("homeSet = %q, want /calendars/user/", homeSet)
	}
}

func TestDiscoverCalendarHomeSet_CachedAfterFirstCall(t *testing.T) {
	mb := &mockBackend{
		principal: "/principals/user/",
		homeSet:   "/calendars/user/",
	}
	c := NewClientWithBackend(mb)

	// First call
	_, err := c.DiscoverCalendarHomeSet(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Change backend to return error - should still return cached value
	mb.principalErr = fmt.Errorf("should not be called")
	homeSet, err := c.DiscoverCalendarHomeSet(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}
	if homeSet != "/calendars/user/" {
		t.Errorf("cached homeSet = %q, want /calendars/user/", homeSet)
	}
}

func TestDiscoverCalendarHomeSet_PrincipalError(t *testing.T) {
	mb := &mockBackend{
		principalErr: fmt.Errorf("principal not found"),
	}
	c := NewClientWithBackend(mb)

	_, err := c.DiscoverCalendarHomeSet(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDiscoverCalendarHomeSet_HomeSetError(t *testing.T) {
	mb := &mockBackend{
		principal:  "/principals/user/",
		homeSetErr: fmt.Errorf("home set not found"),
	}
	c := NewClientWithBackend(mb)

	_, err := c.DiscoverCalendarHomeSet(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListCalendars_Success(t *testing.T) {
	mb := &mockBackend{
		principal: "/principals/user/",
		homeSet:   "/calendars/user/",
		calendars: []extcaldav.Calendar{
			{Path: "/cal/work", Name: "Work", Description: "Work calendar"},
			{Path: "/cal/personal", Name: "Personal"},
		},
	}
	c := NewClientWithBackend(mb)

	cals, err := c.ListCalendars(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cals) != 2 {
		t.Fatalf("expected 2 calendars, got %d", len(cals))
	}
	if cals[0].Name != "Work" {
		t.Errorf("first calendar name = %q, want Work", cals[0].Name)
	}
	if cals[0].Description != "Work calendar" {
		t.Errorf("first calendar description = %q", cals[0].Description)
	}
}

func TestListCalendars_DiscoverError(t *testing.T) {
	mb := &mockBackend{
		principalErr: fmt.Errorf("discovery failed"),
	}
	c := NewClientWithBackend(mb)

	_, err := c.ListCalendars(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListCalendars_FindCalendarsError(t *testing.T) {
	mb := &mockBackend{
		principal:  "/principals/user/",
		homeSet:    "/calendars/user/",
		findCalErr: fmt.Errorf("permission denied"),
	}
	c := NewClientWithBackend(mb)

	_, err := c.ListCalendars(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListCalendars_Empty(t *testing.T) {
	mb := &mockBackend{
		principal: "/principals/user/",
		homeSet:   "/calendars/user/",
		calendars: []extcaldav.Calendar{},
	}
	c := NewClientWithBackend(mb)

	cals, err := c.ListCalendars(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cals) != 0 {
		t.Errorf("expected 0 calendars, got %d", len(cals))
	}
}

// Helper to create a CalendarObject with a VEVENT
func makeCalendarObject(path, uid, title string, start, end time.Time) extcaldav.CalendarObject {
	vevent := ical.NewEvent()
	vevent.Props.SetText(ical.PropUID, uid)
	vevent.Props.SetText(ical.PropSummary, title)
	vevent.Props.SetDateTime(ical.PropDateTimeStart, start)
	vevent.Props.SetDateTime(ical.PropDateTimeEnd, end)

	cal := ical.NewCalendar()
	cal.Children = append(cal.Children, vevent.Component)

	return extcaldav.CalendarObject{
		Path: path,
		Data: cal,
	}
}

func TestSearchEvents_Success(t *testing.T) {
	start := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	mb := &mockBackend{
		queryResult: []extcaldav.CalendarObject{
			makeCalendarObject("/cal/event1.ics", "uid-1", "Meeting", start, end),
		},
	}
	c := NewClientWithBackend(mb)

	events, err := c.SearchEvents(context.Background(), "/cal", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "Meeting" {
		t.Errorf("title = %q, want Meeting", events[0].Title)
	}
	if events[0].ID != "uid-1" {
		t.Errorf("id = %q, want uid-1", events[0].ID)
	}
}

func TestSearchEvents_WithTimeRange(t *testing.T) {
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	mb := &mockBackend{
		queryResult: []extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	events, err := c.SearchEvents(context.Background(), "/cal", &start, &end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestSearchEvents_WithStartTimeOnly(t *testing.T) {
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	mb := &mockBackend{
		queryResult: []extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	_, err := c.SearchEvents(context.Background(), "/cal", &start, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchEvents_WithEndTimeOnly(t *testing.T) {
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	mb := &mockBackend{
		queryResult: []extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	_, err := c.SearchEvents(context.Background(), "/cal", nil, &end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchEvents_QueryError(t *testing.T) {
	mb := &mockBackend{
		queryErr: fmt.Errorf("query failed"),
	}
	c := NewClientWithBackend(mb)

	_, err := c.SearchEvents(context.Background(), "/cal", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSearchEvents_SkipsUnparseableObjects(t *testing.T) {
	start := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	// Object with no VEVENT child
	badCal := ical.NewCalendar()
	badObj := extcaldav.CalendarObject{Path: "/cal/bad.ics", Data: badCal}

	mb := &mockBackend{
		queryResult: []extcaldav.CalendarObject{
			badObj,
			makeCalendarObject("/cal/good.ics", "uid-good", "Good Event", start, end),
		},
	}
	c := NewClientWithBackend(mb)

	events, err := c.SearchEvents(context.Background(), "/cal", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event (bad one skipped), got %d", len(events))
	}
	if events[0].Title != "Good Event" {
		t.Errorf("title = %q, want Good Event", events[0].Title)
	}
}

func TestCreateEvent_Success(t *testing.T) {
	mb := &mockBackend{
		putResult: &extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	event := &Event{
		Title:     "New Meeting",
		StartTime: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
	}

	uid, err := c.CreateEvent(context.Background(), "/cal/work", event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid == "" {
		t.Fatal("expected non-empty UID")
	}
}

func TestCreateEvent_WithExistingID(t *testing.T) {
	mb := &mockBackend{
		putResult: &extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	event := &Event{
		ID:        "custom-uid-123",
		Title:     "Meeting",
		StartTime: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
	}

	uid, err := c.CreateEvent(context.Background(), "/cal/work", event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid != "custom-uid-123" {
		t.Errorf("uid = %q, want custom-uid-123", uid)
	}
	if mb.lastPutPath != "/cal/work/custom-uid-123.ics" {
		t.Errorf("put path = %q, want /cal/work/custom-uid-123.ics", mb.lastPutPath)
	}
}

func TestCreateEvent_WithOptionalFields(t *testing.T) {
	mb := &mockBackend{
		putResult: &extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	event := &Event{
		Title:       "Meeting",
		Description: "Important meeting",
		Location:    "Room A",
		StartTime:   time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
	}

	_, err := c.CreateEvent(context.Background(), "/cal/work", event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateEvent_WithAttendees(t *testing.T) {
	mb := &mockBackend{
		putResult: &extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	event := &Event{
		Title:     "Team Sync",
		StartTime: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
		Attendees: []Attendee{
			{Email: "alice@example.com", Name: "Alice", Role: "CHAIR", Status: "ACCEPTED"},
			{Email: "bob@example.com"},
		},
	}

	_, err := c.CreateEvent(context.Background(), "/cal/work", event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateEvent_PutError(t *testing.T) {
	mb := &mockBackend{
		putErr: fmt.Errorf("quota exceeded"),
	}
	c := NewClientWithBackend(mb)

	event := &Event{
		Title:     "Meeting",
		StartTime: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
	}

	_, err := c.CreateEvent(context.Background(), "/cal/work", event)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateEvent_Success(t *testing.T) {
	// Build an existing calendar object
	existingEvent := ical.NewEvent()
	existingEvent.Props.SetText(ical.PropUID, "uid-1")
	existingEvent.Props.SetText(ical.PropSummary, "Old Title")
	existingEvent.Props.SetText(ical.PropDescription, "Old Desc")
	existingEvent.Props.SetText(ical.PropLocation, "Old Location")

	existingCal := ical.NewCalendar()
	existingCal.Children = append(existingCal.Children, existingEvent.Component)

	mb := &mockBackend{
		getResult: &extcaldav.CalendarObject{Data: existingCal},
		putResult: &extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	newTitle := "New Title"
	update := &EventUpdate{Title: &newTitle}

	err := c.UpdateEvent(context.Background(), "/cal/event.ics", update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateEvent_ClearDescription(t *testing.T) {
	existingEvent := ical.NewEvent()
	existingEvent.Props.SetText(ical.PropUID, "uid-1")
	existingEvent.Props.SetText(ical.PropSummary, "Title")
	existingEvent.Props.SetText(ical.PropDescription, "Old Desc")

	existingCal := ical.NewCalendar()
	existingCal.Children = append(existingCal.Children, existingEvent.Component)

	mb := &mockBackend{
		getResult: &extcaldav.CalendarObject{Data: existingCal},
		putResult: &extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	emptyStr := ""
	update := &EventUpdate{Description: &emptyStr}

	err := c.UpdateEvent(context.Background(), "/cal/event.ics", update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateEvent_ClearTitleAndLocation(t *testing.T) {
	existingEvent := ical.NewEvent()
	existingEvent.Props.SetText(ical.PropUID, "uid-1")
	existingEvent.Props.SetText(ical.PropSummary, "Title")
	existingEvent.Props.SetText(ical.PropLocation, "Room A")

	existingCal := ical.NewCalendar()
	existingCal.Children = append(existingCal.Children, existingEvent.Component)

	mb := &mockBackend{
		getResult: &extcaldav.CalendarObject{Data: existingCal},
		putResult: &extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	emptyStr := ""
	update := &EventUpdate{Title: &emptyStr, Location: &emptyStr}

	err := c.UpdateEvent(context.Background(), "/cal/event.ics", update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateEvent_UpdateTimes(t *testing.T) {
	existingEvent := ical.NewEvent()
	existingEvent.Props.SetText(ical.PropUID, "uid-1")
	existingEvent.Props.SetText(ical.PropSummary, "Meeting")

	existingCal := ical.NewCalendar()
	existingCal.Children = append(existingCal.Children, existingEvent.Component)

	mb := &mockBackend{
		getResult: &extcaldav.CalendarObject{Data: existingCal},
		putResult: &extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	newStart := time.Date(2024, 2, 1, 10, 0, 0, 0, time.UTC)
	newEnd := time.Date(2024, 2, 1, 11, 0, 0, 0, time.UTC)
	update := &EventUpdate{StartTime: &newStart, EndTime: &newEnd}

	err := c.UpdateEvent(context.Background(), "/cal/event.ics", update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateEvent_GetError(t *testing.T) {
	mb := &mockBackend{
		getErr: fmt.Errorf("not found"),
	}
	c := NewClientWithBackend(mb)

	title := "New"
	err := c.UpdateEvent(context.Background(), "/cal/event.ics", &EventUpdate{Title: &title})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateEvent_NoVEVENT(t *testing.T) {
	// Calendar with no VEVENT child
	existingCal := ical.NewCalendar()

	mb := &mockBackend{
		getResult: &extcaldav.CalendarObject{Data: existingCal},
	}
	c := NewClientWithBackend(mb)

	title := "New"
	err := c.UpdateEvent(context.Background(), "/cal/event.ics", &EventUpdate{Title: &title})
	if err == nil {
		t.Fatal("expected error for missing VEVENT")
	}
}

func TestUpdateEvent_PutError(t *testing.T) {
	existingEvent := ical.NewEvent()
	existingEvent.Props.SetText(ical.PropUID, "uid-1")
	existingEvent.Props.SetText(ical.PropSummary, "Title")

	existingCal := ical.NewCalendar()
	existingCal.Children = append(existingCal.Children, existingEvent.Component)

	mb := &mockBackend{
		getResult: &extcaldav.CalendarObject{Data: existingCal},
		putErr:    fmt.Errorf("server error"),
	}
	c := NewClientWithBackend(mb)

	title := "New"
	err := c.UpdateEvent(context.Background(), "/cal/event.ics", &EventUpdate{Title: &title})
	if err == nil {
		t.Fatal("expected error from put")
	}
}

func TestDeleteEvent_Success(t *testing.T) {
	mb := &mockBackend{}
	c := NewClientWithBackend(mb)

	err := c.DeleteEvent(context.Background(), "/cal/event.ics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mb.lastRemovePath != "/cal/event.ics" {
		t.Errorf("remove path = %q, want /cal/event.ics", mb.lastRemovePath)
	}
}

func TestDeleteEvent_Error(t *testing.T) {
	mb := &mockBackend{
		removeErr: fmt.Errorf("permission denied"),
	}
	c := NewClientWithBackend(mb)

	err := c.DeleteEvent(context.Background(), "/cal/event.ics")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCalendarObject_FullEvent(t *testing.T) {
	start := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	vevent := ical.NewEvent()
	vevent.Props.SetText(ical.PropUID, "uid-1")
	vevent.Props.SetText(ical.PropSummary, "Meeting")
	vevent.Props.SetText(ical.PropDescription, "A description")
	vevent.Props.SetText(ical.PropLocation, "Room B")
	vevent.Props.SetDateTime(ical.PropDateTimeStart, start)
	vevent.Props.SetDateTime(ical.PropDateTimeEnd, end)
	vevent.Props.SetText(ical.PropRecurrenceRule, "FREQ=DAILY;COUNT=3")

	// Add attendee
	prop := ical.Prop{
		Name:   "ATTENDEE",
		Value:  "mailto:alice@example.com",
		Params: ical.Params{},
	}
	prop.Params["CN"] = []string{"Alice"}
	prop.Params["ROLE"] = []string{"REQ-PARTICIPANT"}
	prop.Params["PARTSTAT"] = []string{"ACCEPTED"}
	vevent.Props["ATTENDEE"] = append(vevent.Props["ATTENDEE"], prop)

	cal := ical.NewCalendar()
	cal.Children = append(cal.Children, vevent.Component)

	obj := &extcaldav.CalendarObject{
		Path: "/cal/event1.ics",
		Data: cal,
	}

	c := &Client{}
	event, err := c.parseCalendarObject(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.ID != "uid-1" {
		t.Errorf("ID = %q, want uid-1", event.ID)
	}
	if event.Title != "Meeting" {
		t.Errorf("Title = %q, want Meeting", event.Title)
	}
	if event.Description != "A description" {
		t.Errorf("Description = %q", event.Description)
	}
	if event.Location != "Room B" {
		t.Errorf("Location = %q", event.Location)
	}
	if event.Recurrence == "" {
		t.Error("Recurrence should not be empty")
	}
	if event.Path != "/cal/event1.ics" {
		t.Errorf("Path = %q", event.Path)
	}
	if len(event.Attendees) != 1 {
		t.Fatalf("expected 1 attendee, got %d", len(event.Attendees))
	}
	if event.Attendees[0].Email != "alice@example.com" {
		t.Errorf("attendee email = %q", event.Attendees[0].Email)
	}
	if event.Attendees[0].Name != "Alice" {
		t.Errorf("attendee name = %q", event.Attendees[0].Name)
	}
	if event.Attendees[0].Role != "REQ-PARTICIPANT" {
		t.Errorf("attendee role = %q", event.Attendees[0].Role)
	}
	if event.Attendees[0].Status != "ACCEPTED" {
		t.Errorf("attendee status = %q", event.Attendees[0].Status)
	}
}

func TestParseCalendarObject_NoVEVENT(t *testing.T) {
	cal := ical.NewCalendar()
	obj := &extcaldav.CalendarObject{Path: "/cal/bad.ics", Data: cal}

	c := &Client{}
	_, err := c.parseCalendarObject(obj)
	if err == nil {
		t.Fatal("expected error for missing VEVENT")
	}
}

func TestParseCalendarObject_MinimalEvent(t *testing.T) {
	vevent := ical.NewEvent()
	vevent.Props.SetText(ical.PropUID, "uid-min")

	cal := ical.NewCalendar()
	cal.Children = append(cal.Children, vevent.Component)
	obj := &extcaldav.CalendarObject{Path: "/cal/min.ics", Data: cal}

	c := &Client{}
	event, err := c.parseCalendarObject(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ID != "uid-min" {
		t.Errorf("ID = %q, want uid-min", event.ID)
	}
	if event.Timezone != "UTC" {
		t.Errorf("Timezone = %q, want UTC", event.Timezone)
	}
}

func TestParseCalendarObject_AttendeeWithoutEmail(t *testing.T) {
	vevent := ical.NewEvent()
	vevent.Props.SetText(ical.PropUID, "uid-1")

	// Attendee with empty value (no mailto:)
	prop := ical.Prop{
		Name:   "ATTENDEE",
		Value:  "",
		Params: ical.Params{},
	}
	vevent.Props["ATTENDEE"] = append(vevent.Props["ATTENDEE"], prop)

	cal := ical.NewCalendar()
	cal.Children = append(cal.Children, vevent.Component)
	obj := &extcaldav.CalendarObject{Path: "/cal/event.ics", Data: cal}

	c := &Client{}
	event, err := c.parseCalendarObject(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Attendee without email should be skipped
	if len(event.Attendees) != 0 {
		t.Errorf("expected 0 attendees, got %d", len(event.Attendees))
	}
}

func TestCreateEvent_TrailingSlashCalendar(t *testing.T) {
	mb := &mockBackend{
		putResult: &extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	event := &Event{
		ID:        "test-uid",
		Title:     "Meeting",
		StartTime: time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
	}

	_, err := c.CreateEvent(context.Background(), "/cal/work/", event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mb.lastPutPath != "/cal/work/test-uid.ics" {
		t.Errorf("put path = %q, want /cal/work/test-uid.ics", mb.lastPutPath)
	}
}

func TestUpdateEvent_SetNonEmptyDescriptionAndLocation(t *testing.T) {
	existingEvent := ical.NewEvent()
	existingEvent.Props.SetText(ical.PropUID, "uid-1")
	existingEvent.Props.SetText(ical.PropSummary, "Old Title")

	existingCal := ical.NewCalendar()
	existingCal.Children = append(existingCal.Children, existingEvent.Component)

	mb := &mockBackend{
		getResult: &extcaldav.CalendarObject{Data: existingCal},
		putResult: &extcaldav.CalendarObject{},
	}
	c := NewClientWithBackend(mb)

	desc := "New Desc"
	loc := "Room C"
	update := &EventUpdate{Description: &desc, Location: &loc}

	err := c.UpdateEvent(context.Background(), "/cal/event.ics", update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
