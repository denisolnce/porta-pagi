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

// The seed's shape. Days are offsets from the seed's day (negative = past),
// clocks are the village's wall clock, so "day": -3, "time": "18:40" is three
// days ago at twenty to seven, wherever the demo runs.
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
		House, Author, Body string
		Day                 int
		Time                string
		Pinned              bool
		Replies             []struct {
			House, Author, Body string
			Day                 int
			Time                string
		}
	}
	Events []struct {
		House, Kind, Title, Place, Notes, Project string
		Day                                       int
		Start, End                                string
		Answers                                   map[string]string // house key -> yes/no/maybe
		Comments                                  []demoComment
	}
	Runs []struct {
		House, Destination, Notes string
		Day                       int
		Time                      string
		Needs                     []struct {
			House, Text, State, TakenBy string
			Day                         int
		}
	}
	Needs []struct {
		House, Text, State, TakenBy string
		Day                         int
	}
	Offers []struct {
		House, Text, Tag, State, ClaimedBy string
		Day                                int
	}
	Away []struct {
		House, Notes, Watcher string
		From, To              int
		Comments              []demoComment
	}
	Tools []struct {
		House, Name, Notes, Category, HeldBy string
		HeldDays                             int // days on loan; the nudge reads off held_since
	}
	Wishes []struct {
		House, Text string
		Day         int
		Wants       []string
		Options     []struct{ House, Text, URL string }
		Comments    []demoComment
	}
	Projects []struct {
		Key, House, Title, Notes, State string
		Day, DoneDay                    int
		Tasks                           []struct {
			House, Title, Notes, AssignedTo, State, ClosingNote string
			Day, DoneDay                                        int
		}
	}
	Camp []struct {
		House, FromWho, Notes, State, HeldBy string
		Day                                  int
	}
	Contacts []struct {
		House, Name, Phone, Notes, Type string
	}
}

type demoComment struct {
	House, Author, Body string
	Day                 int
	Time                string
}

