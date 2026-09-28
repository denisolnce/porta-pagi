package httpapi

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// rosterOf reads the room as a house sees it.
func rosterOf(t *testing.T, c *client) map[string]any {
	t.Helper()
	code, obj, _ := c.do("GET", "/api/roster", nil)
	c.must(200, code, "roster")
	return obj
}

func dutyIn(t *testing.T, roster map[string]any, title string) map[string]any {
	t.Helper()
	for _, d := range roster["duties"].([]any) {
		if d := d.(map[string]any); d["title"] == title {
			return d
		}
	}
	t.Fatalf("no duty %q in %v", title, roster["duties"])
	return nil
}

// countOf: the one cell of the table for a house and a duty, 0 when absent.
func countOf(roster map[string]any, house, duty float64) float64 {
	for _, c := range roster["counts"].([]any) {
		c := c.(map[string]any)
		if c["house_id"] == house && c["duty_id"] == duty {
			return c["n"].(float64)
		}
	}
	return 0
}

// TestRosterOff: a village that never switched the roster on has no roster —
// not its routes, not its tools.
func TestRosterOff(t *testing.T) {
	srv, _, steward, volk := newVillage(t)
	for _, c := range []struct {
		who          *client
		method, path string
	}{
		{volk, "GET", "/api/roster"},
		{steward, "POST", "/api/roster/duties"},
		{steward, "PUT", "/api/roster/duties/1"},
		{steward, "DELETE", "/api/roster/duties/1"},
		{volk, "POST", "/api/roster/duties/1/turns"},
		{volk, "PUT", "/api/roster/turns/1"},
		{volk, "DELETE", "/api/roster/turns/1"},
	} {
		if code, _, _ := c.who.do(c.method, c.path, map[string]any{"title": "x"}); code != 404 {
			t.Fatalf("%s %s with the roster off: %d", c.method, c.path, code)
		}
	}
	key, _ := mintKey(t, volk, "agent")
	_, out := rpc(t, srv, key, "tools/list", nil)
	for _, tl := range out["result"].(map[string]any)["tools"].([]any) {
		if strings.Contains(tl.(map[string]any)["name"].(string), "roster") {
			t.Fatalf("a roster tool is listed with the roster off: %v", tl)
		}
	}
	if _, out = rpc(t, srv, key, "tools/call", map[string]any{"name": "add_turn", "arguments": map[string]any{"id": 1}}); out["error"] == nil {
		t.Fatalf("add_turn answered with the roster off: %v", out)
	}
}

