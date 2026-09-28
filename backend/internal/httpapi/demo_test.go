package httpapi

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dennislapchenko/porta-pagi/backend/internal/config"
	"github.com/dennislapchenko/porta-pagi/backend/internal/store"
)

// Lisbon: a clock change sits between September and the winter anchors,
// which is how an hour became a day once.
func lisbon(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// seedAt seeds the demo into a fresh database as if it ran at now.
func seedAt(t *testing.T, now time.Time) (*Server, *store.Store) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	srv := New(st, config.Config{Languages: []string{"pt", "en"}, VillageName: "Sobreiro Torto"})
	if err := srv.SeedDemo(now); err != nil {
		t.Fatalf("seeded at %s: %v", now, err)
	}
	return srv, st
}

// value reads one column of one row, "" when there is none.
func value(t *testing.T, st *store.Store, q string, args ...any) string {
	t.Helper()
	row, err := st.One(t.Context(), q, args...)
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	if row == nil || row["v"] == nil {
		return ""
	}
	return fmt.Sprint(row["v"])
}

// TestDemoSeed: the demo village lands whole on an empty ground, dated off
// the day it runs, every house it names defined, and never on top of a
// village that exists.
func TestDemoSeed(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, lisbon(t))
	srv, st := seedAt(t, now)
	if err := srv.SeedDemo(now); err == nil {
		t.Fatal("the demo seeded on top of itself")
	}
	for table, min := range map[string]int{"houses": 11, "posts": 60, "events": 18, "event_signups": 120, "comments": 60, "runs": 4, "needs": 19, "offers": 7, "away": 6, "tools": 15, "wishes": 3, "wish_options": 4, "projects": 5, "project_tasks": 18, "camp_takings": 6, "dues": 4, "dues_owed": 34, "roster_duties": 4, "roster_turns": 90, "contacts": 12, "codex_sections": 5, "codex_texts": 10, "map_files": 2} {
		row, err := st.One(t.Context(), `SELECT count(*) AS n FROM `+table)
		if err != nil {
			t.Fatal(err)
		}
		if n := row["n"].(int64); n < int64(min) {
			t.Errorf("%s: %d rows, expected at least %d", table, n, min)
		}
	}
	// A past event and a future one, off the seed's day; a wall clock, not UTC.
	if row, _ := st.One(t.Context(), `SELECT min(starts_at) AS a, max(starts_at) AS b FROM events`); row == nil || row["a"].(string) > "2026-09-23" || row["b"].(string) < "2026-09-24" || !strings.Contains(row["a"].(string), "T") {
		t.Fatalf("events are not spread around the seed's day: %v", row)
	}
	// Every switch is on in the demo: it shows every room.
	for _, f := range Features {
		if !srv.feature(t.Context(), f.Key) {
			t.Errorf("the demo seeded with %s switched off", f.Key)
		}
	}
	// The steward exists and the common place is not a house.
	if n := value(t, st, `SELECT count(*) AS v FROM houses WHERE is_steward=1`); n != "1" {
		t.Fatal("the demo needs exactly one steward house")
	}
	if n := value(t, st, `SELECT count(*) AS v FROM houses WHERE kind='common'`); n != "1" {
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
		// A row dated off an event moves with it: the vindima's call four
		// days before is written this morning at 12:40, so at 10:00 it is not.
		{`SELECT count(*) AS v FROM posts WHERE body LIKE 'Vindima no dia 27%'`, "0"},
		// The council whose time moved: three answers were given for the old one.
		{`SELECT count(*) AS v FROM event_signups s JOIN events e ON e.id=s.event_id WHERE e.starts_at='2026-10-04T19:00' AND s.answered_version < e.time_version`, "3"},
		// A number corrected by a house other than the one that wrote it down.
		{`SELECT h.name AS v FROM contacts c JOIN houses h ON h.id=c.edited_by WHERE c.name='Junta de Freguesia'`, "Casa das Oliveiras"},
		// A due's title names the quarter and the year the seed runs in.
		{`SELECT group_concat(title, ' | ') AS v FROM (SELECT title FROM dues WHERE title LIKE 'Água%' OR title LIKE 'Seguro%' ORDER BY title)`, "Seguro do caminho 2026 | Água do furo, 2.º trimestre | Água do furo, 3.º trimestre"},
		// The codex section a house rewrote after the council.
		{`SELECT h.name || ' ' || substr(c.updated_at, 1, 10) AS v FROM codex_sections c JOIN houses h ON h.id=c.updated_by`, "Casa da Ribeira 2026-08-14"},
	} {
		if v := value(t, st, c.q); v != c.want {
			t.Errorf("%s: got %q, want %s", c.q, v, c.want)
		}
	}
	// Seeded a week into that absence, the span has begun — not jumped a year.
	_, st2 := seedAt(t, time.Date(2026, 10, 20, 10, 0, 0, 0, lisbon(t)))
	if v := value(t, st2, `SELECT from_date || '..' || to_date AS v FROM away WHERE notes LIKE 'Lisboa%'`); v != "2026-10-13..2026-11-09" {
		t.Fatalf("an absence that has begun: %v", v)
	}
	// Seeded inside the holiday span, past the year end: the span ahead has
	// begun and last New Year's has just ended, neither thrown a year off.
	_, st3 := seedAt(t, time.Date(2027, 1, 5, 10, 0, 0, 0, lisbon(t)))
	for _, c := range []struct{ q, want string }{
		{`SELECT from_date || '..' || to_date AS v FROM away WHERE notes LIKE 'France%'`, "2026-12-20..2027-01-10"},
		{`SELECT from_date || '..' || to_date AS v FROM away WHERE notes LIKE 'Scotland%'`, "2026-12-27..2027-01-04"},
	} {
		if v := value(t, st3, c.q); v != c.want {
			t.Errorf("seeded 2027-01-05, %s: got %v, want %s", c.q, v, c.want)
		}
	}

	// The rooms answer through the API as for any village.
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/status", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"houses":11`) {
		t.Fatalf("status after the seed: %d %s", rec.Code, rec.Body.String())
	}
}

