package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/dennislapchenko/porta-pagi/backend/internal/store"
)

// Duty roster, behind the `roster` switch: the village's recurring chores —
// mow the common ground, clear the lane, check the pump — and the turns each
// house took. One of the two places the portal counts, and only because a
// village switched it on. Design and the options rejected:
// docs/design-more-rooms.md § Duty roster.
//
// Three rules hold it:
//
//   - A steward writes the duties down. What is counted is an agreement the
//     village made in person; a house that could add a duty could start
//     counting its neighbours on a chore nobody agreed to.
//   - A house writes only its own turns, and nobody else touches them — not a
//     steward either. A count is the house's own word about itself.
//   - The count is done turns per house per duty over the last twelve months,
//     one number per cell and no sum across duties: a pump check is not a
//     mowing. Houses are listed in the village's order, never by count.
//
// Nothing here pushes, and nothing assigns a house to a date: the portal
// shows the village's agreement, it does not run it.

const rosterWindow = "the last twelve months"

func (s *Server) today() string { return s.now().Format("2006-01-02") }

// isDay: a plain YYYY-MM-DD that is a real day.
func isDay(v string) bool {
	_, err := time.Parse("2006-01-02", v)
	return err == nil && len(v) == 10
}

// listRoster: every duty with what the room shows under it, and the counts.
// A duty carries its last done turn and, if the village agreed a cadence,
// the day it is next due. Its turns are the plans from today on, the log of
// done turns inside the window, and the caller's own plans whose day has
// passed — so that house can say it happened or let it go. Another house's
// passed plan is nobody's business: a turn that did not happen is not
// counted, and it is not listed as missed either.
func (s *Server) listRoster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	me := houseFrom(r).ID
	today := s.today()
	since := s.now().AddDate(-1, 0, 0).Format("2006-01-02")
	duties, err := s.st.Rows(ctx, `SELECT d.*,
		(SELECT max(t.on_date) FROM roster_turns t WHERE t.duty_id=d.id AND t.state='done') AS last_done
		FROM roster_duties d ORDER BY d.retired_at IS NOT NULL, d.created_at, d.id`)
	if err != nil {
		fail(w, err)
		return
	}
	turns, err := s.st.Rows(ctx, `SELECT t.*, h.name AS house_name, h.crest AS house_crest, h.color AS house_color
		FROM roster_turns t LEFT JOIN houses h ON h.id=t.house_id
		WHERE (t.state='done' AND t.on_date >= ?)
		   OR (t.state='planned' AND t.house_id IS NOT NULL AND (t.on_date >= ? OR t.house_id = ?))
		ORDER BY t.state='done', CASE WHEN t.state='done' THEN '' ELSE t.on_date END, t.on_date DESC, t.id`, since, today, me)
	if err != nil {
		fail(w, err)
		return
	}
	byDuty := map[int64][]map[string]any{}
	for _, t := range turns {
		id := t["duty_id"].(int64)
		byDuty[id] = append(byDuty[id], t)
	}
	for _, d := range duties {
		id := d["id"].(int64)
		d["turns"] = append([]map[string]any{}, byDuty[id]...)
		d["next_due"] = nil
		// A duty nobody has done yet counts its first cadence from the day
		// the steward wrote it down: "the pump, every week" is due a week
		// after it was agreed, not never.
		from, _ := d["last_done"].(string)
		if from == "" {
			from, _ = d["created_at"].(string)
			from = from[:min(len(from), 10)]
		}
		if n, ok := d["every_days"].(int64); ok && n > 0 && from != "" {
			if day, err := time.Parse("2006-01-02", from); err == nil {
				d["next_due"] = day.AddDate(0, 0, int(n)).Format("2006-01-02")
			}
		}
	}
	// Only houses that exist are counted: a done turn of a house that has
	// left stays in the log and still moves "next due", but it has no row.
	counts, err := s.st.Rows(ctx, `SELECT t.duty_id, t.house_id, count(*) AS n FROM roster_turns t
		JOIN houses h ON h.id=t.house_id
		WHERE t.state='done' AND t.on_date >= ? GROUP BY t.duty_id, t.house_id`, since)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"today": today, "since": since, "window": rosterWindow, "duties": duties, "counts": counts})
}