// TestRoster walks the room: a steward writes the duties, a house takes a
// turn ahead and marks it done, records one it did, and the count is done
// turns per house per duty over the last twelve months.
func TestRoster(t *testing.T) {
	srv, _, steward, volk := newVillage(t)
	steward.must(200, func() int { c, _, _ := steward.do("PUT", "/api/features/roster", map[string]any{"on": true}); return c }(), "switch on")
	_, me, _ := volk.do("GET", "/api/me", nil)
	volkID := me["id"].(float64)

	// Duties are the steward's pen.
	if code, _, _ := volk.do("POST", "/api/roster/duties", map[string]any{"title": "Mow"}); code != 403 {
		t.Fatalf("a house wrote a duty: %d", code)
	}
	for _, bad := range []any{0.5, -3, 400, "14"} {
		if code, _, _ := steward.do("POST", "/api/roster/duties", map[string]any{"title": "Mow", "every_days": bad}); code != 400 {
			t.Fatalf("every_days %v: %d", bad, code)
		}
	}
	code, o, _ := steward.do("POST", "/api/roster/duties", map[string]any{"title": "Mow the common ground", "every_days": 14})
	steward.must(201, code, "duty")
	mow := o["id"].(float64)
	code, o, _ = steward.do("POST", "/api/roster/duties", map[string]any{"title": "Clear the lane"})
	steward.must(201, code, "duty without a cadence")
	lane := o["id"].(float64)
	turns := func(id float64) string { return "/api/roster/duties/" + itoa(id) + "/turns" }

	// "Now" is Friday 2026-09-04. A plan is today or later, a done turn today
	// or earlier.
	code, o, _ = volk.do("POST", turns(mow), map[string]any{"on_date": "2026-09-06"})
	volk.must(201, code, "take a turn ahead")
	sunday := o["id"].(float64)
	if code, _, _ := volk.do("POST", turns(mow), map[string]any{"on_date": "2026-09-06"}); code != 409 {
		t.Fatalf("the same house, duty and day twice: %d", code)
	}
	if code, _, _ := volk.do("POST", turns(mow), map[string]any{"on_date": "2026-09-05", "done": true}); code != 400 {
		t.Fatalf("done tomorrow: %d", code)
	}
	if code, _, _ := volk.do("POST", turns(mow), map[string]any{"on_date": "2026-09-01"}); code != 400 {
		t.Fatalf("a plan for a day gone: %d", code)
	}
	if code, _, _ := volk.do("POST", turns(mow), map[string]any{"on_date": "4.9.2026", "done": true}); code != 400 {
		t.Fatalf("a date that is not YYYY-MM-DD: %d", code)
	}
	code, _, _ = volk.do("POST", turns(mow), map[string]any{"on_date": "2026-08-30", "done": true, "note": "upper half only"})
	volk.must(201, code, "record a done turn")
	// A turn thirteen months back is history, not count; one on the window's
	// first day counts, as "since" says.
	code, _, _ = volk.do("POST", turns(mow), map[string]any{"on_date": "2025-08-01", "done": true})
	volk.must(201, code, "an old turn")
	code, _, _ = volk.do("POST", turns(lane), map[string]any{"on_date": "2025-09-04", "done": true})
	volk.must(201, code, "a turn on the window's first day")

	r := rosterOf(t, volk)
	d := dutyIn(t, r, "Mow the common ground")
	if d["last_done"] != "2026-08-30" || d["next_due"] != "2026-09-13" {
		t.Fatalf("last done / next due: %v / %v", d["last_done"], d["next_due"])
	}
	if dutyIn(t, r, "Clear the lane")["next_due"] != nil {
		t.Fatal("a duty with no cadence has no next due")
	}
	if n := countOf(r, volkID, lane); n != 1 {
		t.Fatalf("the window's first day: %v", n)
	}
	if n := countOf(r, volkID, mow); n != 1 {
		t.Fatalf("count: %v, want 1 (the planned turn and last year's do not count)", n)
	}
	if r["since"] != "2025-09-04" {
		t.Fatalf("window: %v", r["since"])
	}

	// Nobody else writes a house's turns, a steward included.
	path := "/api/roster/turns/" + itoa(sunday)
	if code, _, _ := steward.do("PUT", path, map[string]any{"done": true}); code != 403 {
		t.Fatalf("a steward marked another house's turn: %d", code)
	}
	if code, _, _ := steward.do("DELETE", path, nil); code != 403 {
		t.Fatalf("a steward removed another house's turn: %d", code)
	}
	if code, _, _ := volk.do("PUT", path, map[string]any{"done": true}); code != 400 {
		t.Fatalf("marked done before its day: %d", code)
	}

	// Monday: Sunday's plan has passed. Its house still sees it, to say it
	// happened; the others do not see a missed turn at all.
	srv.now = func() time.Time { return time.Date(2026, 9, 7, 10, 0, 0, 0, time.Local) }
	if n := len(dutyIn(t, rosterOf(t, steward), "Mow the common ground")["turns"].([]any)); n != 1 {
		t.Fatalf("the steward sees %d mowing turns, want only the done one", n)
	}
	if n := len(dutyIn(t, rosterOf(t, volk), "Mow the common ground")["turns"].([]any)); n != 2 {
		t.Fatalf("the house sees %d of its turns, want the passed plan too", n)
	}
	if code, _, _ := volk.do("PUT", path, map[string]any{"note": "rain"}); code != 204 {
		t.Fatalf("a note on a passed plan: %d", code)
	}
	if code, _, _ := volk.do("PUT", path, map[string]any{"on_date": "2026-09-05"}); code != 400 {
		t.Fatalf("moved a plan into the past: %d", code)
	}
	volk.must(204, func() int { c, _, _ := volk.do("PUT", path, map[string]any{"done": true}); return c }(), "mark done")
	r = rosterOf(t, volk)
	if n := countOf(r, volkID, mow); n != 2 {
		t.Fatalf("count after marking done: %v", n)
	}
	if d := dutyIn(t, r, "Mow the common ground"); d["next_due"] != "2026-09-20" {
		t.Fatalf("next due: %v", d["next_due"])
	}

	// A duty somebody did is retired, not deleted; a retired one takes no
	// turns, and a plan made before it was retired does not become one.
	code, o, _ = volk.do("POST", turns(mow), map[string]any{"on_date": "2026-09-07"})
	volk.must(201, code, "a plan for today")
	later := o["id"].(float64)
	if code, _, _ := steward.do("DELETE", "/api/roster/duties/"+itoa(mow), nil); code != 409 {
		t.Fatalf("deleted a duty with history: %d", code)
	}
	steward.must(204, func() int {
		c, _, _ := steward.do("PUT", "/api/roster/duties/"+itoa(mow), map[string]any{"retired": true})
		return c
	}(), "retire")
	if code, _, _ := volk.do("POST", turns(mow), map[string]any{"done": true}); code != 409 {
		t.Fatalf("a turn on a retired duty: %d", code)
	}
	if code, _, _ := volk.do("PUT", "/api/roster/turns/"+itoa(later), map[string]any{"done": true}); code != 409 {
		t.Fatalf("a plan on a retired duty marked done: %d", code)
	}
	steward.do("PUT", "/api/roster/duties/"+itoa(mow), map[string]any{"retired": false})
	volk.do("DELETE", "/api/roster/turns/"+itoa(later), nil)
	// A duty nobody did yet goes; its plans with it. And until its first turn
	// it counts its cadence from the day it was written down.
	code, o, _ = steward.do("POST", "/api/roster/duties", map[string]any{"title": "Check the pump", "every_days": 7})
	steward.must(201, code, "a new duty")
	pump := o["id"].(float64)
	if d := dutyIn(t, rosterOf(t, volk), "Check the pump"); d["next_due"] == nil {
		t.Fatal("a duty with a cadence and no turn yet is never due")
	}
	volk.do("POST", turns(pump), map[string]any{"on_date": "2026-12-01"})
	steward.must(204, func() int { c, _, _ := steward.do("DELETE", "/api/roster/duties/"+itoa(pump), nil); return c }(), "delete an unused duty")

	// A house takes back its own turn.
	code, o, _ = volk.do("POST", turns(mow), map[string]any{"done": true})
	volk.must(201, code, "today, by default")
	volk.must(204, func() int { c, _, _ := volk.do("DELETE", "/api/roster/turns/"+itoa(o["id"].(float64)), nil); return c }(), "take it back")

	// The export carries both tables.
	_, exp, _ := steward.do("GET", "/api/export", nil)
	if len(exp["roster_duties"].([]any)) != 2 || len(exp["roster_turns"].([]any)) != 4 {
		t.Fatalf("export: %v %v", exp["roster_duties"], exp["roster_turns"])
	}

	// A house leaves: its row goes from the table, but the lane was still
	// mown that day — next due does not jump back.
	steward.must(204, func() int { c, _, _ := steward.do("DELETE", "/api/houses/"+itoa(volkID), nil); return c }(), "delete house")
	r = rosterOf(t, steward)
	if len(r["counts"].([]any)) != 0 {
		t.Fatalf("a house that left is still counted: %v", r["counts"])
	}
	if d := dutyIn(t, r, "Mow the common ground"); d["last_done"] != "2026-09-06" || d["next_due"] != "2026-09-20" {
		t.Fatalf("a house leaving moved the duty: %v / %v", d["last_done"], d["next_due"])
	}
}

