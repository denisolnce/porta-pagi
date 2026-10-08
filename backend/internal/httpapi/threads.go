package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

// Comment threads, one implementation for every room that wants them: an event
// in the calendar, a wish in the tool shed, an away notice in the watchtower,
// a number in the phone book, a full moon in the almanac. One reply level,
// like the board.
//
// Who hears about a new comment depends on the subject, and only on that — the
// push kind stays the room's own, so nobody's opt-out changes meaning.

var commentSubjects = []string{"event", "wish", "away", "contact", "almanac"}

// subjectSwitch names the switch a subject lives behind (features.go). While
// it is off the thread is absent like the rows it hangs on: read, written
// and deleted nowhere, and its comments wait for the switch to come back.
var subjectSwitch = map[string]string{"almanac": "celestial"}

func (s *Server) subjectOn(ctx context.Context, subject string) bool {
	key, ok := subjectSwitch[subject]
	return !ok || s.feature(ctx, key)
}

func subjectOf(r *http.Request) (string, int64, bool) {
	s := r.PathValue("subject")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || !contains(commentSubjects, s) {
		return "", 0, false
	}
	return s, id, true
}

func (s *Server) listThread(w http.ResponseWriter, r *http.Request) {
	subject, id, ok := subjectOf(r)
	if !ok {
		writeErr(w, 400, "bad subject")
		return
	}
	if !s.subjectOn(r.Context(), subject) {
		writeErr(w, 404, "not in this village")
		return
	}
	rows, err := s.st.Rows(r.Context(), `SELECT c.*,`+houseJoin+` FROM comments c JOIN houses h ON h.id=c.house_id
		WHERE c.subject=? AND c.subject_id=? ORDER BY c.created_at LIMIT 500`, subject, id)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, rows)
}

