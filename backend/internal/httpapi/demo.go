package httpapi

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dennislapchenko/porta-pagi/backend/internal/store"
)

// The demo village: a year of an invented collective's life, written into an
// empty database by `/server demo-seed`, dated relative to the moment it runs
// so it never reads as abandoned. Every house, name and parcel is invented;
// the cadastre is a synthetic mesh that is nobody's land. The words are in
// demo/seed.json — the writing job, not the coding job — and the demo refuses
// a village that already has houses, like the codex import: it is a seed for
// an empty ground, never an overlay.

//go:embed demo/*
var demoFS embed.FS

// The seed's shape. A row is dated one of two ways. "day" is an offset from
// the seed's day (negative = past): for the last few weeks and the next,
// where "three days ago" is the point. "on" is a calendar anchor, "MM-DD",
// resolved to its most recent occurrence — or, with "ahead", to its next —
// for anything the land decides: the olive harvest, the frost, São João, so
// a demo reseeded in March does not pick grapes. Clocks are the village's
// wall clock. The one seam this leaves: a post written for the weeks before
// an anchored event ahead is relative, and in March it precedes a September
// vindima by half a year. The demo is truest around the month it was written.
type demoWhen struct {
	Day   int
	On    string
	Ahead bool
	Time  string
}

type demoSeed struct {
	Houses []struct {
		Key, Name, Crest, Color, Kind string
		Steward                       bool
		Parcels, Homes                []string
	}
	Codex struct {
		Adopted  int // days ago
		Sections []struct {
			Texts map[string]codexText
		}
	}
	Posts []struct {
		demoWhen
		House, Author, Body string
		Pinned              bool
		Replies             []demoComment
	}
	Events []struct {
		demoWhen
		House, Kind, Title, Place, Notes, Project string
		Start, End                                string
		Answers                                   map[string]string // house key -> yes/no/maybe
		Comments                                  []demoComment
	}
	Runs []struct {
		demoWhen
		House, Destination, Notes string
		Needs                     []demoNeed
	}
	Needs  []demoNeed
	Offers []struct {
		demoWhen
		House, Text, Tag, State, ClaimedBy string
	}
	Away []struct {
		House, Notes, Watcher string
		From, To              int
		OnFrom, OnTo          string
		Ahead                 bool
		Comments              []demoComment
	}
	Tools []struct {
		House, Name, Notes, Category, HeldBy string
		HeldDays                             int // days on loan; the nudge reads off held_since
	}
	Wishes []struct {
		demoWhen
		House, Text string
		Wants       []string
		Options     []struct{ House, Text, URL string }
		Comments    []demoComment
	}
	Projects []struct {
		demoWhen
		Key, House, Title, Notes, State string
		DoneDay                         int
		DoneOn                          string
		Tasks                           []struct {
			demoWhen
			House, Title, Notes, AssignedTo, State, ClosingNote string
			DoneDay                                             int
			DoneOn                                              string
		}
	}
	Camp []struct {
		demoWhen
		House, FromWho, Notes, State, HeldBy string
	}
	Contacts []struct {
		House, Name, Phone, Notes, Type string
	}
}

type demoNeed struct {
	demoWhen
	House, Text, State, TakenBy string
}

type demoComment struct {
	demoWhen
	House, Author, Body string
}

// SeedDemo writes the demo village into an empty database, in one
// transaction. now is the village's wall clock; every date hangs off it.
func (s *Server) SeedDemo(now time.Time) error {
	ctx := context.Background()
	row, err := s.st.One(ctx, `SELECT count(*) AS n FROM houses`)
	if err != nil {
		return err
	}
	if row["n"].(int64) > 0 {
		return fmt.Errorf("the village already has %d houses; the demo seeds an empty ground only", row["n"])
	}
	var counts map[string]int
	if err := s.st.Tx(ctx, func(tx *store.Store) error {
		counts, err = seedDemo(ctx, tx, now)
		return err
	}); err != nil {
		return err
	}
	fmt.Printf("demo village seeded: %v\n", counts)
	return nil
}