// TestDemoSeasons: what the land decides moves as one thing — a harvest
// keeps its thread for the days after it, then gives its place to next
// year's, which has no thread yet; a project anchored in April is still
// being built in April.
func TestDemoSeasons(t *testing.T) {
	loc := lisbon(t)
	at := func(y int, m time.Month, d int) *store.Store {
		_, st := seedAt(t, time.Date(y, m, d, 10, 0, 0, 0, loc))
		return st
	}
	thanks := `SELECT count(*) AS v FROM comments c JOIN events e ON e.id=c.subject_id AND c.subject='event' WHERE e.title='Vindima' AND c.body LIKE 'Obrigada a todos%'`
	for _, c := range []struct {
		st      *store.Store
		when    string
		q, want string
	}{
		// Three days after the vindima: it stayed, with the thanks under it
		// and the call before it on the board.
		{at(2026, 9, 30), "2026-09-30", `SELECT starts_at AS v FROM events WHERE title='Vindima'`, "2026-09-27T08:00"},
		{at(2026, 9, 30), "2026-09-30", thanks, "1"},
		{at(2026, 9, 30), "2026-09-30", `SELECT count(*) AS v FROM posts WHERE body LIKE 'Vindima no dia 27%'`, "1"},
		// Two weeks after, next year's vindima is on the calendar, bare.
		{at(2026, 10, 10), "2026-10-10", `SELECT starts_at AS v FROM events WHERE title='Vindima'`, "2027-09-27T08:00"},
		{at(2026, 10, 10), "2026-10-10", thanks, "0"},
		// The olive harvest: kept six days after, next year's sixteen days after.
		{at(2026, 11, 20), "2026-11-20", `SELECT starts_at AS v FROM events WHERE title LIKE 'Apanha%'`, "2026-11-14T08:30"},
		{at(2026, 11, 30), "2026-11-30", `SELECT starts_at AS v FROM events WHERE title LIKE 'Apanha%'`, "2027-11-14T08:30"},
		// The chicken fence in April: begun on the 1st, its second work party
		// ahead, the door not built yet, the project not finished.
		{at(2026, 4, 10), "2026-04-10", `SELECT state AS v FROM projects WHERE title LIKE 'Vedação%'`, "open"},
		{at(2026, 4, 10), "2026-04-10", `SELECT starts_at AS v FROM events WHERE title LIKE 'Galinheiro: porta%'`, "2026-04-25T09:00"},
		{at(2026, 4, 10), "2026-04-10", `SELECT state AS v FROM project_tasks WHERE title LIKE 'Porta com fecho%'`, "open"},
		// And in September it is done, on the day it was.
		{at(2026, 9, 23), "2026-09-23", `SELECT state || ' ' || substr(done_at, 1, 10) AS v FROM projects WHERE title LIKE 'Vedação%'`, "done 2026-05-01"},
	} {
		if v := value(t, c.st, c.q); v != c.want {
			t.Errorf("seeded %s, %s: got %q, want %s", c.when, c.q, v, c.want)
		}
	}
}

