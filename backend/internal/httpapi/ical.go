package httpapi

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// The village calendar as a feed (owner's ask 2026-11-03; CLAUDE.md § The
// calendar as a feed, docs/design-calendar-feed.md). A house's phone
// calendar, Google's or Apple's, subscribes to one link and keeps the
// village's events and work parties beside its own, re-reading them by
// itself. A subscription cannot sign in, so the link is the credential: a
// secret per house, any of its people may open it, and ♻ makes a new one
// that kills the old in every calendar that has it. It carries events and
// work parties only — never an away notice, a birthday or the sky (the
// owner's pick) — and no headcount and no answers, as an iPhone's lock
// screen might show a reminder of it.

// GET /api/calendar/feed: the house's secret, or none yet; a read writes
// nothing, so the panel POSTs the first one.
func (s *Server) getFeed(w http.ResponseWriter, r *http.Request) {
	row, err := s.st.One(r.Context(), `SELECT token FROM calendar_feeds WHERE house_id=?`, houseFrom(r).ID)
	if err != nil {
		fail(w, err)
		return
	}
	if row == nil {
		writeJSON(w, 200, map[string]any{})
		return
	}
	writeJSON(w, 200, map[string]any{"token": row["token"]})
}

// POST /api/calendar/feed: the house's first secret, or a new one; the old stops.
func (s *Server) newFeed(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	tok := randomToken(24)
	if _, err := s.st.Exec(r.Context(), `INSERT INTO calendar_feeds(house_id, token) VALUES (?,?)
		ON CONFLICT(house_id) DO UPDATE SET token=excluded.token, created_at=datetime('now')`, h.ID, tok); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"token": tok})
}

// GET /api/ical/{name}: the feed itself, public by its secret. `name` is the
// token with ".ics" after it, the ending a calendar app expects.
func (s *Server) ical(w http.ResponseWriter, r *http.Request) {
	tok := strings.TrimSuffix(r.PathValue("name"), ".ics")
	if tok == "" {
		writeErr(w, 404, "no such calendar")
		return
	}
	feed, err := s.st.One(r.Context(), `SELECT house_id FROM calendar_feeds WHERE token=?`, tok)
	if err != nil {
		fail(w, err)
		return
	}
	if feed == nil {
		writeErr(w, 404, "no such calendar")
		return
	}
	// The same window the calendar room reads: sixty days back, every series.
	since := s.now().AddDate(0, 0, -60).Format("2006-01-02")
	rows, err := s.st.Rows(r.Context(), `SELECT e.id, e.title, e.kind, e.starts_at, e.ends_at, e.place, e.notes, e.recur,
		e.time_version, e.created_at, coalesce(e.edited_at, e.created_at) AS modified, h.name AS house_name, h.crest AS house_crest
		FROM events e JOIN houses h ON h.id=e.house_id
		WHERE e.kind IN ('event','work') AND (e.starts_at >= ? OR e.recur != '') ORDER BY e.starts_at LIMIT 500`, since)
	if err != nil {
		fail(w, err)
		return
	}
	zone, loc := feedZone()
	base := strings.TrimSuffix(s.cfg.PublicURL, "/")
	host := "porta-pagi"
	if u, err := url.Parse(base); err == nil && u.Host != "" {
		host = u.Host
	}
	now := s.now().UTC()

	var b strings.Builder
	line := func(l string) { icsLine(&b, l) }
	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//Porta Pagi//" + icsText(s.cfg.VillageName) + "//EN")
	line("CALSCALE:GREGORIAN")
	line("METHOD:PUBLISH")
	line("X-WR-CALNAME:" + icsText(s.cfg.VillageName))
	// A hint only: Google reads a subscription when it chooses, every few hours.
	line("REFRESH-INTERVAL;VALUE=DURATION:PT1H")
	line("X-PUBLISHED-TTL:PT1H")
	if zone != "" {
		line("X-WR-TIMEZONE:" + zone)
		from := now.AddDate(-1, 0, 0)
		for _, e := range rows {
			if t, ok := wallClock(str(e, "starts_at"), loc); ok && t.Before(from) {
				from = t
			}
		}
		vtimezone(line, zone, loc, from.AddDate(0, 0, -1), now.AddDate(10, 0, 0))
	}
	for _, e := range rows {
		start, ok := wallClock(str(e, "starts_at"), loc)
		if !ok {
			continue // a start the page could not draw either
		}
		allDay := len(str(e, "starts_at")) <= 10
		made := str(e, "created_at")
		line("BEGIN:VEVENT")
		line(fmt.Sprintf("UID:event-%d-%s@%s", e["id"], strings.NewReplacer("-", "", ":", "", " ", "T").Replace(made), host))
		line("DTSTAMP:" + now.Format("20060102T150405Z"))
		if m, err := time.Parse("2006-01-02 15:04:05", str(e, "modified")); err == nil {
			line("LAST-MODIFIED:" + m.Format("20060102T150405Z"))
		}
		line("SEQUENCE:" + strconv.FormatInt(e["time_version"].(int64), 10))
		if allDay {
			line("DTSTART;VALUE=DATE:" + start.Format("20060102"))
			last := start
			if end, ok := wallClock(str(e, "ends_at"), loc); ok && !end.Before(start) {
				last = end
			}
			// An all-day end is the day after, exclusive.
			line("DTEND;VALUE=DATE:" + time.Date(last.Year(), last.Month(), last.Day()+1, 0, 0, 0, 0, loc).Format("20060102"))
		} else {
			line("DTSTART" + icsTime(start, zone))
			if end, ok := wallClock(str(e, "ends_at"), loc); ok && !end.Before(start) {
				if len(str(e, "ends_at")) <= 10 {
					// An end with no hour lasts to the end of its day.
					end = time.Date(end.Year(), end.Month(), end.Day()+1, 0, 0, 0, 0, loc)
				}
				line("DTEND" + icsTime(end, zone))
			}
			// No end is no end: the page draws none either.
		}
		if rule := rrule(str(e, "recur"), start); rule != "" {
			line("RRULE:" + rule)
		}
		title := str(e, "title")
		if str(e, "kind") == "work" {
			title = "🤝 " + title
		}
		line("SUMMARY:" + icsText(title))
		if p := str(e, "place"); p != "" {
			line("LOCATION:" + icsText(p))
		}
		link := fmt.Sprintf("%s/#/go/event/%d/%s", base, e["id"], firstN(made, 10))
		desc := strings.TrimSpace(strings.Join([]string{str(e, "notes"), str(e, "house_crest") + " " + str(e, "house_name"), link}, "\n\n"))
		line("DESCRIPTION:" + icsText(desc))
		line("URL:" + link)
		line("END:VEVENT")
	}
	line("END:VCALENDAR")
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Disposition", `inline; filename="calendar.ics"`)
	w.Write([]byte(b.String()))
}