// everyDays reads the cadence: a whole number of days from 1 to 366, or
// empty for "when needed". ok is false for anything else.
func everyDays(m map[string]any) (v any, ok bool) {
	switch x := m["every_days"].(type) {
	case nil:
		return nil, true
	case string:
		return nil, x == ""
	case float64:
		if x == 0 {
			return nil, true
		}
		if x != float64(int64(x)) || x < 1 || x > 366 {
			return nil, false
		}
		return int64(x), true
	}
	return nil, false
}

// createDuty: a steward writes down a duty the village agreed.
func (s *Server) createDuty(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil || str(m, "title") == "" {
		writeErr(w, 400, "title required")
		return
	}
	every, ok := everyDays(m)
	if !ok {
		writeErr(w, 400, "every_days is a whole number of days, 1 to 366, or empty")
		return
	}
	id, err := s.st.Exec(r.Context(), `INSERT INTO roster_duties(house_id, title, notes, every_days) VALUES (?,?,?,?)`,
		houseFrom(r).ID, str(m, "title"), str(m, "notes"), every)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"id": id})
}

// updateDuty: a steward edits the words or the cadence, or retires a duty
// (retired: true) and brings it back (false). A retired duty takes no new
// turns and leaves the count table; its log stays readable.
func (s *Server) updateDuty(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	row, err := s.st.One(r.Context(), `SELECT id FROM roster_duties WHERE id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if row == nil {
		writeErr(w, 404, "no such duty")
		return
	}
	// Every check before any write: a mixed body must not half-apply.
	if v, ok := m["title"]; ok {
		if title, _ := v.(string); title == "" {
			writeErr(w, 400, "a duty needs a title")
			return
		}
	}
	every, everyOK := everyDays(m)
	if _, ok := m["every_days"]; ok && !everyOK {
		writeErr(w, 400, "every_days is a whole number of days, 1 to 366, or empty")
		return
	}
	// One transaction: a duty is edited whole or not at all, and a failed
	// write is an error the steward sees, not a 204.
	if err := s.st.Tx(r.Context(), func(tx *store.Store) error {
		for _, k := range []string{"title", "notes"} {
			if _, ok := m[k]; ok {
				if _, err := tx.Exec(r.Context(), `UPDATE roster_duties SET `+k+`=? WHERE id=?`, str(m, k), id); err != nil {
					return err
				}
			}
		}
		if _, ok := m["every_days"]; ok {
			if _, err := tx.Exec(r.Context(), `UPDATE roster_duties SET every_days=? WHERE id=?`, every, id); err != nil {
				return err
			}
		}
		if retired, ok := m["retired"].(bool); ok {
			q := `UPDATE roster_duties SET retired_at=NULL WHERE id=?`
			if retired {
				q = `UPDATE roster_duties SET retired_at=COALESCE(retired_at, datetime('now')) WHERE id=?`
			}
			if _, err := tx.Exec(r.Context(), q, id); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}

// deleteDuty: a steward removes a duty written by mistake. Once anybody has
// done it, it is history and the count read it — retire it instead.
func (s *Server) deleteDuty(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	row, err := s.st.One(r.Context(), `SELECT (SELECT count(*) FROM roster_turns t WHERE t.duty_id=d.id AND t.state='done') AS done
		FROM roster_duties d WHERE d.id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if row == nil {
		writeErr(w, 404, "no such duty")
		return
	}
	if row["done"].(int64) > 0 {
		writeErr(w, 409, "somebody has done this duty; retire it instead")
		return
	}
	if _, err := s.st.Exec(r.Context(), `DELETE FROM roster_duties WHERE id=?`, id); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}

// createTurn: the caller's house takes a turn ahead ("I'll mow on
// Saturday", done=false, today or later) or records one it did (done=true,
// today or earlier). Always for itself: there is no house field to send.
func (s *Server) createTurn(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	did, err := pathID(r)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	d, err := s.st.One(r.Context(), `SELECT retired_at FROM roster_duties WHERE id=?`, did)
	if err != nil {
		fail(w, err)
		return
	}
	if d == nil {
		writeErr(w, 404, "no such duty")
		return
	}
	// Land does not take turns: a common place is not a household, and its
	// turn would count in no row of the table.
	if k, _ := s.st.One(r.Context(), `SELECT kind FROM houses WHERE id=?`, h.ID); k != nil && k["kind"] == "common" {
		writeErr(w, 403, "a common place takes no turns")
		return
	}
	if d["retired_at"] != nil {
		writeErr(w, 409, "this duty is retired")
		return
	}
	day := str(m, "on_date")
	if day == "" {
		day = s.today()
	}
	done, _ := m["done"].(bool)
	if msg := turnDayProblem(day, done, s.today()); msg != "" {
		writeErr(w, 400, msg)
		return
	}
	state, doneAt := "planned", any(nil)
	if done {
		state, doneAt = "done", s.now().UTC().Format("2006-01-02 15:04:05")
	}
	id, err := s.st.Exec(r.Context(), `INSERT INTO roster_turns(duty_id, house_id, on_date, state, note, done_at) VALUES (?,?,?,?,?,?)`,
		did, h.ID, day, state, str(m, "note"), doneAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeErr(w, 409, "your house already has a turn on this duty that day")
			return
		}
		fail(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"id": id})
}

