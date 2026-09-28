package httpapi

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
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

// The seed's shape. A top-level row is dated one of three ways:
//
//   - "day" alone is an offset from the seed's day (negative = past), for
//     what could happen in any month: the boar at the bins, a lift to town.
//   - "on" is a calendar anchor, "MM-DD", for what the land decides — the
//     olive harvest, the frost, São João — so a demo reseeded in March does
//     not pick grapes. With "day" beside it, the row sits that many days off
//     the anchor, and the pair resolves as one date. It resolves to its most
//     recent occurrence, or with "ahead" to its next; "keep" lets an
//     occurrence stay that many days after it passed, so a harvest just done
//     keeps its thread for a week before the next year's takes its place.
//   - "at" names a keyed project or event, and the row is "day" days off it,
//     so a post announcing the grape harvest moves with the harvest.
//
// A row inside another — a reply, a comment, a need on a run, a task, a
// wish's finding — has only "day", and it counts from its parent's day.
//
// Words are never written in the future: a post, comment, need or task whose
// moment has not come yet is not seeded, and neither is what hangs off it. A
// work party eleven months ahead therefore has no thread yet, and the one
// three days ahead has the lines people wrote before it. A row that is a
// container — an event, a run, a project, an absence — is seeded whenever it
// falls, with the moment it was created pulled back into the past. Clocks are
// the village's wall clock.
//
// The seam this leaves: an anchored absence does not silence its house's
// rows dated off today, so in the fortnight around the year end the house
// in France for the holidays also thanks Amarela for last night's film.
// TestDemoAnyDay proves every room shows its features on any day; it does
// not prove every day is coherent.
type demoWhen struct {
	Day   int
	On    string
	Ahead bool
	Keep  int
	At    string
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
			// A section a house rewrote after the council adopted it: which
			// house, and how many days ago.
			EditedBy string
			Edited   int
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
		Key, House, Kind, Title, Place, Notes string
		Project, Task                         string // a project's key, and one of its tasks by title
		Start, End                            string
		Answers                               map[string]string // house key -> yes/no/maybe
		// A house that edited the event after it was called. With "moved",
		// the edit moved its time that many days off the event, and the
		// answers in "stale" were given before it — the room strikes them.
		EditedBy string
		Moved    int
		Stale    map[string]string
		Comments []demoComment
	}
	Runs []struct {
		demoWhen
		House, Kind, Destination, Notes string // kind: trip (a car) or order (a delivery)
		Needs                           []demoNeed
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
		Comments              []demoComment // days count from the first day away
	}
	Tools []struct {
		House, Name, Notes, Category, HeldBy string
		HeldDays                             int // days on loan; the nudge reads off held_since
	}
	Wishes []struct {
		demoWhen
		House, Text string
		Wants       []string
		Options     []struct {
			Day              int
			House, Text, URL string
		}
		Comments []demoComment
	}
	Projects []struct {
		demoWhen
		Key, House, Title, Notes, State string
		Done, Due                       int // days off the project's day; 0 = not done, no due date
		Tasks                           []struct {
			Day                                          int
			House, Title, Notes, AssignedTo, ClosingNote string
			Done, Due                                    int // days off the project's day
		}
	}
	Camp []struct {
		demoWhen
		House, Notes, State, HeldBy string
	}
	// A due a steward wrote down, the house that collects it, and who has
	// paid: "paid" is days after the due was written, absent for not yet.
	// In a title, "{trimestre}" and "{ano}" become the quarter and the year
	// the seed runs in, "{trimestre passado}" the quarter before, so the
	// water of the third quarter is not asked for in March.
	Dues []struct {
		demoWhen
		House, Collector, Title, Notes, Currency string
		Amount                                   int // cents; 0 = no amount
		Due                                      int // days after it was written; 0 = no date
		Owed                                     []struct {
			House, PaidBy string // PaidBy defaults to the collector
			Paid          *int
		}
	}
	// Duties the steward wrote down a year ago, and the turns houses took on
	// them, each dated like a top-level row: a done turn whose day has not
	// come, or a planned one whose day has passed, is not seeded.
	Duties []struct {
		House, Title, Notes string
		Every               int // days; 0 = when needed
		Turns               []struct {
			demoWhen
			House, State, Note string
		}
	}
	Contacts []struct {
		demoWhen
		House, Name, Phone, Notes, Type string
		EditedBy                        string
		Edited                          int // days after it was written down
		Comments                        []demoComment
	}
}