// feedZone: the village's clock, the TZ the compose file names (main.go).
// Without one the times go out floating, read in each subscriber's own zone,
// which is what the page does with them too.
func feedZone() (string, *time.Location) {
	name := strings.TrimPrefix(os.Getenv("TZ"), ":")
	if name == "" {
		return "", time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return "", time.UTC
	}
	return name, loc
}

// wallClock reads a time a villager typed: a date, or a date and a minute.
func wallClock(v string, loc *time.Location) (time.Time, bool) {
	for _, f := range []string{"2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(f, v, loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func icsTime(t time.Time, zone string) string {
	if zone == "" {
		return ":" + t.Format("20060102T150405")
	}
	return ";TZID=" + zone + ":" + t.Format("20060102T150405")
}

// rrule says the page's repeat rule (recur.go) in RFC 5545: a monthly series
// begun on a day some months lack lands on their last day, and so does a
// yearly one begun on 29 February — the earliest of that day and the last.
func rrule(recur string, start time.Time) string {
	switch recur {
	case "weekly":
		return "FREQ=WEEKLY"
	case "monthly":
		if start.Day() > 28 {
			return fmt.Sprintf("FREQ=MONTHLY;BYMONTHDAY=%d,-1;BYSETPOS=1", start.Day())
		}
		return "FREQ=MONTHLY"
	case "yearly":
		if start.Month() == time.February && start.Day() == 29 {
			return "FREQ=YEARLY;BYMONTH=2;BYMONTHDAY=29,-1;BYSETPOS=1"
		}
		return "FREQ=YEARLY"
	}
	return ""
}

// vtimezone writes the zone's changes between from and to, one onset each,
// so a calendar that does not know the zone by name still places the hours;
// Google and Apple know it by name anyway.
func vtimezone(line func(string), zone string, loc *time.Location, from, to time.Time) {
	off := func(sec int) string {
		sign := "+"
		if sec < 0 {
			sign, sec = "-", -sec
		}
		return fmt.Sprintf("%s%02d%02d", sign, sec/3600, sec%3600/60)
	}
	line("BEGIN:VTIMEZONE")
	line("TZID:" + zone)
	cur := from.In(loc)
	changed := false
	for {
		_, end := cur.ZoneBounds()
		if end.IsZero() || end.After(to) {
			break
		}
		_, before := cur.Zone()
		next := end.In(loc)
		name, after := next.Zone()
		kind := "STANDARD"
		if next.IsDST() {
			kind = "DAYLIGHT"
		}
		line("BEGIN:" + kind)
		line("DTSTART:" + end.In(time.FixedZone("", before)).Format("20060102T150405"))
		line("TZOFFSETFROM:" + off(before))
		line("TZOFFSETTO:" + off(after))
		line("TZNAME:" + name)
		line("END:" + kind)
		cur, changed = next, true
	}
	if !changed {
		name, o := cur.Zone()
		line("BEGIN:STANDARD")
		line("DTSTART:19700101T000000")
		line("TZOFFSETFROM:" + off(o))
		line("TZOFFSETTO:" + off(o))
		line("TZNAME:" + name)
		line("END:STANDARD")
	}
	line("END:VTIMEZONE")
}

// icsText escapes a value: a backslash, a semicolon, a comma and a new line.
func icsText(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`, "\r", `\n`).Replace(s)
}

// icsLine folds a content line at 75 octets, never inside a character, each
// continuation starting with the one space that counts against it.
func icsLine(b *strings.Builder, l string) {
	limit := 75
	for len(l) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(l[cut]) {
			cut--
		}
		b.WriteString(l[:cut])
		b.WriteString("\r\n ")
		l = l[cut:]
		limit = 74
	}
	b.WriteString(l)
	b.WriteString("\r\n")
}
