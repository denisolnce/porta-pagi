package httpapi

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dennislapchenko/porta-pagi/backend/internal/store"
)

// Dues: who paid the water this quarter. Behind the `dues` switch — one of
// the two places the portal counts, and only because a village switched it
// on (docs/design-more-rooms.md § Dues ledger).
//
// A steward writes a due down, because asking every house for money is the
// village's decision, taken at its meeting, and not one house's. The steward
// names the house that collects it. That house — or a steward — marks a house
// paid, because the one holding the money is the one who knows it arrived.
// A house does not mark itself: a ledger of claims answers nobody.
//
// The portal records; it never moves money. No payment link, no account
// number field, no card anything. Nothing here pushes: money should not
// buzz phones (the Campground's first rule), and the collector says it in
// the Tavern in its own words.
//
// Totals are per due — collected against expected — and never per house
// across dues: the ledger counts a due, not a house.

func (s *Server) listDues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Open dues first, soonest date first and the undated after them; a due
	// every house has paid sinks below, newest first.
	dues, err := s.st.Rows(ctx, `SELECT d.*, c.name AS collector_name, c.crest AS collector_crest, c.color AS collector_color,
		(SELECT count(*) FROM dues_owed o WHERE o.due_id=d.id) AS owed,
		(SELECT count(*) FROM dues_owed o WHERE o.due_id=d.id AND o.paid_at IS NOT NULL) AS paid
		FROM dues d LEFT JOIN houses c ON c.id=d.collector_id
		ORDER BY paid = owed, CASE WHEN paid = owed THEN 0 ELSE d.due_on IS NULL END, CASE WHEN paid = owed THEN NULL ELSE d.due_on END,
		d.created_at DESC, d.id DESC LIMIT 200`)
	if err != nil {
		fail(w, err)
		return
	}
	// A departed house's paid row has no name and sorts last.
	owed, err := s.st.Rows(ctx, `SELECT o.id AS row_id, o.due_id, o.house_id, o.paid_at, o.paid_by, h.name, h.crest, h.color, b.name AS paid_by_name
		FROM dues_owed o LEFT JOIN houses h ON h.id=o.house_id LEFT JOIN houses b ON b.id=o.paid_by
		ORDER BY o.house_id IS NULL, h.name COLLATE NOCASE`)
	if err != nil {
		fail(w, err)
		return
	}
	byDue := map[int64][]store.Row{}
	for _, o := range owed {
		id := o["due_id"].(int64)
		delete(o, "due_id")
		byDue[id] = append(byDue[id], o)
	}
	for _, d := range dues {
		id := d["id"].(int64)
		d["houses"] = byDue[id]
		if d["houses"] == nil {
			d["houses"] = []store.Row{}
		}
		if a, ok := d["amount_cents"].(int64); ok {
			d["expected_cents"] = a * d["owed"].(int64)
			d["collected_cents"] = a * d["paid"].(int64)
		}
	}
	writeJSON(w, 200, dues)
}

// parseAmount reads what a villager typed — "40", "40.50", "40,50", "1 200" —
// into cents. Empty is no amount. More than two decimals is refused rather
// than rounded: an amount that changes under you is the silent clamp.
func parseAmount(v any) (*int64, string) {
	var x float64
	switch a := v.(type) {
	case nil:
		return nil, ""
	case float64:
		x = a
	case string:
		t := strings.NewReplacer(" ", "", " ", "", " ", "", ",", ".").Replace(a)
		if t == "" {
			return nil, ""
		}
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return nil, "the amount is not a number"
		}
		x = f
	default:
		return nil, "the amount is not a number"
	}
	c := math.Round(x * 100)
	if math.Abs(x*100-c) > 1e-6 {
		return nil, "an amount has at most two decimals"
	}
	if c <= 0 || c > 1e11 || math.IsNaN(c) {
		return nil, "the amount must be above zero"
	}
	n := int64(c)
	return &n, ""
}

