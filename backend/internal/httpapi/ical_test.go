package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestCalendarFeed: a house's link serves the village's events and work
// parties to a calendar that cannot sign in — times on the village's clock,
// series in RFC 5545, lines folded — and nothing else: no birthday, no away
// notice. A new link kills the old one, and a key cannot ask for one.
func TestCalendarFeed(t *testing.T) {
	t.Setenv("TZ", "Europe/Ljubljana")
	srv, _, _, volk := newVillage(t)
	long := "Delovna akcija na skupnem travniku: košnja, grabljenje in spravilo sena pred dežjem"
	for _, ev := range []map[string]any{
		{"title": "Košnja, travnik", "kind": "work", "starts_at": "2026-09-20T08:00", "ends_at": "2026-09-20T12:00", "place": "Pri vodnjaku; levo", "notes": "Prinesite kose.\nIn vodo."},
		{"title": "Tržnica", "starts_at": "2026-09-05T09:00", "recur": "weekly"},
		{"title": "Obračun", "starts_at": "2026-08-31", "recur": "monthly"},
		{"title": long, "starts_at": "2026-09-12T10:00"},
		{"title": "Ana", "kind": "birthday", "starts_at": "1990-05-01"},
	} {
		code, _, _ := volk.do("POST", "/api/events", ev)
		volk.must(201, code, "event "+ev["title"].(string))
	}
	code, _, _ := volk.do("POST", "/api/away", map[string]any{"from_date": "2026-09-10", "to_date": "2026-09-20", "notes": "ključ pod predpražnikom"})
	volk.must(201, code, "away")

	code, f, _ := volk.do("GET", "/api/calendar/feed", nil)
	volk.must(201, code, "the first ask makes the link")
	tok := f["token"].(string)
	if _, again, _ := volk.do("GET", "/api/calendar/feed", nil); again["token"] != tok {
		t.Fatal("the link changed on a second ask")
	}

	get := func(path string) (int, string, string) {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		return rec.Code, rec.Header().Get("Content-Type"), rec.Body.String()
	}
	code, ctype, body := get("/api/ical/" + tok + ".ics")
	if code != 200 || !strings.HasPrefix(ctype, "text/calendar") {
		t.Fatalf("feed without a session: %d %s", code, ctype)
	}
	for _, l := range strings.Split(strings.TrimSuffix(body, "\r\n"), "\r\n") {
		if len(l) > 75 {
			t.Fatalf("a line longer than 75 octets: %q", l)
		}
	}
	flat := strings.ReplaceAll(body, "\r\n ", "")
	for _, want := range []string{
		"BEGIN:VCALENDAR", "X-WR-TIMEZONE:Europe/Ljubljana", "BEGIN:VTIMEZONE", "TZID:Europe/Ljubljana", "BEGIN:DAYLIGHT", "TZOFFSETTO:+0200",
		"SUMMARY:🤝 Košnja\\, travnik", "LOCATION:Pri vodnjaku\\; levo", "Prinesite kose.\\nIn vodo.",
		"DTSTART;TZID=Europe/Ljubljana:20260920T080000", "DTEND;TZID=Europe/Ljubljana:20260920T120000",
		"SUMMARY:Tržnica", "RRULE:FREQ=WEEKLY",
		"DTSTART;VALUE=DATE:20260831", "DTEND;VALUE=DATE:20260901", "RRULE:FREQ=MONTHLY;BYMONTHDAY=31,-1;BYSETPOS=1",
		"SUMMARY:" + icsText(long), "#/go/event/",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("feed lacks %q:\n%s", want, flat)
		}
	}
	for _, never := range []string{"SUMMARY:Ana", "ključ pod predpražnikom", "19900501"} {
		if strings.Contains(flat, never) {
			t.Fatalf("feed carries %q", never)
		}
	}

	code, f2, _ := volk.do("POST", "/api/calendar/feed", nil)
	volk.must(201, code, "a new link")
	if code, _, _ := get("/api/ical/" + tok + ".ics"); code != 404 {
		t.Fatalf("the old link after a new one: %d", code)
	}
	if code, _, _ := get("/api/ical/" + f2["token"].(string) + ".ics"); code != 200 {
		t.Fatalf("the new link: %d", code)
	}
	if code, _, _ := get("/api/ical/nothing.ics"); code != 404 {
		t.Fatalf("an unknown link: %d", code)
	}
	key, _ := mintKey(t, volk, "agent")
	agent := &client{t: t, h: srv.Handler(), token: key}
	code, _, _ = agent.do("GET", "/api/calendar/feed", nil)
	agent.must(403, code, "a key asks for the link")

	if r := rrule("yearly", time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)); r != "FREQ=YEARLY;BYMONTH=2;BYMONTHDAY=29,-1;BYSETPOS=1" {
		t.Fatalf("29 February: %s", r)
	}
}

// TestFeedWithoutZone: a village with no TZ sends floating times, read in
// each subscriber's own zone, and no zone block.
func TestFeedWithoutZone(t *testing.T) {
	t.Setenv("TZ", "")
	srv, _, _, volk := newVillage(t)
	volk.do("POST", "/api/events", map[string]any{"title": "Kino", "starts_at": "2026-09-12T20:00"})
	_, f, _ := volk.do("GET", "/api/calendar/feed", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/ical/"+f["token"].(string)+".ics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "DTSTART:20260912T200000\r\n") || strings.Contains(body, "VTIMEZONE") {
		t.Fatalf("floating feed:\n%s", body)
	}
}