type demoNeed struct {
	demoWhen
	House, Text, State, TakenBy string
}

// A comment, a reply to a post, or a reply to a comment (one level, as the
// thread has). Its day counts from its parent's.
type demoComment struct {
	Day                 int
	Time                string
	House, Author, Body string
	Replies             []demoComment
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
	dec := json.NewDecoder(bytes.NewReader(raw))
	// A misspelt field would otherwise be dropped without a word, and the
	// row would seed without the thing it was written to show.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&seed); err != nil {
		return nil, fmt.Errorf("demo/seed.json: %w", err)
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	// Calendar days between two dates, counted in UTC so a clock change
	// between them is not a day lost or gained.
	daysFrom := func(t time.Time) int {
		u := func(x time.Time) time.Time { return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, time.UTC) }
		return int(u(t).Sub(u(today)).Hours() / 24)
	}
	parse := func(on string, year int) time.Time {
		t, err := time.ParseInLocation("2006-01-02", fmt.Sprintf("%d-%s", year, on), now.Location())
		if err != nil {
			panic("demo seed: bad anchor " + on)
		}
		return t
	}
	// keyed holds the day of every project and event with a key, for the
	// rows dated "at" one.
	keyed := map[string]int{}
	// dayOf resolves a top-level row's date to an offset from today.
	dayOf := func(w demoWhen) int {
		if w.At != "" {
			base, ok := keyed[w.At]
			if !ok {
				panic("demo seed: a row is dated at " + w.At + ", which no project or event before it is keyed as")
			}
			return base + w.Day
		}
		if w.On == "" {
			return w.Day
		}
		at := func(y int) int { return daysFrom(parse(w.On, y).AddDate(0, 0, w.Day)) }
		if w.Ahead {
			for y := now.Year() - 1; ; y++ {
				if d := at(y); d >= -w.Keep {
					return d
				}
			}
		}
		for y := now.Year() + 1; ; y-- {
			if d := at(y); d <= 0 {
				return d
			}
		}
	}
	// A row inside another counts from its parent, and nothing else.
	child := func(w demoWhen) int {
		if w.On != "" || w.At != "" || w.Ahead || w.Keep != 0 {
			panic("demo seed: a row inside another is dated by its day alone")
		}
		return w.Day
	}
	anchor := func(on string, ahead bool) int { return dayOf(demoWhen{On: on, Ahead: ahead}) }
	// span resolves an away's two ends as one thing: one end by its anchor,
	// the other in that end's year, so a span that has already begun is not
	// thrown a year ahead by its own start date. For a span, "ahead" means
	// not yet over (the far end anchors), else already begun (the near end).
	// The one-year step covers a span across the year end in either
	// direction; the two holiday absences in the seed cross it, and
	// TestDemoSeed pins them from September and from the week after the year end.
	span := func(from, to int, onFrom, onTo string, ahead bool) (int, int) {
		if onFrom == "" && onTo == "" {
			return from, to
		}
		if onFrom == "" || onTo == "" {
			panic("demo seed: an away anchors both ends or neither")
		}
		if ahead {
			t := anchor(onTo, true)
			f := parse(onFrom, today.AddDate(0, 0, t).Year())
			if daysFrom(f) > t {
				f = f.AddDate(-1, 0, 0)
			}
			return daysFrom(f), t
		}
		f := anchor(onFrom, false)
		t := parse(onTo, today.AddDate(0, 0, f).Year())
		if daysFrom(t) < f {
			t = t.AddDate(1, 0, 0)
		}
		return f, daysFrom(t)
	}
	// Two clocks: a wall-clock string for what a villager typed, a UTC stamp
	// with a space for what the database would have written itself.
	wall := func(day int, clock string) string {
		if clock == "" {
			clock = "12:00"
		}
		return today.AddDate(0, 0, day).Format("2006-01-02") + "T" + clock
	}
	date := func(day int) string { return today.AddDate(0, 0, day).Format("2006-01-02") }
	moment := func(day int, clock string) time.Time {
		t, err := time.ParseInLocation("2006-01-02T15:04", wall(day, clock), now.Location())
		if err != nil {
			panic("demo seed: bad time " + clock)
		}
		return t
	}
	utc := func(t time.Time) string { return t.UTC().Format("2006-01-02 15:04:05") }
	// written is the stamp of something a villager wrote, and false when its
	// moment has not come yet: such a row is not seeded.
	written := func(day int, clock string) (string, bool) {
		t := moment(day, clock)
		return utc(t), !t.After(now)
	}
	// past is the stamp of something the database recorded — a row created,
	// an answer given — pulled back to half an hour ago when it would fall
	// later, because a container is seeded whenever it falls.
	past := func(day int, clock string) string {
		t := moment(day, clock)
		if lim := now.Add(-30 * time.Minute); t.After(lim) {
			t = lim
		}
		return utc(t)
	}
	// after refuses a row that precedes what it hangs off: a reply before
	// its post reads as a village that answers questions nobody asked.
	after := func(what, s, parent string) {
		if s < parent {
			panic(fmt.Sprintf("demo seed: %s at %s comes before what it answers (%s)", what, s, parent))
		}
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
	// comments writes a thread off its subject's day, replies one level down.
	var comments func(subject string, sid int64, base int, parentStamp string, parent any, cs []demoComment) error
	comments = func(subject string, sid int64, base int, parentStamp string, parent any, cs []demoComment) error {
		for _, c := range cs {
			day := base + c.Day
			s, ok := written(day, orDefault(c.Time, "20:00"))
			if !ok {
				continue
			}
			after(subject+" comment", s, parentStamp)
			cid, err := tx.Exec(ctx, `INSERT INTO comments(subject, subject_id, house_id, parent_id, author, body, created_at) VALUES (?,?,?,?,?,?,?)`, subject, sid, id(c.House), parent, c.Author, c.Body, s)
			if err != nil {
				return err
			}
			counts["comments"]++
			if len(c.Replies) > 0 {
				if parent != nil {
					panic("demo seed: a thread has one reply level")
				}
				if err := comments(subject, sid, day, s, cid, c.Replies); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, h := range seed.Houses {
		kind, crest, color := orDefault(h.Kind, "house"), orDefault(h.Crest, "🏠"), orDefault(h.Color, "#b5651d")
		hid, err := tx.Exec(ctx, `INSERT INTO houses(name, crest, color, kind, is_steward, created_at) VALUES (?,?,?,?,?,?)`, h.Name, crest, color, kind, b2i(h.Steward), past(-400, "10:00"))
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
	// The demo shows every room, so every switch is on, and stays on: the
	// demo server refuses a flip (demoLocked) and the settings page greys
	// the switches out. The steward switched them on a year ago, so the
	// settings page reads "switched on by", not "had it before switches".
	var steward any
	for _, h := range seed.Houses {
		if h.Steward {
			steward = houses[h.Key]
		}
	}
	for _, f := range Features {
		if _, err := tx.Exec(ctx, `INSERT INTO features(key, enabled, set_by, set_at) VALUES (?, 1, ?, ?)`, f.Key, steward, past(-365, "21:00")); err != nil {
			return nil, err
		}
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
	// The codex, adopted by the council on a day, no house; a section a
	// house rewrote since names that house and that day.
	for i, sec := range seed.Codex.Sections {
		stamp, by, rev := date(-seed.Codex.Adopted)+" 12:00:00", any(nil), 0
		if sec.EditedBy != "" {
			stamp, by, rev = past(-sec.Edited, "21:40"), id(sec.EditedBy), 1
		}
		sid, err := tx.Exec(ctx, `INSERT INTO codex_sections(ord, rev, updated_at, updated_by) VALUES (?,?,?,?)`, i+1, rev, stamp, by)
		if err != nil {
			return nil, err
		}
		if err := writeTexts(ctx, tx, sid, sec.Texts); err != nil {
			return nil, err
		}
		counts["codex sections"]++
	}
	// Projects before events, and both before anything dated "at" them.
	for _, p := range seed.Projects {
		if p.Key != "" {
			keyed[p.Key] = dayOf(p.demoWhen)
		}
	}
	for _, e := range seed.Events {
		if e.Key != "" {
			keyed[e.Key] = dayOf(e.demoWhen)
		}
	}
	projects, tasks, named := map[string]int64{}, map[string]int64{}, map[string]bool{}
	for _, p := range seed.Projects {
		day := dayOf(p.demoWhen)
		created := past(day, "20:00")
		// A project marked done whose day has not come is still in progress.
		state := orDefault(p.State, "planned")
		var done, due any
		if state == "done" {
			if s, ok := written(day+p.Done, "17:00"); ok {
				done = s
			} else {
				state = "open"
			}
		}
		if p.Due != 0 {
			due = date(day + p.Due)
		}
		pid, err := tx.Exec(ctx, `INSERT INTO projects(house_id, title, notes, state, due_at, done_at, created_at) VALUES (?,?,?,?,?,?,?)`, id(p.House), p.Title, p.Notes, state, due, done, created)
		if err != nil {
			return nil, err
		}
		projects[p.Key] = pid
		counts["projects"]++
		for _, t := range p.Tasks {
			named[p.Key+"/"+t.Title] = true
			ts, ok := written(day+t.Day, "20:10")
			if !ok {
				continue
			}
			after("task", ts, created)
			state, closing := "open", ""
			var tdone, tdue any
			if t.Done != 0 {
				if s, ok := written(day+t.Done, "16:00"); ok {
					after("a task done", s, ts)
					state, closing, tdone = "done", t.ClosingNote, s
				}
			}
			if t.Due != 0 {
				tdue = date(day + t.Due)
			}
			tid, err := tx.Exec(ctx, `INSERT INTO project_tasks(project_id, house_id, title, notes, assigned_to, due_at, state, done_at, closing_note, created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, pid, id(t.House), t.Title, t.Notes, id(t.AssignedTo), tdue, state, tdone, closing, ts)
			if err != nil {
				return nil, err
			}
			tasks[p.Key+"/"+t.Title] = tid
			counts["tasks"]++
		}
	}
	for _, e := range seed.Events {
		day := dayOf(e.demoWhen)
		var end any
		if e.End != "" {
			end = wall(day, e.End)
		}
		var proj, task any
		if e.Project != "" {
			if _, ok := projects[e.Project]; !ok {
				panic("demo seed: an event belongs to a project it never defined: " + e.Project)
			}
			proj = projects[e.Project]
		}
		if e.Task != "" {
			if !named[e.Project+"/"+e.Task] {
				panic("demo seed: an event belongs to a task its project does not have: " + e.Task)
			}
			// A task not written yet leaves the event on the project alone.
			if v, ok := tasks[e.Project+"/"+e.Task]; ok {
				task = v
			}
		}
		// Called nine days before, or before the first line written on it.
		createdDay := day - 9
		for _, c := range e.Comments {
			if d := day + c.Day - 1; d < createdDay {
				createdDay = d
			}
		}
		if e.Moved != 0 && day+e.Moved-1 < createdDay {
			createdDay = day + e.Moved - 1
		}
		created := past(createdDay, "19:30")
		var editedBy, editedAt, changedAt any
		version := 0
		if e.EditedBy != "" {
			editedBy, editedAt = id(e.EditedBy), past(createdDay+2, "10:00")
			if e.Moved != 0 {
				editedAt, changedAt, version = past(day+e.Moved, "10:00"), past(day+e.Moved, "10:00"), 1
			}
		} else if e.Moved != 0 || len(e.Stale) > 0 {
			panic("demo seed: an event whose time moved names the house that moved it: " + e.Title)
		}
		eid, err := tx.Exec(ctx, `INSERT INTO events(house_id, title, kind, starts_at, ends_at, place, notes, project_id, task_id, edited_by, edited_at, time_changed_at, time_version, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id(e.House), e.Title, orDefault(e.Kind, "event"), wall(day, e.Start), end, e.Place, e.Notes, proj, task, editedBy, editedAt, changedAt, version, created)
		if err != nil {
			return nil, err
		}
		counts["events"]++
		// The caller answers yes in the same breath, like the app does. An
		// answer given before the time moved is stale; one given after is not.
		type answer struct {
			state, at string
			version   int
		}
		answers := map[string]answer{}
		fresh := past(createdDay+2, "21:00")
		if e.Moved != 0 {
			fresh = past(day+e.Moved+1, "21:00")
		}
		answers[e.House] = answer{"yes", created, 0}
		for k, v := range e.Stale {
			answers[k] = answer{v, past(createdDay, "21:00"), 0}
		}
		for k, v := range e.Answers {
			answers[k] = answer{v, fresh, version}
		}
		for k, a := range answers {
			if _, err := tx.Exec(ctx, `INSERT INTO event_signups(event_id, house_id, state, answered_at, answered_version) VALUES (?,?,?,?,?)`, eid, id(k), a.state, a.at, a.version); err != nil {
				return nil, err
			}
		}
		if err := comments("event", eid, day, created, nil, e.Comments); err != nil {
			return nil, err
		}
	}
	for _, p := range seed.Posts {
		day := dayOf(p.demoWhen)
		s, ok := written(day, p.Time)
		if !ok {
			continue
		}
		pid, err := tx.Exec(ctx, `INSERT INTO posts(house_id, author, body, pinned, created_at) VALUES (?,?,?,?,?)`, id(p.House), p.Author, p.Body, b2i(p.Pinned), s)
		if err != nil {
			return nil, err
		}
		counts["posts"]++
		for _, r := range p.Replies {
			if len(r.Replies) > 0 {
				panic("demo seed: the board has one reply level")
			}
			rs, ok := written(day+r.Day, r.Time)
			if !ok {
				continue
			}
			after("reply", rs, s)
			if _, err := tx.Exec(ctx, `INSERT INTO posts(house_id, parent_id, author, body, created_at) VALUES (?,?,?,?,?)`, id(r.House), pid, r.Author, r.Body, rs); err != nil {
				return nil, err
			}
			counts["posts"]++
		}
	}
	for _, r := range seed.Runs {
		day := dayOf(r.demoWhen)
		createdDay := day - 2
		for _, n := range r.Needs {
			if d := day + child(n.demoWhen); d < createdDay {
				createdDay = d
			}
		}
		created := past(createdDay, "08:15")
		rid, err := tx.Exec(ctx, `INSERT INTO runs(house_id, kind, destination, cutoff_at, notes, created_at) VALUES (?,?,?,?,?,?)`, id(r.House), orDefault(r.Kind, "trip"), r.Destination, wall(day, r.Time), r.Notes, created)
		if err != nil {
			return nil, err
		}
		counts["runs"]++
		for _, n := range r.Needs {
			s, ok := written(day+child(n.demoWhen), orDefault(n.Time, "09:40"))
			if !ok {
				continue
			}
			after("need", s, created)
			if _, err := tx.Exec(ctx, `INSERT INTO needs(house_id, text, state, taken_by, run_id, created_at) VALUES (?,?,?,?,?,?)`, id(n.House), n.Text, orDefault(n.State, "open"), id(n.TakenBy), rid, s); err != nil {
				return nil, err
			}
			counts["needs"]++
		}
	}
	for _, n := range seed.Needs {
		s, ok := written(dayOf(n.demoWhen), orDefault(n.Time, "11:20"))
		if !ok {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO needs(house_id, text, state, taken_by, created_at) VALUES (?,?,?,?,?)`, id(n.House), n.Text, orDefault(n.State, "open"), id(n.TakenBy), s); err != nil {
			return nil, err
		}
		counts["needs"]++
	}
	for _, o := range seed.Offers {
		s, ok := written(dayOf(o.demoWhen), orDefault(o.Time, "17:50"))
		if !ok {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO offers(house_id, text, tag, state, claimed_by, created_at) VALUES (?,?,?,?,?,?)`, id(o.House), o.Text, orDefault(o.Tag, "giveaway"), orDefault(o.State, "open"), id(o.ClaimedBy), s); err != nil {
			return nil, err
		}
		counts["offers"]++
	}
	for _, a := range seed.Away {
		from, to := span(a.From, a.To, a.OnFrom, a.OnTo, a.Ahead)
		created := past(from-5, "20:30")
		aid, err := tx.Exec(ctx, `INSERT INTO away(house_id, from_date, to_date, notes, watcher, created_at) VALUES (?,?,?,?,?,?)`, id(a.House), date(from), date(to), a.Notes, id(a.Watcher), created)
		if err != nil {
			return nil, err
		}
		counts["away"]++
		if err := comments("away", aid, from, created, nil, a.Comments); err != nil {
			return nil, err
		}
	}
	for _, t := range seed.Tools {
		// held_since is a date, the way the room's "I take it" writes it.
		var held, since any
		if t.HeldBy != "" {
			held, since = id(t.HeldBy), date(-t.HeldDays)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO tools(house_id, name, notes, category, held_by, held_since, created_at) VALUES (?,?,?,?,?,?,?)`, id(t.House), t.Name, t.Notes, orDefault(t.Category, "other"), held, since, past(-300, "12:00")); err != nil {
			return nil, err
		}
		counts["tools"]++
	}
	for _, w := range seed.Wishes {
		day := dayOf(w.demoWhen)
		s, ok := written(day, orDefault(w.Time, "21:15"))
		if !ok {
			continue
		}
		wid, err := tx.Exec(ctx, `INSERT INTO wishes(house_id, text, created_at) VALUES (?,?,?)`, id(w.House), w.Text, s)
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
			os, ok := written(day+o.Day, "22:00")
			if !ok {
				continue
			}
			after("a wish's finding", os, s)
			if _, err := tx.Exec(ctx, `INSERT INTO wish_options(wish_id, house_id, text, url, created_at) VALUES (?,?,?,?,?)`, wid, id(o.House), o.Text, o.URL, os); err != nil {
				return nil, err
			}
		}
		if err := comments("wish", wid, day, s, nil, w.Comments); err != nil {
			return nil, err
		}
	}
	for _, c := range seed.Camp {
		day := dayOf(c.demoWhen)
		s, ok := written(day, orDefault(c.Time, "17:30"))
		if !ok {
			continue
		}
		// The room writes the camper into the note and leaves from_who empty,
		// so the demo does too: no amount, no plate, no nationality.
		state := orDefault(c.State, "arrived")
		var held, heldAt, handed any
		if state == "held" || state == "handed" {
			held, heldAt = id(orDefault(c.HeldBy, c.House)), past(day, "18:00")
		}
		if state == "handed" {
			// Two days to reach the box; a stay that has not had them is
			// still held.
			if hs, ok := written(day+2, "19:00"); ok {
				handed = hs
			} else {
				state = "held"
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO camp_takings(house_id, from_who, taken_on, notes, state, held_by, held_at, handed_at, created_at) VALUES (?,'',?,?,?,?,?,?,?)`, id(c.House), date(day), c.Notes, state, held, heldAt, handed, s); err != nil {
			return nil, err
		}
		counts["camp rows"]++
	}
	for _, d := range seed.Dues {
		day := dayOf(d.demoWhen)
		s, ok := written(day, orDefault(d.Time, "10:00"))
		if !ok {
			continue
		}
		q := (int(today.Month())-1)/3 + 1
		title := strings.NewReplacer("{trimestre passado}", fmt.Sprintf("%d.º", (q+2)%4+1), "{trimestre}", fmt.Sprintf("%d.º", q), "{ano}", fmt.Sprint(today.Year())).Replace(d.Title)
		var amount, dueOn any
		if d.Amount > 0 {
			amount = d.Amount
		}
		if d.Due != 0 {
			dueOn = date(day + d.Due)
		}
		did, err := tx.Exec(ctx, `INSERT INTO dues(title, amount_cents, currency, due_on, notes, collector_id, house_id, created_at) VALUES (?,?,?,?,?,?,?,?)`, title, amount, d.Currency, dueOn, d.Notes, id(d.Collector), id(d.House), s)
		if err != nil {
			return nil, err
		}
		counts["dues"]++
		for _, o := range d.Owed {
			var paidAt, paidBy any
			if o.Paid != nil {
				if ps, ok := written(day+*o.Paid, "19:00"); ok {
					after("a due paid", ps, s)
					paidAt, paidBy = ps, id(orDefault(o.PaidBy, d.Collector))
				}
			}
			if _, err := tx.Exec(ctx, `INSERT INTO dues_owed(due_id, house_id, paid_at, paid_by) VALUES (?,?,?,?)`, did, id(o.House), paidAt, paidBy); err != nil {
				return nil, err
			}
		}
	}
	for _, d := range seed.Duties {
		var every any
		if d.Every > 0 {
			every = d.Every
		}
		did, err := tx.Exec(ctx, `INSERT INTO roster_duties(house_id, title, notes, every_days, created_at) VALUES (?,?,?,?,?)`, id(d.House), d.Title, d.Notes, every, past(-380, "21:00"))
		if err != nil {
			return nil, err
		}
		counts["duties"]++
		for _, t := range d.Turns {
			day := dayOf(t.demoWhen)
			state := orDefault(t.State, "done")
			var doneAt any
			created := past(-1, "20:00")
			switch state {
			case "done":
				s, ok := written(day, orDefault(t.Time, "18:00"))
				if !ok {
					continue
				}
				doneAt, created = s, s
			case "planned":
				if day < 0 {
					continue
				}
			default:
				panic("demo seed: a turn is done or planned")
			}
			if _, err := tx.Exec(ctx, `INSERT INTO roster_turns(duty_id, house_id, on_date, state, note, done_at, created_at) VALUES (?,?,?,?,?,?,?)`, did, id(t.House), date(day), state, t.Note, doneAt, created); err != nil {
				return nil, fmt.Errorf("demo seed: %s on %s: %w", d.Title, date(day), err)
			}
			counts["turns"]++
		}
	}
	for _, c := range seed.Contacts {
		day := dayOf(c.demoWhen)
		s, ok := written(day, orDefault(c.Time, "13:00"))
		if !ok {
			continue
		}
		var editedBy, editedAt any
		if c.EditedBy != "" {
			if es, ok := written(day+c.Edited, "20:00"); ok {
				after("a contact's correction", es, s)
				editedBy, editedAt = id(c.EditedBy), es
			}
		}
		cid, err := tx.Exec(ctx, `INSERT INTO contacts(house_id, name, phone, notes, type, created_at, edited_by, edited_at) VALUES (?,?,?,?,?,?,?,?)`, id(c.House), c.Name, c.Phone, c.Notes, c.Type, s, editedBy, editedAt)
		if err != nil {
			return nil, err
		}
		counts["contacts"]++
		if err := comments("contact", cid, day, s, nil, c.Comments); err != nil {
			return nil, err
		}
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