// dueHouses reads the houses a due asks: ids of houses, never a common
// place, which is land and pays nothing. Absent means every house.
func (s *Server) dueHouses(r *http.Request, v any) ([]int64, string) {
	ctx := r.Context()
	if v == nil {
		rows, err := s.st.Rows(ctx, `SELECT id FROM houses WHERE kind='house'`)
		if err != nil {
			return nil, "server error"
		}
		out := []int64{}
		for _, row := range rows {
			out = append(out, row["id"].(int64))
		}
		return out, ""
	}
	list, ok := v.([]any)
	if !ok {
		return nil, "houses is a list of house ids"
	}
	seen := map[int64]bool{}
	out := []int64{}
	for _, x := range list {
		f, ok := x.(float64)
		if !ok {
			return nil, "houses is a list of house ids"
		}
		id := int64(f)
		if seen[id] {
			continue
		}
		row, _ := s.st.One(ctx, `SELECT kind FROM houses WHERE id=?`, id)
		if row == nil || row["kind"] != "house" {
			return nil, "a due asks houses, not common places"
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, "a due asks at least one house"
	}
	return out, ""
}

// dueFields reads the fields both create and update take, where present.
// It answers the columns to write and an error for the caller.
func (s *Server) dueFields(r *http.Request, m map[string]any) (map[string]any, string) {
	set := map[string]any{}
	if _, ok := m["title"]; ok {
		if str(m, "title") == "" {
			return nil, "a due needs a title"
		}
		set["title"] = str(m, "title")
	}
	if v, ok := m["amount"]; ok {
		n, msg := parseAmount(v)
		if msg != "" {
			return nil, msg
		}
		if n == nil {
			set["amount_cents"] = nil
		} else {
			set["amount_cents"] = *n
		}
	}
	if _, ok := m["currency"]; ok {
		c := str(m, "currency")
		if utf8.RuneCountInString(c) > 8 {
			return nil, "a currency is a symbol or a short word"
		}
		set["currency"] = c
	}
	if _, ok := m["due_on"]; ok {
		d := str(m, "due_on")
		if d == "" {
			set["due_on"] = nil
		} else if _, err := time.Parse("2006-01-02", d); err != nil {
			return nil, "due_on is YYYY-MM-DD"
		} else {
			set["due_on"] = d
		}
	}
	if _, ok := m["notes"]; ok {
		set["notes"] = str(m, "notes")
	}
	if v, ok := m["collector_id"]; ok {
		f, ok := v.(float64)
		if !ok {
			return nil, "collector_id is a house id"
		}
		row, _ := s.st.One(r.Context(), `SELECT kind FROM houses WHERE id=?`, int64(f))
		if row == nil || row["kind"] != "house" {
			return nil, "a due is collected by a house"
		}
		set["collector_id"] = int64(f)
	}
	return set, ""
}

func (s *Server) createDue(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	if !h.IsSteward {
		writeErr(w, 403, "stewards only")
		return
	}
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	if str(m, "title") == "" {
		writeErr(w, 400, "a due needs a title")
		return
	}
	set, msg := s.dueFields(r, m)
	if msg != "" {
		writeErr(w, 400, msg)
		return
	}
	if _, ok := set["collector_id"]; !ok {
		set["collector_id"] = h.ID
	}
	houses, msg := s.dueHouses(r, m["houses"])
	if msg != "" {
		writeErr(w, 400, msg)
		return
	}
	var id int64
	err = s.st.Tx(r.Context(), func(tx *store.Store) error {
		var err error
		id, err = tx.Exec(r.Context(), `INSERT INTO dues(title, amount_cents, currency, due_on, notes, collector_id, house_id) VALUES (?,?,?,?,?,?,?)`,
			set["title"], set["amount_cents"], orEmpty(set["currency"]), set["due_on"], orEmpty(set["notes"]), set["collector_id"], h.ID)
		if err != nil {
			return err
		}
		for _, hid := range houses {
			if _, err := tx.Exec(r.Context(), `INSERT INTO dues_owed(due_id, house_id) VALUES (?,?)`, id, hid); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"id": id})
}

func orEmpty(v any) any {
	if v == nil {
		return ""
	}
	return v
}

// updateDue: a steward corrects a due. `houses` replaces the set it asks; a
// house that has paid cannot be dropped from it — mark it unpaid first, so
// money that came in never leaves the ledger by a side door.
func (s *Server) updateDue(w http.ResponseWriter, r *http.Request) {
	if !houseFrom(r).IsSteward {
		writeErr(w, 403, "stewards only")
		return
	}
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
	if row, err := s.st.One(r.Context(), `SELECT id FROM dues WHERE id=?`, id); err != nil {
		fail(w, err)
		return
	} else if row == nil {
		writeErr(w, 404, "not found")
		return
	}
	set, msg := s.dueFields(r, m)
	if msg != "" {
		writeErr(w, 400, msg)
		return
	}
	var houses []int64
	if v, ok := m["houses"]; ok {
		if v == nil {
			writeErr(w, 400, "houses is a list of house ids")
			return
		}
		if houses, msg = s.dueHouses(r, v); msg != "" {
			writeErr(w, 400, msg)
			return
		}
		keep := map[int64]bool{}
		for _, hid := range houses {
			keep[hid] = true
		}
		paid, err := s.st.Rows(r.Context(), `SELECT house_id FROM dues_owed WHERE due_id=? AND paid_at IS NOT NULL AND house_id IS NOT NULL`, id)
		if err != nil {
			fail(w, err)
			return
		}
		for _, p := range paid {
			if !keep[p["house_id"].(int64)] {
				writeErr(w, 409, "a house that has paid stays on the due; mark it unpaid first")
				return
			}
		}
	}
	err = s.st.Tx(r.Context(), func(tx *store.Store) error {
		for k, v := range set { // k is one of dueFields' own column names
			if k == "currency" || k == "notes" {
				v = orEmpty(v)
			}
			if _, err := tx.Exec(r.Context(), `UPDATE dues SET `+k+`=? WHERE id=?`, v, id); err != nil {
				return err
			}
		}
		if houses == nil {
			return nil
		}
		keep := map[int64]bool{}
		for _, hid := range houses {
			keep[hid] = true
			if _, err := tx.Exec(r.Context(), `INSERT OR IGNORE INTO dues_owed(due_id, house_id) VALUES (?,?)`, id, hid); err != nil {
				return err
			}
		}
		rows, err := tx.Rows(r.Context(), `SELECT id, house_id FROM dues_owed WHERE due_id=? AND paid_at IS NULL`, id)
		if err != nil {
			return err
		}
		for _, o := range rows {
			// A row whose house has left is not the edit's to drop: its
			// debt stays until the due goes (026_dues.sql).
			hid, ok := o["house_id"].(int64)
			if !ok || keep[hid] {
				continue
			}
			if _, err := tx.Exec(r.Context(), `DELETE FROM dues_owed WHERE id=?`, o["id"]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}

// deleteDue: a steward takes a due away, with its ledger. The confirm dialog
// says so, and last night's backup is the undo.
func (s *Server) deleteDue(w http.ResponseWriter, r *http.Request) {
	if !houseFrom(r).IsSteward {
		writeErr(w, 403, "stewards only")
		return
	}
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	if _, err := s.st.Exec(r.Context(), `DELETE FROM dues WHERE id=?`, id); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}

// markDue: {"paid": true|false} for one house on one due. The collecting
// house or a steward; the row keeps who marked it and when.
func (s *Server) markDue(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	hid, err := strconv.ParseInt(r.PathValue("house_id"), 10, 64)
	if err != nil {
		writeErr(w, 400, "bad house id")
		return
	}
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	paid, ok := m["paid"].(bool)
	if !ok {
		writeErr(w, 400, "paid must be true or false")
		return
	}
	due, err := s.st.One(r.Context(), `SELECT collector_id FROM dues WHERE id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if due == nil {
		writeErr(w, 404, "not found")
		return
	}
	if c, ok := due["collector_id"].(int64); !h.IsSteward && (!ok || c != h.ID) {
		writeErr(w, 403, "the house that collects it, or a steward")
		return
	}
	if row, err := s.st.One(r.Context(), `SELECT id FROM dues_owed WHERE due_id=? AND house_id=?`, id, hid); err != nil {
		fail(w, err)
		return
	} else if row == nil {
		writeErr(w, 404, "this due does not ask that house")
		return
	}
	// Marking a paid house paid again keeps the first stamp: the day the
	// money came is the fact, not the day somebody tapped twice.
	q := `UPDATE dues_owed SET paid_at=datetime('now'), paid_by=? WHERE due_id=? AND house_id=? AND paid_at IS NULL`
	args := []any{h.ID, id, hid}
	if !paid {
		q, args = `UPDATE dues_owed SET paid_at=NULL, paid_by=NULL WHERE due_id=? AND house_id=?`, []any{id, hid}
	}
	if _, err := s.st.Exec(r.Context(), q, args...); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}

// settleLeft: a row whose house has left, by the row's own id — the house
// is gone, so the house route cannot name it. PUT {"paid": true|false}
// marks it as markDue marks a named row (the household paid at the door on
// its way out); DELETE forgives an unpaid one, the collector's or a
// steward's deliberate act, never a side effect of deleting the house.
func (s *Server) settleLeft(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	rowID, err := strconv.ParseInt(r.PathValue("row"), 10, 64)
	if err != nil {
		writeErr(w, 400, "bad row id")
		return
	}
	due, err := s.st.One(r.Context(), `SELECT collector_id FROM dues WHERE id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if due == nil {
		writeErr(w, 404, "not found")
		return
	}
	if c, ok := due["collector_id"].(int64); !h.IsSteward && (!ok || c != h.ID) {
		writeErr(w, 403, "the house that collects it, or a steward")
		return
	}
	row, err := s.st.One(r.Context(), `SELECT paid_at FROM dues_owed WHERE id=? AND due_id=? AND house_id IS NULL`, rowID, id)
	if err != nil {
		fail(w, err)
		return
	}
	if row == nil {
		writeErr(w, 404, "no row of a house that has left")
		return
	}
	if r.Method == http.MethodDelete {
		if row["paid_at"] != nil {
			writeErr(w, 409, "paid money stays in the ledger; mark it unpaid first")
			return
		}
		if _, err := s.st.Exec(r.Context(), `DELETE FROM dues_owed WHERE id=?`, rowID); err != nil {
			fail(w, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	paid, ok := m["paid"].(bool)
	if !ok {
		writeErr(w, 400, "paid must be true or false")
		return
	}
	q, args := `UPDATE dues_owed SET paid_at=datetime('now'), paid_by=? WHERE id=? AND paid_at IS NULL`, []any{h.ID, rowID}
	if !paid {
		q, args = `UPDATE dues_owed SET paid_at=NULL, paid_by=NULL WHERE id=?`, []any{rowID}
	}
	if _, err := s.st.Exec(r.Context(), q, args...); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}