// TestRosterMCP: with the switch on, an agent records its own house's turn.
func TestRosterMCP(t *testing.T) {
	srv, _, steward, volk := newVillage(t)
	steward.do("PUT", "/api/features/roster", map[string]any{"on": true})
	_, o, _ := steward.do("POST", "/api/roster/duties", map[string]any{"title": "Check the pump", "every_days": 7})
	key, _ := mintKey(t, volk, "agent")
	isErr, text := call(t, srv, key, "add_turn", map[string]any{"id": o["id"], "done": true, "note": "pressure fine"})
	if isErr {
		t.Fatalf("add_turn: %s", text)
	}
	isErr, text = call(t, srv, key, "list_roster", nil)
	if isErr || !strings.Contains(text, `"next_due":"2026-09-11"`) || !strings.Contains(text, "pressure fine") {
		t.Fatalf("list_roster: %s", text)
	}
}

// TestRosterCommonPlace: land takes no turns. A common place never holds a
// token in the app; one is planted to prove the route refuses it anyway.
func TestRosterCommonPlace(t *testing.T) {
	srv, _, steward, _ := newVillage(t)
	steward.must(200, func() int { c, _, _ := steward.do("PUT", "/api/features/roster", map[string]any{"on": true}); return c }(), "switch on")
	_, duty, _ := steward.do("POST", "/api/roster/duties", map[string]any{"title": "Mow"})
	_, obj, _ := steward.do("POST", "/api/houses", map[string]any{"name": "Largo", "kind": "common"})
	tok := randomToken(32)
	srv.st.Exec(nil2(), `INSERT INTO devices(house_id, token_hash, label) VALUES (?,?,?)`, int64(obj["id"].(float64)), hashToken(tok), "planted")
	common := &client{t: t, h: srv.Handler(), token: tok}
	if code, _, _ := common.do("POST", "/api/roster/duties/"+fmt.Sprint(duty["id"])+"/turns", map[string]any{"done": true, "on_date": srv.now().Format("2006-01-02")}); code != 403 {
		t.Fatalf("a common place took a turn: %d", code)
	}
}