// TestDemoAnyDay: whenever the hour reseeds it, the demo shows every
// feature a visitor can open in the rooms, and never a line written in the
// future. Twice a month through a year, at three in the morning (before
// the day's lines exist) and late in the evening.
func TestDemoAnyDay(t *testing.T) {
	loc := lisbon(t)
	for m := time.January; m <= time.December; m++ {
		for _, d := range []int{1, 16} {
			for _, hh := range []int{3, 22} {
				now := time.Date(2027, m, d, hh, 30, 0, 0, loc)
				t.Run(now.Format("01-02T15"), func(t *testing.T) { demoShowsEverything(t, now) })
			}
		}
	}
}

func demoShowsEverything(t *testing.T, now time.Time) {
	_, st := seedAt(t, now)
	utcNow := now.UTC().Format("2006-01-02 15:04:05")
	wallNow := now.Format("2006-01-02T15:04")
	today := now.Format("2006-01-02")
	// The windows the rooms list by: sixty days of events, runs from a day
	// back, done needs and give-aways for a week, absences three days on.
	since60 := now.AddDate(0, 0, -60).Format("2006-01-02")
	runCut := now.AddDate(0, 0, -1).Format("2006-01-02T15:04")
	week := now.AddDate(0, 0, -7).UTC().Format("2006-01-02 15:04:05")
	awayCut := now.AddDate(0, 0, -3).Format("2006-01-02")
	for _, table := range []string{"posts", "comments", "needs", "offers", "wishes", "wish_options", "camp_takings", "contacts", "project_tasks", "events", "event_signups", "away", "projects", "runs"} {
		col := "created_at"
		if table == "event_signups" {
			col = "answered_at"
		}
		if v := value(t, st, `SELECT count(*) AS v FROM `+table+` WHERE `+col+` > ?`, utcNow); v != "0" {
			t.Errorf("%s: %s rows written after now", table, v)
		}
	}
	for what, q := range map[string]string{
		"a pinned post":                                      `SELECT count(*) AS v FROM posts WHERE pinned=1`,
		"a reply on the board":                               `SELECT count(*) AS v FROM posts WHERE parent_id IS NOT NULL`,
		"a past event with a thread":                         `SELECT count(*) AS v FROM events e WHERE starts_at >= '` + since60 + `' AND starts_at < '` + wallNow + `' AND EXISTS (SELECT 1 FROM comments c WHERE c.subject='event' AND c.subject_id=e.id)`,
		"an event ahead with a thread":                       `SELECT count(*) AS v FROM events e WHERE starts_at > '` + wallNow + `' AND EXISTS (SELECT 1 FROM comments c WHERE c.subject='event' AND c.subject_id=e.id)`,
		"a work party listed":                                `SELECT count(*) AS v FROM events WHERE kind='work' AND starts_at >= '` + since60 + `'`,
		"an ordinary event listed":                           `SELECT count(*) AS v FROM events WHERE kind='event' AND starts_at >= '` + since60 + `'`,
		"a no listed":                                        `SELECT count(*) AS v FROM event_signups s JOIN events e ON e.id=s.event_id WHERE s.state='no' AND e.starts_at >= '` + since60 + `'`,
		"a maybe listed":                                     `SELECT count(*) AS v FROM event_signups s JOIN events e ON e.id=s.event_id WHERE s.state='maybe' AND e.starts_at >= '` + since60 + `'`,
		"a moved event with struck answers":                  `SELECT count(*) AS v FROM event_signups s JOIN events e ON e.id=s.event_id WHERE s.answered_version < e.time_version AND e.edited_by IS NOT NULL AND e.starts_at >= '` + since60 + `'`,
		"an event on a project's task":                       `SELECT count(*) AS v FROM events WHERE task_id IS NOT NULL AND starts_at >= '` + since60 + `'`,
		"a reply inside a thread":                            `SELECT count(*) AS v FROM comments WHERE parent_id IS NOT NULL`,
		"a run ahead with needs on it":                       `SELECT count(*) AS v FROM runs r WHERE cutoff_at >= '` + runCut + `' AND EXISTS (SELECT 1 FROM needs n WHERE n.run_id=r.id AND n.state!='done')`,
		"an open need":                                       `SELECT count(*) AS v FROM needs WHERE state='open'`,
		"a taken need":                                       `SELECT count(*) AS v FROM needs WHERE state='taken'`,
		"a done need still listed":                           `SELECT count(*) AS v FROM needs WHERE state='done' AND created_at >= '` + week + `'`,
		"an open give-away":                                  `SELECT count(*) AS v FROM offers WHERE state='open'`,
		"a claimed give-away":                                `SELECT count(*) AS v FROM offers WHERE state='claimed'`,
		"a done give-away still listed":                      `SELECT count(*) AS v FROM offers WHERE state='done' AND created_at >= '` + week + `'`,
		"seeds":                                              `SELECT count(*) AS v FROM offers WHERE tag='seeds' AND state!='done'`,
		"surplus":                                            `SELECT count(*) AS v FROM offers WHERE tag='surplus' AND state!='done'`,
		"a trip ahead with needs on it":                      `SELECT count(*) AS v FROM runs r WHERE kind='trip' AND cutoff_at >= '` + runCut + `' AND EXISTS (SELECT 1 FROM needs n WHERE n.run_id=r.id)`,
		"an order ahead with needs on it":                    `SELECT count(*) AS v FROM runs r WHERE kind='order' AND cutoff_at >= '` + runCut + `' AND EXISTS (SELECT 1 FROM needs n WHERE n.run_id=r.id)`,
		"a due partly paid, with an amount and a date ahead": `SELECT count(*) AS v FROM dues d WHERE amount_cents > 0 AND due_on >= '` + today + `' AND EXISTS (SELECT 1 FROM dues_owed o WHERE o.due_id=d.id AND o.paid_at IS NOT NULL) AND EXISTS (SELECT 1 FROM dues_owed o WHERE o.due_id=d.id AND o.paid_at IS NULL)`,
		"a due every house has paid":                         `SELECT count(*) AS v FROM dues d WHERE NOT EXISTS (SELECT 1 FROM dues_owed o WHERE o.due_id=d.id AND o.paid_at IS NULL)`,
		"a due with no amount":                               `SELECT count(*) AS v FROM dues WHERE amount_cents IS NULL`,
		"a payment a steward marked":                         `SELECT count(*) AS v FROM dues_owed o JOIN dues d ON d.id=o.due_id JOIN houses h ON h.id=o.paid_by WHERE h.is_steward=1 AND o.paid_by != d.collector_id`,
		"a duty due":                                         `SELECT count(*) AS v FROM roster_duties d WHERE every_days IS NOT NULL AND date((SELECT max(on_date) FROM roster_turns t WHERE t.duty_id=d.id AND t.state='done'), '+' || every_days || ' days') <= '` + today + `'`,
		"a duty not due yet":                                 `SELECT count(*) AS v FROM roster_duties d WHERE every_days IS NOT NULL AND date((SELECT max(on_date) FROM roster_turns t WHERE t.duty_id=d.id AND t.state='done'), '+' || every_days || ' days') > '` + today + `'`,
		"a duty done when needed":                            `SELECT count(*) AS v FROM roster_duties d WHERE every_days IS NULL AND EXISTS (SELECT 1 FROM roster_turns t WHERE t.duty_id=d.id AND t.state='done')`,
		"a turn planned ahead":                               `SELECT count(*) AS v FROM roster_turns WHERE state='planned' AND on_date >= '` + today + `'`,
		"a turn with a note":                                 `SELECT count(*) AS v FROM roster_turns WHERE state='done' AND note != '' AND on_date >= '` + now.AddDate(-1, 0, 0).Format("2006-01-02") + `'`,
		"a house away now, watched, talked about":            `SELECT count(*) AS v FROM away a WHERE from_date <= '` + today + `' AND to_date >= '` + today + `' AND watcher IS NOT NULL AND EXISTS (SELECT 1 FROM comments c WHERE c.subject='away' AND c.subject_id=a.id)`,
		"a house away soon":                                  `SELECT count(*) AS v FROM away WHERE from_date > '` + today + `'`,
		"an absence with nobody watching":                    `SELECT count(*) AS v FROM away WHERE watcher IS NULL AND to_date >= '` + awayCut + `'`,
		"an absence just over":                               `SELECT count(*) AS v FROM away WHERE to_date < '` + today + `' AND to_date >= '` + awayCut + `'`,
		"a camper arrived":                                   `SELECT count(*) AS v FROM camp_takings WHERE state='arrived'`,
		"a camper's money held":                              `SELECT count(*) AS v FROM camp_takings WHERE state='held' AND held_by IS NOT NULL`,
		"a camper's money handed":                            `SELECT count(*) AS v FROM camp_takings WHERE state='handed'`,
		"a tool out over a week":                             `SELECT count(*) AS v FROM tools WHERE held_by IS NOT NULL AND held_since < '` + now.AddDate(0, 0, -7).Format("2006-01-02") + `'`,
		"a tool in the shed":                                 `SELECT count(*) AS v FROM tools WHERE held_by IS NULL`,
		"a wish with findings and a thread":                  `SELECT count(*) AS v FROM wishes w WHERE EXISTS (SELECT 1 FROM wish_options o WHERE o.wish_id=w.id) AND EXISTS (SELECT 1 FROM comments c WHERE c.subject='wish' AND c.subject_id=w.id) AND EXISTS (SELECT 1 FROM wish_wants x WHERE x.wish_id=w.id)`,
		"a planned project":                                  `SELECT count(*) AS v FROM projects WHERE state='planned'`,
		"a project in progress":                              `SELECT count(*) AS v FROM projects WHERE state='open'`,
		"a finished project":                                 `SELECT count(*) AS v FROM projects WHERE state='done'`,
		"a free task":                                        `SELECT count(*) AS v FROM project_tasks WHERE state='open' AND assigned_to IS NULL`,
		"a task another house holds":                         `SELECT count(*) AS v FROM project_tasks WHERE state='open' AND assigned_to IS NOT NULL AND assigned_to != house_id`,
		"a task done with a note":                            `SELECT count(*) AS v FROM project_tasks WHERE state='done' AND closing_note != ''`,
		"a task with a due date":                             `SELECT count(*) AS v FROM project_tasks WHERE state='open' AND due_at IS NOT NULL`,
		"a number with a thread":                             `SELECT count(*) AS v FROM contacts c WHERE EXISTS (SELECT 1 FROM comments m WHERE m.subject='contact' AND m.subject_id=c.id)`,
		"a number corrected":                                 `SELECT count(*) AS v FROM contacts WHERE edited_by IS NOT NULL`,
		"a number written down a year ago":                   `SELECT count(*) AS v FROM contacts WHERE created_at < '` + now.AddDate(0, 0, -300).UTC().Format("2006-01-02") + `'`,
		"a number written down lately":                       `SELECT count(*) AS v FROM contacts WHERE created_at > '` + now.AddDate(0, 0, -90).UTC().Format("2006-01-02") + `'`,
		"a codex section a house rewrote":                    `SELECT count(*) AS v FROM codex_sections WHERE updated_by IS NOT NULL`,
	} {
		if v := value(t, st, q); v == "0" || v == "" {
			t.Errorf("nothing shows %s", what)
		}
	}
	// The rooms' own rules on dates: a done turn is today or before, a plan
	// today or after; a payment is never ahead. The give-away pickers no
	// longer offer a joint order, so the demo does not show one.
	for what, q := range map[string]string{
		"a done turn ahead":        `SELECT count(*) AS v FROM roster_turns WHERE state='done' AND (on_date > '` + today + `' OR done_at > '` + utcNow + `')`,
		"a plan in the past":       `SELECT count(*) AS v FROM roster_turns WHERE state='planned' AND on_date < '` + today + `'`,
		"a payment ahead":          `SELECT count(*) AS v FROM dues_owed WHERE paid_at > '` + utcNow + `'`,
		"a due written ahead":      `SELECT count(*) AS v FROM dues WHERE created_at > '` + utcNow + `'`,
		"a joint give-away":        `SELECT count(*) AS v FROM offers WHERE tag='joint'`,
		"a common place on a rota": `SELECT count(*) AS v FROM roster_turns t JOIN houses h ON h.id=t.house_id WHERE h.kind!='house'`,
	} {
		if v := value(t, st, q); v != "0" {
			t.Errorf("%s: %s rows", what, v)
		}
	}
	// The newcomer has taken the fewest turns.
	if v := value(t, st, `SELECT h.name AS v FROM houses h LEFT JOIN roster_turns t ON t.house_id=h.id AND t.state='done' WHERE h.kind='house' GROUP BY h.id ORDER BY count(t.id), h.name LIMIT 1`); v != "Casa Nova" {
		t.Errorf("the house with the fewest turns is %s, not the newcomer", v)
	}
	// Every subject that takes a thread has one.
	for _, s := range commentSubjects {
		if v := value(t, st, `SELECT count(*) AS v FROM comments WHERE subject=?`, s); v == "0" {
			t.Errorf("no %s has a thread", s)
		}
	}
	// No plate, no amount, no nationality on a camper.
	if v := value(t, st, `SELECT count(*) AS v FROM camp_takings WHERE from_who != '' OR notes GLOB '*[0-9]*€*' OR notes GLOB '*€*'`); v != "0" {
		t.Errorf("%s camp rows carry a label or an amount", v)
	}
}