func (s *Server) createThreadComment(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	subject, id, ok := subjectOf(r)
	if !ok {
		writeErr(w, 400, "bad subject")
		return
	}
	if !s.subjectOn(r.Context(), subject) {
		writeErr(w, 404, "not in this village")
		return
	}
	m, err := readJSON(r)
	if err != nil || str(m, "body") == "" {
		writeErr(w, 400, "body required")
		return
	}
	table := map[string]string{"event": "events", "wish": "wishes", "away": "away", "contact": "contacts", "almanac": "almanac"}[subject]
	// An away notice has no title, and must not grow one: what it says is
	// burglary information, so nothing about it reaches a push. An almanac
	// row has no house and is worded per language, so its kind goes along.
	label := map[string]string{"event": "title", "wish": "text", "away": "''", "contact": "name", "almanac": "kind"}[subject]
	ownerCol := "house_id"
	if subject == "almanac" {
		ownerCol = "NULL"
	}
	owner, err := s.st.One(r.Context(), `SELECT `+ownerCol+` AS house_id, `+label+` AS label FROM `+table+` WHERE id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if owner == nil {
		writeErr(w, 404, "no such "+subject)
		return
	}
	// A reply belongs to the same thread, and one level deep: a reply to a
	// reply hangs off the root.
	var parent any
	if v, ok := m["parent_id"].(float64); ok && v > 0 {
		p, _ := s.st.One(r.Context(), `SELECT subject, subject_id, parent_id FROM comments WHERE id=?`, int64(v))
		if p == nil || p["subject"] != subject || p["subject_id"].(int64) != id {
			writeErr(w, 400, "no such comment here")
			return
		}
		if p["parent_id"] != nil {
			parent = p["parent_id"]
		} else {
			parent = int64(v)
		}
	}
	cid, err := s.st.Exec(r.Context(), `INSERT INTO comments(subject, subject_id, house_id, parent_id, author, body) VALUES (?,?,?,?,?,?)`,
		subject, id, h.ID, parent, str(m, "author"), str(m, "body"))
	if err != nil {
		fail(w, err)
		return
	}
	// A contact outlives the house that added it, so this one column may be
	// NULL where every other subject's is not.
	ownerHouse, _ := owner["house_id"].(int64)
	s.tellThread(r, subject, id, h, ownerHouse, owner["label"].(string), snippet(str(m, "body"), 100))
	writeJSON(w, 201, map[string]any{"id": cid})
}

// tellThread pushes to the people the comment concerns: for an event, the house
// that called it plus everyone who answered yes or maybe; for a wish, the
// wisher plus every house that wants one; for an away notice, the house that is
// away plus its watcher.
func (s *Server) tellThread(r *http.Request, subject string, id int64, from *House, ownerHouse int64, label, body string) {
	// The phone book rings nobody: it has no push kind of its own, and
	// borrowing another room's would make somebody's opt-out mean something
	// they did not choose. A note on a number is read when the number is.
	if subject == "contact" {
		return
	}
	// The watchtower is the exception, and stays the one it already was: the
	// thread is read by the house that is away and by whoever watches for it,
	// and the push says only that something was said. A lock screen anyone can
	// read must not name an empty house, its dates or its notes.
	if subject == "away" {
		told := map[int64]bool{from.ID: true}
		row, _ := s.st.One(r.Context(), `SELECT watcher FROM away WHERE id=?`, id)
		for _, to := range []any{ownerHouse, row["watcher"]} {
			w, ok := to.(int64)
			if !ok || told[w] {
				continue
			}
			told[w] = true
			s.notifyHouse("away", w, func(lang string) Payload {
				return Payload{
					Title: "🕯️ " + tr(lang, "Watchtower"),
					Body:  tr(lang, "someone wrote on an absence — open the portal"),
					URL:   "#/watch",
				}
			})
		}
		return
	}
	kind, icon, room, table := "events", "💬", "#/tavern", "events"
	q := `SELECT DISTINCT house_id FROM event_signups WHERE event_id=? AND state IN ('yes','maybe')`
	if subject == "wish" {
		kind, room, table = "tools", "#/shed", "wishes"
		q = `SELECT DISTINCT house_id FROM wish_wants WHERE wish_id=?`
	}
	if subject == "almanac" {
		table = "almanac"
	}
	// The push opens the entry, not the room: the away notice above is the
	// one thread whose push stays at the room.
	url := s.entryLink(r.Context(), subject, table, id, room)
	// The sky has no caller and no answers: whoever already talks about this
	// full moon hears the reply, and nobody else is woken for it.
	if subject == "almanac" {
		key := label
		rows, _ := s.st.Rows(r.Context(), `SELECT DISTINCT house_id FROM comments WHERE subject='almanac' AND subject_id=? AND house_id != ?`, id, from.ID)
		for _, row := range rows {
			s.notifyHouse(kind, row["house_id"].(int64), func(lang string) Payload {
				return Payload{Title: icon + " " + from.Name + tr(lang, " on: ") + tr(lang, almanacTitles[key]), Body: body, URL: url}
			})
		}
		return
	}
	rows, _ := s.st.Rows(r.Context(), q, id)
	told := map[int64]bool{from.ID: true}
	for _, row := range append(rows, map[string]any{"house_id": ownerHouse}) {
		to := row["house_id"].(int64)
		if told[to] {
			continue
		}
		told[to] = true
		s.notifyHouse(kind, to, func(lang string) Payload {
			return Payload{Title: icon + " " + from.Name + tr(lang, " on: ") + label, Body: body, URL: url}
		})
	}
}

// entryLink is the address of one entry, the resolver route the page's Share
// button writes too (frontend/src/links.tsx): #/go/<kind>/<id>/<day>. The day
// is the UTC day the row was written, its created_at as stored, so the page
// can tell a reused id — no table uses AUTOINCREMENT — from the entry the link
// meant. The sky has no created_at, and its own day is what names a row. A
// row that cannot be read keeps the room's address.
func (s *Server) entryLink(ctx context.Context, kind, table string, id int64, room string) string {
	col := "created_at"
	if kind == "almanac" {
		col = "starts_at"
	}
	row, _ := s.st.One(ctx, `SELECT substr(`+col+`, 1, 10) AS day FROM `+table+` WHERE id=?`, id)
	day, _ := row["day"].(string)
	if day == "" {
		return room
	}
	return "#/go/" + kind + "/" + strconv.FormatInt(id, 10) + "/" + day
}

func (s *Server) deleteComment(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	id, _ := pathID(r)
	row, err := s.st.One(r.Context(), `SELECT house_id, subject FROM comments WHERE id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if row == nil || !s.subjectOn(r.Context(), row["subject"].(string)) {
		writeErr(w, 404, "not found")
		return
	}
	if row["house_id"].(int64) != h.ID && !h.IsSteward {
		writeErr(w, 403, "not yours")
		return
	}
	s.st.Exec(r.Context(), `DELETE FROM comments WHERE id=?`, id)
	w.WriteHeader(204)
}

// ---- options on a wish ----------------------------------------------------

// Anyone may add an option they found — a model, a price, a link. It is a
// finding, not a vote: options are never counted or ranked.
func (s *Server) createWishOption(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	m, err := readJSON(r)
	if err != nil || str(m, "text") == "" {
		writeErr(w, 400, "text required")
		return
	}
	wish, err := s.st.One(r.Context(), `SELECT house_id, text FROM wishes WHERE id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if wish == nil {
		writeErr(w, 404, "no such wish")
		return
	}
	url := strings.TrimSpace(str(m, "url"))
	if url != "" && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		url = "https://" + url
	}
	oid, err := s.st.Exec(r.Context(), `INSERT INTO wish_options(wish_id, house_id, text, url) VALUES (?,?,?,?)`, id, h.ID, str(m, "text"), url)
	if err != nil {
		fail(w, err)
		return
	}
	s.tellThread(r, "wish", id, h, wish["house_id"].(int64), wish["text"].(string), snippet(str(m, "text"), 100))
	writeJSON(w, 201, map[string]any{"id": oid})
}

func (s *Server) deleteWishOption(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	id, _ := pathID(r)
	row, err := s.st.One(r.Context(), `SELECT house_id FROM wish_options WHERE id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if row == nil {
		writeErr(w, 404, "not found")
		return
	}
	if row["house_id"].(int64) != h.ID && !h.IsSteward {
		writeErr(w, 403, "not yours")
		return
	}
	s.st.Exec(r.Context(), `DELETE FROM wish_options WHERE id=?`, id)
	w.WriteHeader(204)
}