// turnDayProblem: a done turn cannot be in the future, and a plan cannot be
// for a day that has passed — that is a turn to record as done. Both dates
// are local YYYY-MM-DD, so they compare as strings.
func turnDayProblem(day string, done bool, today string) string {
	if !isDay(day) {
		return "on_date is YYYY-MM-DD"
	}
	if done && day > today {
		return "a turn ahead cannot be done yet"
	}
	if !done && day < today {
		return "that day has passed: record the turn as done"
	}
	return ""
}

// updateTurn: only the turn's own house. done=true marks a plan done once
// its day has come; on_date moves the turn; note says how it went ("pressure
// low, called the plumber"). There is no done=false: a turn recorded by
// mistake is removed, not turned back into a plan for a day already gone.
func (s *Server) updateTurn(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	t, err := s.st.One(r.Context(), `SELECT t.house_id, t.on_date, t.state, d.retired_at FROM roster_turns t
		JOIN roster_duties d ON d.id=t.duty_id WHERE t.id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if t == nil {
		writeErr(w, 404, "no such turn")
		return
	}
	if owner, ok := t["house_id"].(int64); !ok || owner != h.ID {
		writeErr(w, 403, "not yours: a house writes only its own turns")
		return
	}
	day, done := t["on_date"].(string), t["state"] == "done"
	markDone, _ := m["done"].(bool)
	if v := str(m, "on_date"); v != "" || markDone {
		// A retired duty takes no turns, and a plan made before it was
		// retired does not become one: the table would show it the day the
		// steward brings the duty back.
		if t["retired_at"] != nil {
			writeErr(w, 409, "this duty is retired")
			return
		}
		if v != "" {
			day = v
		}
		done = done || markDone
		// A plan whose day passed may still have its note edited; only a
		// new day or a new state is checked.
		if msg := turnDayProblem(day, done, s.today()); msg != "" {
			writeErr(w, 400, msg)
			return
		}
	}
	state := "planned"
	if done {
		state = "done"
	}
	q := `UPDATE roster_turns SET on_date=?, state=?, done_at=CASE WHEN state=? THEN done_at ELSE ? END`
	args := []any{day, state, state, s.now().UTC().Format("2006-01-02 15:04:05")}
	if _, ok := m["note"]; ok {
		q += `, note=?`
		args = append(args, str(m, "note"))
	}
	if _, err := s.st.Exec(r.Context(), q+` WHERE id=?`, append(args, id)...); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeErr(w, 409, "your house already has a turn on this duty that day")
			return
		}
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}

// deleteTurn: only the turn's own house — letting a plan go, or taking back
// a turn recorded by mistake. A steward removes none: a count a steward could
// lower is a count nobody trusts.
func (s *Server) deleteTurn(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	t, err := s.st.One(r.Context(), `SELECT house_id FROM roster_turns WHERE id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if t == nil {
		writeErr(w, 404, "no such turn")
		return
	}
	if owner, ok := t["house_id"].(int64); !ok || owner != houseFrom(r).ID {
		writeErr(w, 403, "not yours: a house writes only its own turns")
		return
	}
	if _, err := s.st.Exec(r.Context(), `DELETE FROM roster_turns WHERE id=?`, id); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}