// seedDemo does the writing, inside the caller's transaction, so the demo
// build's reset can wipe and reseed as one thing.
func seedDemo(ctx context.Context, tx *store.Store, now time.Time) (map[string]int, error) {
	raw, err := demoFS.ReadFile("demo/seed.json")
	if err != nil {
		return nil, err
	}
	var seed demoSeed
	if err := json.Unmarshal(raw, &seed); err != nil {
		return nil, fmt.Errorf("demo/seed.json: %w", err)
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// dayOf resolves a row's date to an offset from today: the anchor's most
	// recent occurrence (or its next, when the row is ahead), the plain offset
	// when it has none.
	dayOf := func(w demoWhen) int {
		if w.On == "" {
			return w.Day
		}
		t, err := time.ParseInLocation("2006-01-02", fmt.Sprintf("%d-%s", now.Year(), w.On), now.Location())
		if err != nil {
			panic("demo seed: bad anchor " + w.On)
		}
		off := int(t.Sub(today).Hours() / 24)
		if w.Ahead && off < 0 {
			off = int(t.AddDate(1, 0, 0).Sub(today).Hours() / 24)
		} else if !w.Ahead && off > 0 {
			off = int(t.AddDate(-1, 0, 0).Sub(today).Hours() / 24)
		}
		return off
	}
	anchor := func(day int, on string, ahead bool) int { return dayOf(demoWhen{Day: day, On: on, Ahead: ahead}) }
	// Two clocks: a wall-clock string for what a villager typed, a UTC stamp
	// with a space for what the database would have written itself.
	wall := func(day int, clock string) string {
		if clock == "" {
			clock = "12:00"
		}
		return today.AddDate(0, 0, day).Format("2006-01-02") + "T" + clock
	}
	date := func(day int) string { return today.AddDate(0, 0, day).Format("2006-01-02") }
	stamp := func(day int, clock string) string {
		t, err := time.ParseInLocation("2006-01-02T15:04", wall(day, clock), now.Location())
		if err != nil {
			t = today.AddDate(0, 0, day)
		}
		return t.UTC().Format("2006-01-02 15:04:05")
	}
	houses := map[string]int64{}
	id := func(key string) any {
		if key == "" {
			return nil
		}
		v, ok := houses[key]
		if !ok {
			panic("demo seed names a house it never defined: " + key)
		}
		return v
	}
	counts := map[string]int{}
	comments := func(subject string, sid int64, cs []demoComment) error {
		for _, c := range cs {
			if _, err := tx.Exec(ctx, `INSERT INTO comments(subject, subject_id, house_id, author, body, created_at) VALUES (?,?,?,?,?,?)`, subject, sid, id(c.House), c.Author, c.Body, stamp(dayOf(c.demoWhen), c.Time)); err != nil {
				return err
			}
		}
		return nil
	}
	for _, h := range seed.Houses {
		kind, crest, color := orDefault(h.Kind, "house"), orDefault(h.Crest, "🏠"), orDefault(h.Color, "#b5651d")
		hid, err := tx.Exec(ctx, `INSERT INTO houses(name, crest, color, kind, is_steward, created_at) VALUES (?,?,?,?,?,?)`, h.Name, crest, color, kind, b2i(h.Steward), stamp(-400, "10:00"))
		if err != nil {
			return nil, err
		}
		houses[h.Key] = hid
		for _, p := range h.Parcels {
			if _, err := tx.Exec(ctx, `INSERT INTO house_parcels(house_id, parcel) VALUES (?,?)`, hid, p); err != nil {
				return nil, err
			}
		}
		for _, p := range h.Homes {
			if _, err := tx.Exec(ctx, `INSERT INTO house_homes(house_id, parcel) VALUES (?,?)`, hid, p); err != nil {
				return nil, err
			}
		}
		counts["houses"]++
	}
	// The map: the synthetic cadastre and water, as rows like any village's.
	for name, src := range map[string]struct{ file, source, snap string }{"parcels": {"demo/parcels.geojson", "synthetic cadastre — nobody's land", date(-40)}, "water": {"demo/water.json", "", ""}} {
		b, err := demoFS.ReadFile(src.file)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO map_files (name, bytes, source, snapshot) VALUES (?,?,?,?)`, name, b, src.source, src.snap); err != nil {
			return nil, err
		}
	}
	// The codex, adopted by the council on a day, no house.
	for i, sec := range seed.Codex.Sections {
		sid, err := tx.Exec(ctx, `INSERT INTO codex_sections(ord, updated_at, updated_by) VALUES (?,?,NULL)`, i+1, date(-seed.Codex.Adopted)+" 12:00:00")
		if err != nil {
			return nil, err
		}
		if err := writeTexts(ctx, tx, sid, sec.Texts); err != nil {
			return nil, err
		}
		counts["codex sections"]++
	}
	for _, p := range seed.Posts {
		pid, err := tx.Exec(ctx, `INSERT INTO posts(house_id, author, body, pinned, created_at) VALUES (?,?,?,?,?)`, id(p.House), p.Author, p.Body, b2i(p.Pinned), stamp(dayOf(p.demoWhen), p.Time))
		if err != nil {
			return nil, err
		}
		counts["posts"]++
		for _, r := range p.Replies {
			if _, err := tx.Exec(ctx, `INSERT INTO posts(house_id, parent_id, author, body, created_at) VALUES (?,?,?,?,?)`, id(r.House), pid, r.Author, r.Body, stamp(dayOf(r.demoWhen), r.Time)); err != nil {
				return nil, err
			}
			counts["posts"]++
		}
	}
	projects := map[string]int64{}
	for _, p := range seed.Projects {
		state := orDefault(p.State, "planned")
		var done any
		if state == "done" {
			done = stamp(anchor(p.DoneDay, p.DoneOn, false), "17:00")
		}
		pid, err := tx.Exec(ctx, `INSERT INTO projects(house_id, title, notes, state, done_at, created_at) VALUES (?,?,?,?,?,?)`, id(p.House), p.Title, p.Notes, state, done, stamp(dayOf(p.demoWhen), "20:00"))
		if err != nil {
			return nil, err
		}
		projects[p.Key] = pid
		counts["projects"]++
		for _, t := range p.Tasks {
			ts := orDefault(t.State, "open")
			var tdone any
			if ts == "done" {
				tdone = stamp(anchor(t.DoneDay, t.DoneOn, false), "16:00")
			}
			if _, err := tx.Exec(ctx, `INSERT INTO project_tasks(project_id, house_id, title, notes, assigned_to, state, done_at, closing_note, created_at) VALUES (?,?,?,?,?,?,?,?,?)`, pid, id(t.House), t.Title, t.Notes, id(t.AssignedTo), ts, tdone, t.ClosingNote, stamp(dayOf(t.demoWhen), "20:10")); err != nil {
				return nil, err
			}
			counts["tasks"]++
		}
	}
	for _, e := range seed.Events {
		day := dayOf(e.demoWhen)
		var end any
		if e.End != "" {
			end = wall(day, e.End)
		}
		var proj any
		if e.Project != "" {
			proj = projects[e.Project]
		}
		eid, err := tx.Exec(ctx, `INSERT INTO events(house_id, title, kind, starts_at, ends_at, place, notes, project_id, created_at) VALUES (?,?,?,?,?,?,?,?,?)`, id(e.House), e.Title, orDefault(e.Kind, "event"), wall(day, e.Start), end, e.Place, e.Notes, proj, stamp(day-9, "19:30"))
		if err != nil {
			return nil, err
		}
		counts["events"]++
		// The caller answers yes in the same breath, like the app does.
		answers := map[string]string{e.House: "yes"}
		for k, v := range e.Answers {
			answers[k] = v
		}
		for k, v := range answers {
			if _, err := tx.Exec(ctx, `INSERT INTO event_signups(event_id, house_id, state, answered_at, answered_version) VALUES (?,?,?,?,0)`, eid, id(k), v, stamp(day-7, "21:00")); err != nil {
				return nil, err
			}
		}
		if err := comments("event", eid, e.Comments); err != nil {
			return nil, err
		}
	}
	for _, r := range seed.Runs {
		day := dayOf(r.demoWhen)
		rid, err := tx.Exec(ctx, `INSERT INTO runs(house_id, destination, cutoff_at, notes, created_at) VALUES (?,?,?,?,?)`, id(r.House), r.Destination, wall(day, r.Time), r.Notes, stamp(day-2, "08:15"))
		if err != nil {
			return nil, err
		}
		counts["runs"]++
		for _, n := range r.Needs {
			if _, err := tx.Exec(ctx, `INSERT INTO needs(house_id, text, state, taken_by, run_id, created_at) VALUES (?,?,?,?,?,?)`, id(n.House), n.Text, orDefault(n.State, "open"), id(n.TakenBy), rid, stamp(dayOf(n.demoWhen), "09:40")); err != nil {
				return nil, err
			}
			counts["needs"]++
		}
	}
	for _, n := range seed.Needs {
		if _, err := tx.Exec(ctx, `INSERT INTO needs(house_id, text, state, taken_by, created_at) VALUES (?,?,?,?,?)`, id(n.House), n.Text, orDefault(n.State, "open"), id(n.TakenBy), stamp(dayOf(n.demoWhen), "11:20")); err != nil {
			return nil, err
		}
		counts["needs"]++
	}
	for _, o := range seed.Offers {
		if _, err := tx.Exec(ctx, `INSERT INTO offers(house_id, text, tag, state, claimed_by, created_at) VALUES (?,?,?,?,?,?)`, id(o.House), o.Text, orDefault(o.Tag, "giveaway"), orDefault(o.State, "open"), id(o.ClaimedBy), stamp(dayOf(o.demoWhen), "17:50")); err != nil {
			return nil, err
		}
		counts["offers"]++
	}
	for _, a := range seed.Away {
		from, to := anchor(a.From, a.OnFrom, a.Ahead), anchor(a.To, a.OnTo, a.Ahead)
		if to < from { // an anchored span that wrapped the year end
			to += 365
		}
		aid, err := tx.Exec(ctx, `INSERT INTO away(house_id, from_date, to_date, notes, watcher, created_at) VALUES (?,?,?,?,?,?)`, id(a.House), date(from), date(to), a.Notes, id(a.Watcher), stamp(from-5, "20:30"))
		if err != nil {
			return nil, err
		}
		counts["away"]++
		if err := comments("away", aid, a.Comments); err != nil {
			return nil, err
		}
	}
	for _, t := range seed.Tools {
		var held, since any
		if t.HeldBy != "" {
			held, since = id(t.HeldBy), stamp(-t.HeldDays, "16:00")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO tools(house_id, name, notes, category, held_by, held_since, created_at) VALUES (?,?,?,?,?,?,?)`, id(t.House), t.Name, t.Notes, orDefault(t.Category, "other"), held, since, stamp(-300, "12:00")); err != nil {
			return nil, err
		}
		counts["tools"]++
	}
	for _, w := range seed.Wishes {
		day := dayOf(w.demoWhen)
		wid, err := tx.Exec(ctx, `INSERT INTO wishes(house_id, text, created_at) VALUES (?,?,?)`, id(w.House), w.Text, stamp(day, "21:15"))
		if err != nil {
			return nil, err
		}
		counts["wishes"]++
		for _, k := range w.Wants {
			if _, err := tx.Exec(ctx, `INSERT INTO wish_wants(wish_id, house_id) VALUES (?,?)`, wid, id(k)); err != nil {
				return nil, err
			}
		}
		for _, o := range w.Options {
			if _, err := tx.Exec(ctx, `INSERT INTO wish_options(wish_id, house_id, text, url, created_at) VALUES (?,?,?,?,?)`, wid, id(o.House), o.Text, o.URL, stamp(day+2, "22:00")); err != nil {
				return nil, err
			}
		}
		if err := comments("wish", wid, w.Comments); err != nil {
			return nil, err
		}
	}
	for _, c := range seed.Camp {
		day := dayOf(c.demoWhen)
		state := orDefault(c.State, "arrived")
		var held, heldAt, handed any
		if state == "held" || state == "handed" {
			held, heldAt = id(orDefault(c.HeldBy, c.House)), stamp(day, "18:00")
		}
		if state == "handed" {
			handed = stamp(day+2, "19:00")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO camp_takings(house_id, from_who, taken_on, notes, state, held_by, held_at, handed_at, created_at) VALUES (?,?,?,?,?,?,?,?,?)`, id(c.House), c.FromWho, date(day), c.Notes, state, held, heldAt, handed, stamp(day, "17:30")); err != nil {
			return nil, err
		}
		counts["camp rows"]++
	}
	for _, c := range seed.Contacts {
		if _, err := tx.Exec(ctx, `INSERT INTO contacts(house_id, name, phone, notes, type, created_at) VALUES (?,?,?,?,?,?)`, id(c.House), c.Name, c.Phone, c.Notes, c.Type, stamp(-120, "13:00")); err != nil {
			return nil, err
		}
		counts["contacts"]++
	}
	return counts, nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