// SeedDemo writes the demo village into an empty database. now is the
// village's wall clock; every date in the seed hangs off it.
func (s *Server) SeedDemo(now time.Time) error {
	ctx := context.Background()
	row, err := s.st.One(ctx, `SELECT count(*) AS n FROM houses`)
	if err != nil {
		return err
	}
	if row["n"].(int64) > 0 {
		return fmt.Errorf("the village already has %d houses; the demo seeds an empty ground only", row["n"])
	}
	raw, err := demoFS.ReadFile("demo/seed.json")
	if err != nil {
		return err
	}
	var seed demoSeed
	if err := json.Unmarshal(raw, &seed); err != nil {
		return fmt.Errorf("demo/seed.json: %w", err)
	}
	// Two clocks: a wall-clock string for what a villager typed, a UTC stamp
	// with a space for what the database would have written itself.
	wall := func(day int, clock string) string {
		if clock == "" {
			clock = "12:00"
		}
		return now.AddDate(0, 0, day).Format("2006-01-02") + "T" + clock
	}
	date := func(day int) string { return now.AddDate(0, 0, day).Format("2006-01-02") }
	stamp := func(day int, clock string) string {
		t, err := time.ParseInLocation("2006-01-02T15:04", wall(day, clock), now.Location())
		if err != nil {
			t = now.AddDate(0, 0, day)
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
	err = s.st.Tx(ctx, func(tx *store.Store) error {
		for _, h := range seed.Houses {
			kind, crest, color := h.Kind, h.Crest, h.Color
			if kind == "" {
				kind = "house"
			}
			if crest == "" {
				crest = "🏠"
			}
			if color == "" {
				color = "#b5651d"
			}
			hid, err := tx.Exec(ctx, `INSERT INTO houses(name, crest, color, kind, is_steward, created_at) VALUES (?,?,?,?,?,?)`, h.Name, crest, color, kind, b2i(h.Steward), stamp(-400, "10:00"))
			if err != nil {
				return err
			}
			houses[h.Key] = hid
			for _, p := range h.Parcels {
				if _, err := tx.Exec(ctx, `INSERT INTO house_parcels(house_id, parcel) VALUES (?,?)`, hid, p); err != nil {
					return err
				}
			}
			for _, p := range h.Homes {
				if _, err := tx.Exec(ctx, `INSERT INTO house_homes(house_id, parcel) VALUES (?,?)`, hid, p); err != nil {
					return err
				}
			}
			counts["houses"]++
		}
		// The map: the synthetic cadastre and water, as rows like any village's.
		for name, src := range map[string]struct{ file, source, snap string }{"parcels": {"demo/parcels.geojson", "synthetic cadastre — nobody's land", date(-40)}, "water": {"demo/water.json", "", ""}} {
			b, err := demoFS.ReadFile(src.file)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO map_files (name, bytes, source, snapshot) VALUES (?,?,?,?)`, name, b, src.source, src.snap); err != nil {
				return err
			}
		}
		// The codex, adopted by the council on a day, no house.
		for i, sec := range seed.Codex.Sections {
			sid, err := tx.Exec(ctx, `INSERT INTO codex_sections(ord, updated_at, updated_by) VALUES (?,?,NULL)`, i+1, date(-seed.Codex.Adopted)+" 12:00:00")
			if err != nil {
				return err
			}
			if err := writeTexts(ctx, tx, sid, sec.Texts); err != nil {
				return err
			}
			counts["codex sections"]++
		}
		for _, p := range seed.Posts {
			pid, err := tx.Exec(ctx, `INSERT INTO posts(house_id, author, body, pinned, created_at) VALUES (?,?,?,?,?)`, id(p.House), p.Author, p.Body, b2i(p.Pinned), stamp(p.Day, p.Time))
			if err != nil {
				return err
			}
			counts["posts"]++
			for _, r := range p.Replies {
				if _, err := tx.Exec(ctx, `INSERT INTO posts(house_id, parent_id, author, body, created_at) VALUES (?,?,?,?,?)`, id(r.House), pid, r.Author, r.Body, stamp(r.Day, r.Time)); err != nil {
					return err
				}
				counts["posts"]++
			}
		}
		projects := map[string]int64{}
		for _, p := range seed.Projects {
			state := p.State
			if state == "" {
				state = "planned"
			}
			var done any
			if state == "done" {
				done = stamp(p.DoneDay, "17:00")
			}
			pid, err := tx.Exec(ctx, `INSERT INTO projects(house_id, title, notes, state, done_at, created_at) VALUES (?,?,?,?,?,?)`, id(p.House), p.Title, p.Notes, state, done, stamp(p.Day, "20:00"))
			if err != nil {
				return err
			}
			projects[p.Key] = pid
			counts["projects"]++
			for _, t := range p.Tasks {
				ts := t.State
				if ts == "" {
					ts = "open"
				}
				var tdone any
				if ts == "done" {
					tdone = stamp(t.DoneDay, "16:00")
				}
				if _, err := tx.Exec(ctx, `INSERT INTO project_tasks(project_id, house_id, title, notes, assigned_to, state, done_at, closing_note, created_at) VALUES (?,?,?,?,?,?,?,?,?)`, pid, id(t.House), t.Title, t.Notes, id(t.AssignedTo), ts, tdone, t.ClosingNote, stamp(t.Day, "20:10")); err != nil {
					return err
				}
				counts["tasks"]++
			}
		}
		for _, e := range seed.Events {
			kind := e.Kind
			if kind == "" {
				kind = "event"
			}
			var end any
			if e.End != "" {
				end = wall(e.Day, e.End)
			}
			var proj any
			if e.Project != "" {
				proj = projects[e.Project]
			}
			eid, err := tx.Exec(ctx, `INSERT INTO events(house_id, title, kind, starts_at, ends_at, place, notes, project_id, created_at) VALUES (?,?,?,?,?,?,?,?,?)`, id(e.House), e.Title, kind, wall(e.Day, e.Start), end, e.Place, e.Notes, proj, stamp(e.Day-9, "19:30"))
			if err != nil {
				return err
			}
			counts["events"]++
			// The caller answers yes in the same breath, like the app does.
			answers := map[string]string{e.House: "yes"}
			for k, v := range e.Answers {
				answers[k] = v
			}
			for k, v := range answers {
				if _, err := tx.Exec(ctx, `INSERT INTO event_signups(event_id, house_id, state, answered_at, answered_version) VALUES (?,?,?,?,0)`, eid, id(k), v, stamp(e.Day-7, "21:00")); err != nil {
					return err
				}
			}
			if err := seedComments(ctx, tx, "event", eid, e.Comments, id, stamp); err != nil {
				return err
			}
		}
		for _, r := range seed.Runs {
			rid, err := tx.Exec(ctx, `INSERT INTO runs(house_id, destination, cutoff_at, notes, created_at) VALUES (?,?,?,?,?)`, id(r.House), r.Destination, wall(r.Day, r.Time), r.Notes, stamp(r.Day-2, "08:15"))
			if err != nil {
				return err
			}
			counts["runs"]++
			for _, n := range r.Needs {
				if _, err := tx.Exec(ctx, `INSERT INTO needs(house_id, text, state, taken_by, run_id, created_at) VALUES (?,?,?,?,?,?)`, id(n.House), n.Text, orDefault(n.State, "open"), id(n.TakenBy), rid, stamp(n.Day, "09:40")); err != nil {
					return err
				}
				counts["needs"]++
			}
		}
		for _, n := range seed.Needs {
			if _, err := tx.Exec(ctx, `INSERT INTO needs(house_id, text, state, taken_by, created_at) VALUES (?,?,?,?,?)`, id(n.House), n.Text, orDefault(n.State, "open"), id(n.TakenBy), stamp(n.Day, "11:20")); err != nil {
				return err
			}
			counts["needs"]++
		}
		for _, o := range seed.Offers {
			if _, err := tx.Exec(ctx, `INSERT INTO offers(house_id, text, tag, state, claimed_by, created_at) VALUES (?,?,?,?,?,?)`, id(o.House), o.Text, orDefault(o.Tag, "giveaway"), orDefault(o.State, "open"), id(o.ClaimedBy), stamp(o.Day, "17:50")); err != nil {
				return err
			}
			counts["offers"]++
		}
		for _, a := range seed.Away {
			aid, err := tx.Exec(ctx, `INSERT INTO away(house_id, from_date, to_date, notes, watcher, created_at) VALUES (?,?,?,?,?,?)`, id(a.House), date(a.From), date(a.To), a.Notes, id(a.Watcher), stamp(a.From-5, "20:30"))
			if err != nil {
				return err
			}
			counts["away"]++
			if err := seedComments(ctx, tx, "away", aid, a.Comments, id, stamp); err != nil {
				return err
			}
		}
		for _, t := range seed.Tools {
			var held, since any
			if t.HeldBy != "" {
				held, since = id(t.HeldBy), stamp(-t.HeldDays, "16:00")
			}
			if _, err := tx.Exec(ctx, `INSERT INTO tools(house_id, name, notes, category, held_by, held_since, created_at) VALUES (?,?,?,?,?,?,?)`, id(t.House), t.Name, t.Notes, orDefault(t.Category, "other"), held, since, stamp(-300, "12:00")); err != nil {
				return err
			}
			counts["tools"]++
		}
		for _, w := range seed.Wishes {
			wid, err := tx.Exec(ctx, `INSERT INTO wishes(house_id, text, created_at) VALUES (?,?,?)`, id(w.House), w.Text, stamp(w.Day, "21:15"))
			if err != nil {
				return err
			}
			counts["wishes"]++
			for _, k := range w.Wants {
				if _, err := tx.Exec(ctx, `INSERT INTO wish_wants(wish_id, house_id) VALUES (?,?)`, wid, id(k)); err != nil {
					return err
				}
			}
			for _, o := range w.Options {
				if _, err := tx.Exec(ctx, `INSERT INTO wish_options(wish_id, house_id, text, url, created_at) VALUES (?,?,?,?,?)`, wid, id(o.House), o.Text, o.URL, stamp(w.Day+2, "22:00")); err != nil {
					return err
				}
			}
			if err := seedComments(ctx, tx, "wish", wid, w.Comments, id, stamp); err != nil {
				return err
			}
		}
		for _, c := range seed.Camp {
			state := orDefault(c.State, "arrived")
			var held, heldAt, handed any
			if state == "held" || state == "handed" {
				held, heldAt = id(orDefault(c.HeldBy, c.House)), stamp(c.Day, "18:00")
			}
			if state == "handed" {
				handed = stamp(c.Day+2, "19:00")
			}
			if _, err := tx.Exec(ctx, `INSERT INTO camp_takings(house_id, from_who, taken_on, notes, state, held_by, held_at, handed_at, created_at) VALUES (?,?,?,?,?,?,?,?,?)`, id(c.House), c.FromWho, date(c.Day), c.Notes, state, held, heldAt, handed, stamp(c.Day, "17:30")); err != nil {
				return err
			}
			counts["camp rows"]++
		}
		for _, c := range seed.Contacts {
			if _, err := tx.Exec(ctx, `INSERT INTO contacts(house_id, name, phone, notes, type, created_at) VALUES (?,?,?,?,?,?)`, id(c.House), c.Name, c.Phone, c.Notes, c.Type, stamp(-120, "13:00")); err != nil {
				return err
			}
			counts["contacts"]++
		}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("demo village seeded: %v\n", counts)
	return nil
}

func seedComments(ctx context.Context, tx *store.Store, subject string, sid int64, cs []demoComment, id func(string) any, stamp func(int, string) string) error {
	for _, c := range cs {
		if _, err := tx.Exec(ctx, `INSERT INTO comments(subject, subject_id, house_id, author, body, created_at) VALUES (?,?,?,?,?,?)`, subject, sid, id(c.House), c.Author, c.Body, stamp(c.Day, c.Time)); err != nil {
			return err
		}
	}
	return nil
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
