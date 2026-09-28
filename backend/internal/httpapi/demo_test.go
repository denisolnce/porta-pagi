package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dennislapchenko/porta-pagi/backend/internal/config"
	"github.com/dennislapchenko/porta-pagi/backend/internal/store"
)

// TestDemoSeed: the demo village lands whole on an empty ground, dated off
// the day it runs, every house it names defined, and never on top of a
// village that exists.
func TestDemoSeed(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := New(st, config.Config{Languages: []string{"pt", "en"}, VillageName: "Sobreiro Torto"})
	// Lisbon: a clock change sits between September and the winter anchors,
	// which is how an hour became a day once.
	lisbon, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, lisbon)
	if err := srv.SeedDemo(now); err != nil {
		t.Fatal(err)
	}
	if err := srv.SeedDemo(now); err == nil {
		t.Fatal("the demo seeded on top of itself")
	}
	count := func(table string) int64 {
		row, err := st.One(t.Context(), `SELECT count(*) AS n FROM `+table)
		if err != nil {
			t.Fatal(err)
		}
		return row["n"].(int64)
	}
	for table, min := range map[string]int64{"houses": 10, "posts": 20, "events": 12, "event_signups": 40, "comments": 8, "runs": 2, "needs": 5, "offers": 3, "away": 3, "tools": 8, "wishes": 2, "projects": 3, "project_tasks": 8, "camp_takings": 3, "contacts": 6, "codex_sections": 5, "codex_texts": 10, "map_files": 2} {
		if n := count(table); n < min {
			t.Errorf("%s: %d rows, expected at least %d", table, n, min)
		}
	}
	// A past event and a future one, off the seed's day; a wall clock, not UTC.
	if row, _ := st.One(t.Context(), `SELECT min(starts_at) AS a, max(starts_at) AS b FROM events`); row == nil || row["a"].(string) > "2026-09-23" || row["b"].(string) < "2026-09-24" || !strings.Contains(row["a"].(string), "T") {
		t.Fatalf("events are not spread around the seed's day: %v", row)
	}
	// A tool on loan long enough for the return nudge to have something to say.
	if row, _ := st.One(t.Context(), `SELECT count(*) AS n FROM tools WHERE held_by IS NOT NULL AND held_since < ?`, now.AddDate(0, 0, -7).UTC().Format("2006-01-02 15:04:05")); row["n"].(int64) == 0 {
		t.Fatal("no tool has been out for a week")
	}
	// Every switch is on in the demo: it shows every room.
	for _, f := range Features {
		if !srv.feature(t.Context(), f.Key) {
			t.Errorf("the demo seeded with %s switched off", f.Key)
		}
	}
	// The steward exists and the common place is not a house.
	if row, _ := st.One(t.Context(), `SELECT count(*) AS n FROM houses WHERE is_steward=1`); row["n"].(int64) != 1 {
		t.Fatal("the demo needs exactly one steward house")
	}
	if row, _ := st.One(t.Context(), `SELECT count(*) AS n FROM houses WHERE kind='common'`); row["n"].(int64) != 1 {
		t.Fatal("the demo needs its square")
	}
	// Anchors land on their calendar day whatever the clock did in between:
	// the pruning last January, the vindima ahead this September, the olive
	// house's absence ahead in October.
	for _, c := range []struct{ q, want string }{
		{`SELECT starts_at AS v FROM events WHERE title LIKE 'Poda%'`, "2026-01-17T08:30"},
		{`SELECT starts_at AS v FROM events WHERE title='Vindima'`, "2026-09-27T08:00"},
		{`SELECT starts_at AS v FROM events WHERE title LIKE 'Magusto%'`, "2025-11-08T17:30"},
		{`SELECT from_date AS v FROM away WHERE notes LIKE 'Lisboa%'`, "2026-10-13"},
		{`SELECT to_date AS v FROM away WHERE notes LIKE 'Lisboa%'`, "2026-11-09"},
		// Two spans across the year end: the holiday ahead and last New Year.
		{`SELECT from_date || '..' || to_date AS v FROM away WHERE notes LIKE 'France%'`, "2026-12-20..2027-01-10"},
		{`SELECT from_date || '..' || to_date AS v FROM away WHERE notes LIKE 'Scotland%'`, "2025-12-27..2026-01-04"},
	} {
		if row, _ := st.One(t.Context(), c.q); row == nil || row["v"] != c.want {
			t.Errorf("%s: got %v, want %s", c.q, row, c.want)
		}
	}
	// Seeded a week into that absence, the span has begun — not jumped a year.
	st2, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	srv2 := New(st2, config.Config{Languages: []string{"pt", "en"}, VillageName: "Sobreiro Torto"})
	if err := srv2.SeedDemo(time.Date(2026, 10, 20, 10, 0, 0, 0, lisbon)); err != nil {
		t.Fatal(err)
	}
	if row, _ := st2.One(t.Context(), `SELECT from_date, to_date FROM away WHERE notes LIKE 'Lisboa%'`); row == nil || row["from_date"] != "2026-10-13" || row["to_date"] != "2026-11-09" {
		t.Fatalf("an absence that has begun: %v", row)
	}
	// Seeded inside the holiday span, past the year end: the span ahead has
	// begun and last New Year's has just ended, neither thrown a year off.
	st3, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st3.Close()
	srv3 := New(st3, config.Config{Languages: []string{"pt", "en"}, VillageName: "Sobreiro Torto"})
	if err := srv3.SeedDemo(time.Date(2027, 1, 5, 10, 0, 0, 0, lisbon)); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ q, want string }{
		{`SELECT from_date || '..' || to_date AS v FROM away WHERE notes LIKE 'France%'`, "2026-12-20..2027-01-10"},
		{`SELECT from_date || '..' || to_date AS v FROM away WHERE notes LIKE 'Scotland%'`, "2026-12-27..2027-01-04"},
	} {
		if row, _ := st3.One(t.Context(), c.q); row == nil || row["v"] != c.want {
			t.Errorf("seeded 2027-01-05, %s: got %v, want %s", c.q, row, c.want)
		}
	}

	// The rooms answer through the API as for any village.
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/status", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"houses":11`) {
		t.Fatalf("status after the seed: %d %s", rec.Code, rec.Body.String())
	}
}
