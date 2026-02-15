package caldav

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"
)

const (
	// DefaultMaxResponseBytes is the default maximum response body size (10 MB).
	DefaultMaxResponseBytes int64 = 10 * 1024 * 1024
)

// ErrResponseTooLarge is returned when a response body exceeds the configured size limit.
var ErrResponseTooLarge = errors.New("response body exceeds size limit")

const (
	iCloudBaseURL = "https://caldav.icloud.com"
	timeout       = 30 * time.Second
)

// Client wraps the CalDAV client with iCloud-specific functionality
type Client struct {
	backend         backend
	calendarHomeSet string
	homeSetOnce     sync.Once
	homeSetErr      error
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
	ID          string     `json:"id"`
	Path        string     `json:"path"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Location    string     `json:"location"`
	StartTime   time.Time  `json:"startTime"`
	EndTime     time.Time  `json:"endTime"`
	Recurrence  string     `json:"recurrence,omitempty"`
	Timezone    string     `json:"timezone"`
	Attendees   []Attendee `json:"attendees,omitempty"`
}

// EventUpdate represents fields to update on an event.
// nil pointer = don't change, non-nil empty string = clear field.
type EventUpdate struct {
	Title       *string
	Description *string
	Location    *string
	StartTime   *time.Time
	EndTime     *time.Time
}

// ClientOptions configures the CalDAV client.
type ClientOptions struct {
	MaxConnsPerHost  int
	Timeout          time.Duration
	TLSCertFile      string
	TLSKeyFile       string
	TLSCAFile        string
	MaxResponseBytes int64
}

// DefaultClientOptions returns sensible defaults.
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		MaxConnsPerHost: 10,
		Timeout:         timeout,
	}
}

// NewClient creates a new CalDAV client configured for iCloud
func NewClient(email, password string, opts ...ClientOptions) (*Client, error) {
	opt := DefaultClientOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Timeout == 0 {
		opt.Timeout = timeout
	}
	if opt.MaxConnsPerHost == 0 {
		opt.MaxConnsPerHost = 10
	}

	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   opt.MaxConnsPerHost,
		MaxConnsPerHost:       opt.MaxConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		DisableCompression:    false,
	}

	// Configure mTLS if cert/key files are provided
	if opt.TLSCertFile != "" && opt.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(opt.TLSCertFile, opt.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS client certificate: %w", err)
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		if opt.TLSCAFile != "" {
			caCert, err := os.ReadFile(opt.TLSCAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read TLS CA file: %w", err)
			}
			caPool := x509.NewCertPool()
			if !caPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse TLS CA certificate")
			}
			tlsConfig.RootCAs = caPool
		}
		transport.TLSClientConfig = tlsConfig
	}

	maxBody := opt.MaxResponseBytes
	if maxBody <= 0 {
		maxBody = DefaultMaxResponseBytes
	}

	httpClient := &http.Client{
		Timeout:   opt.Timeout,
		Transport: &limitedTransport{inner: transport, maxBytes: maxBody},
	}

	// Create basic auth HTTP client
	authClient := webdav.HTTPClientWithBasicAuth(httpClient, email, password)

	// Create CalDAV client
	caldavClient, err := caldav.NewClient(authClient, iCloudBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create CalDAV client: %w", err)
	}

	return &Client{
		backend: caldavClient,
	}, nil
}

// NewClientWithBackend creates a Client with a custom backend for testing.
func NewClientWithBackend(b backend) *Client {
	return &Client{backend: b}
}

// DiscoverCalendarHomeSet discovers the user's calendar home set.
// The result is cached after the first successful call using sync.Once.
func (c *Client) DiscoverCalendarHomeSet(ctx context.Context) (string, error) {
	c.homeSetOnce.Do(func() {
		principal, err := c.backend.FindCurrentUserPrincipal(ctx)
		if err != nil {
			c.homeSetErr = fmt.Errorf("failed to find user principal: %w", err)
			return
		}

		homeSet, err := c.backend.FindCalendarHomeSet(ctx, principal)
		if err != nil {
			c.homeSetErr = fmt.Errorf("failed to find calendar home set: %w", err)
			return
		}

		c.calendarHomeSet = homeSet
	})
	return c.calendarHomeSet, c.homeSetErr
}

// ListCalendars lists all available calendars
func (c *Client) ListCalendars(ctx context.Context) ([]Calendar, error) {
	homeSet, err := c.DiscoverCalendarHomeSet(ctx)
	if err != nil {
		return nil, err
	}

	caldavCals, err := c.backend.FindCalendars(ctx, homeSet)
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

	calendarObjects, err := c.backend.QueryCalendar(ctx, calendarPath, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query calendar: %w", err)
	}

	events := make([]Event, 0, len(calendarObjects))
	for _, obj := range calendarObjects {
		event, err := c.parseCalendarObject(&obj)
		if err != nil {
			slog.Warn("skipping unparseable calendar object", "path", obj.Path, "error", err)
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
		uid = fmt.Sprintf("%s@mcp-icloud-calendar", uuid.New().String())
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

	// Add attendees
	for _, a := range event.Attendees {
		prop := ical.Prop{
			Name:   "ATTENDEE",
			Value:  "mailto:" + a.Email,
			Params: ical.Params{},
		}
		if a.Name != "" {
			prop.Params["CN"] = []string{a.Name}
		}
		if a.Role != "" {
			prop.Params["ROLE"] = []string{a.Role}
		} else {
			prop.Params["ROLE"] = []string{"REQ-PARTICIPANT"}
		}
		if a.Status != "" {
			prop.Params["PARTSTAT"] = []string{a.Status}
		} else {
			prop.Params["PARTSTAT"] = []string{"NEEDS-ACTION"}
		}
		vevent.Props["ATTENDEE"] = append(vevent.Props["ATTENDEE"], prop)
	}

	cal.Children = append(cal.Children, vevent.Component)

	// Create the event path
	eventPath := fmt.Sprintf("%s/%s.ics", strings.TrimSuffix(calendarPath, "/"), uid)

	// Put the calendar object
	_, err := c.backend.PutCalendarObject(ctx, eventPath, cal)
	if err != nil {
		return "", fmt.Errorf("failed to create event: %w", err)
	}

	return uid, nil
}

// UpdateEvent updates an existing event using pointer fields.
// nil pointer = don't change, non-nil empty string = clear the field.
func (c *Client) UpdateEvent(ctx context.Context, eventPath string, update *EventUpdate) error {
	// Get the existing event
	existingObj, err := c.backend.GetCalendarObject(ctx, eventPath)
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

	// Update properties: nil = skip, empty string = delete property
	if update.Title != nil {
		if *update.Title == "" {
			delete(vevent.Props, ical.PropSummary)
		} else {
			vevent.Props.SetText(ical.PropSummary, *update.Title)
		}
	}

	if update.Description != nil {
		if *update.Description == "" {
			delete(vevent.Props, ical.PropDescription)
		} else {
			vevent.Props.SetText(ical.PropDescription, *update.Description)
		}
	}

	if update.Location != nil {
		if *update.Location == "" {
			delete(vevent.Props, ical.PropLocation)
		} else {
			vevent.Props.SetText(ical.PropLocation, *update.Location)
		}
	}

	if update.StartTime != nil {
		vevent.Props.SetDateTime(ical.PropDateTimeStart, *update.StartTime)
	}

	if update.EndTime != nil {
		vevent.Props.SetDateTime(ical.PropDateTimeEnd, *update.EndTime)
	}

	// Update timestamp
	vevent.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())

	// Put the updated calendar object
	_, err = c.backend.PutCalendarObject(ctx, eventPath, existingObj.Data)
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

// DeleteEvent deletes an event by its path
func (c *Client) DeleteEvent(ctx context.Context, eventPath string) error {
	err := c.backend.RemoveAll(ctx, eventPath)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	return nil
}

// GetEventPath constructs the full path to an event
func (c *Client) GetEventPath(calendarPath, eventID string) string {
	calPath := strings.TrimSuffix(calendarPath, "/")
	if !strings.HasSuffix(eventID, ".ics") {
		eventID += ".ics"
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

	// Extract attendees
	for _, prop := range vevent.Props["ATTENDEE"] {
		attendee := Attendee{}
		val := prop.Value
		if strings.HasPrefix(val, "mailto:") {
			attendee.Email = strings.TrimPrefix(val, "mailto:")
		}
		if cn := prop.Params.Get("CN"); cn != "" {
			attendee.Name = cn
		}
		if role := prop.Params.Get("ROLE"); role != "" {
			attendee.Role = role
		}
		if status := prop.Params.Get("PARTSTAT"); status != "" {
			attendee.Status = status
		}
		if attendee.Email != "" {
			event.Attendees = append(event.Attendees, attendee)
		}
	}

	if event.Timezone == "" {
		event.Timezone = "UTC"
	}

	return event, nil
}

// limitedTransport wraps an http.RoundTripper and enforces a maximum response body size.
type limitedTransport struct {
	inner    http.RoundTripper
	maxBytes int64
}

// RoundTrip executes the request and wraps the response body with a size-limited reader.
func (t *limitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Body = &limitedReadCloser{
		rc:        resp.Body,
		remaining: t.maxBytes,
	}
	return resp, nil
}

// limitedReadCloser wraps a ReadCloser and returns an error if the read exceeds the limit.
type limitedReadCloser struct {
	rc        io.ReadCloser
	remaining int64
}

// Read reads up to len(p) bytes, returning ErrResponseTooLarge if the body exceeds the limit.
func (l *limitedReadCloser) Read(p []byte) (int, error) {
	if l.remaining < 0 {
		return 0, ErrResponseTooLarge
	}
	if l.remaining == 0 {
		// Check if there's more data beyond the limit.
		var probe [1]byte
		n, err := l.rc.Read(probe[:])
		if n > 0 {
			l.remaining = -1
			return 0, ErrResponseTooLarge
		}
		return 0, err
	}
	if int64(len(p)) > l.remaining {
		p = p[:l.remaining]
	}
	n, err := l.rc.Read(p)
	l.remaining -= int64(n)
	return n, err
}

// Close closes the underlying ReadCloser.
func (l *limitedReadCloser) Close() error {
	return l.rc.Close()
}
