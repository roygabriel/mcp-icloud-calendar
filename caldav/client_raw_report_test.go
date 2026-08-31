package caldav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-webdav"
)

// TestSearchEvents_RawReport_ParsesEventWithAlarm is a regression test for a
// confirmed iCloud CalDAV server bug: when a calendar-query REPORT's
// <calendar-data> element requests specific components/properties (via a
// <comp> child, whether empty, <allprop/>, or <allcomp/>), iCloud truncates
// the returned calendar-data for any matched VEVENT that contains a VALARM,
// dropping every VEVENT-level property while keeping only the VALARM. This
// was confirmed against a real iCloud account and does not reproduce with a
// bare <calendar-data></calendar-data> (no children at all).
//
// SearchEvents works around this by issuing its own minimal REPORT request
// (queryCalendarDataRaw) instead of using go-webdav's caldav.Client.QueryCalendar,
// whose CalendarCompRequest always encodes at least one <comp> child once
// CompRequest.Name is set. This test exercises that raw path end-to-end
// against a fake server, and would fail if a future change reintroduced a
// <comp>/<allprop>/<allcomp> element into the request.
func TestSearchEvents_RawReport_ParsesEventWithAlarm(t *testing.T) {
	const icalBody = "BEGIN:VCALENDAR\r\n" +
		"VERSION:2.0\r\n" +
		"PRODID:-//test//EN\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:test-uid-1\r\n" +
		"SUMMARY:Anniversary\r\n" +
		"DTSTART:20260101T090000Z\r\n" +
		"DTEND:20260101T100000Z\r\n" +
		"BEGIN:VALARM\r\n" +
		"ACTION:DISPLAY\r\n" +
		"DESCRIPTION:Reminder\r\n" +
		"TRIGGER:-PT15M\r\n" +
		"UID:alarm-uid-1\r\n" +
		"END:VALARM\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	var capturedRequestBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "REPORT" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		b, _ := io.ReadAll(r.Body)
		capturedRequestBody = string(b)

		// Simulate the real iCloud server bug: if the request's
		// calendar-data element contains a <comp> child, truncate the
		// response to only the VALARM. A bare <calendar-data/> (what the
		// fix must send) gets the full body.
		responseICal := icalBody
		if strings.Contains(capturedRequestBody, "<calendar-data") &&
			strings.Contains(capturedRequestBody, "<comp") &&
			strings.Contains(capturedRequestBody, "calendar-data>") &&
			!strings.Contains(capturedRequestBody, "<calendar-data></calendar-data>") &&
			!strings.Contains(capturedRequestBody, "<calendar-data/>") {
			responseICal = "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nBEGIN:VALARM\r\n" +
				"ACTION:DISPLAY\r\nDESCRIPTION:Reminder\r\nTRIGGER:-PT15M\r\n" +
				"UID:alarm-uid-1\r\nEND:VALARM\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
		}

		multistatus := `<?xml version="1.0" encoding="UTF-8"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/cal/test.ics</href>
    <propstat>
      <prop>
        <getetag xmlns="DAV:">"etag1"</getetag>
        <calendar-data xmlns="urn:ietf:params:xml:ns:caldav"><![CDATA[` + responseICal + `]]></calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		io.WriteString(w, multistatus)
	}))
	defer srv.Close()

	c := &Client{
		httpClient: webdav.HTTPClientWithBasicAuth(&http.Client{}, "user", "pass"),
		baseURL:    srv.URL,
	}

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	events, err := c.SearchEvents(context.Background(), "/cal/", &start, &end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	got := events[0]
	if got.Title != "Anniversary" {
		t.Errorf("title = %q, want %q (event data was likely truncated - request body: %s)", got.Title, "Anniversary", capturedRequestBody)
	}
	if got.ID != "test-uid-1" {
		t.Errorf("id = %q, want %q", got.ID, "test-uid-1")
	}
	if got.StartTime.IsZero() {
		t.Errorf("startTime is zero, want 2026-01-01T09:00:00Z")
	}
	if got.EndTime.IsZero() {
		t.Errorf("endTime is zero, want 2026-01-01T10:00:00Z")
	}

	// Confirm the actual request we sent avoided the buggy shape: no <comp>
	// element inside <calendar-data>.
	calDataStart := strings.Index(capturedRequestBody, "<C:calendar-data")
	calDataEnd := strings.Index(capturedRequestBody, "</C:calendar-data>")
	if calDataStart == -1 {
		t.Fatalf("request did not contain a calendar-data element: %s", capturedRequestBody)
	}
	var calDataElement string
	if calDataEnd == -1 {
		// Self-closing form, e.g. <C:calendar-data/>.
		selfCloseEnd := strings.Index(capturedRequestBody[calDataStart:], "/>")
		if selfCloseEnd == -1 {
			t.Fatalf("could not find end of calendar-data element: %s", capturedRequestBody)
		}
		calDataElement = capturedRequestBody[calDataStart : calDataStart+selfCloseEnd]
	} else {
		calDataElement = capturedRequestBody[calDataStart:calDataEnd]
	}
	if strings.Contains(calDataElement, "<C:comp") || strings.Contains(calDataElement, "<comp") {
		t.Errorf("request's calendar-data element contains a <comp> child, which triggers the iCloud truncation bug: %s", calDataElement)
	}
}
