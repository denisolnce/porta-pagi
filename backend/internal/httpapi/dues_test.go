package httpapi

import (
	"fmt"
	"strings"
	"testing"
)

// dueByTitle: one due from the list, by its title.
func dueByTitle(t *testing.T, c *client, title string) map[string]any {
	t.Helper()
	code, _, list := c.do("GET", "/api/dues", nil)
	c.must(200, code, "list dues")
	for _, d := range list {
		if d["title"] == title {
			return d
		}
	}
	t.Fatalf("no due %q in %v", title, list)
	return nil
}

// owedRow: one house's row on a due, by house id; nil when the due does not
// ask that house.
func owedRow(d map[string]any, house float64) map[string]any {
	for _, x := range d["houses"].([]any) {
		if row := x.(map[string]any); row["house_id"] == house {
			return row
		}
	}
	return nil
}

// TestDuesOff: a village that never switched dues on has no ledger — every
// route is a 404 and the agent is not offered the tools.
func TestDuesOff(t *testing.T) {
	srv, _, steward, volk := newVillage(t)
	for _, c := range []struct{ method, path string }{
		{"GET", "/api/dues"}, {"POST", "/api/dues"}, {"PUT", "/api/dues/1"}, {"DELETE", "/api/dues/1"}, {"PUT", "/api/dues/1/houses/1"},
	} {
		if code, _, _ := steward.do(c.method, c.path, map[string]any{"title": "Voda"}); code != 404 {
			t.Fatalf("%s %s with dues off: %d", c.method, c.path, code)
		}
	}
	key, _ := mintKey(t, volk, "agent")
	_, out := rpc(t, srv, key, "tools/list", nil)
	if strings.Contains(fmt.Sprint(out), "list_dues") {
		t.Fatal("list_dues offered with dues off")
	}
	if _, out := rpc(t, srv, key, "tools/call", map[string]any{"name": "list_dues", "arguments": map[string]any{}}); out["error"] == nil {
		t.Fatalf("list_dues answered with dues off: %v", out)
	}
}

// TestDuesLedger walks a quarter: a steward asks every house, the collector
// marks who paid, the totals follow, and the ledger keeps the money a house
// paid after that house is gone.
func TestDuesLedger(t *testing.T) {
	srv, _, steward, volk := newVillage(t)
	steward.must(200, first(steward.do("PUT", "/api/features/dues", map[string]any{"on": true})), "dues on")
	_, me, _ := steward.do("GET", "/api/me", nil)
	_, vme, _ := volk.do("GET", "/api/me", nil)
	sID, vID := me["id"].(float64), vme["id"].(float64)
	// A common place is land and owes nothing.
	_, park, _ := steward.do("POST", "/api/houses", map[string]any{"name": "Parkirišče", "kind": "common"})
	parkID := park["id"].(float64)

	// Only a steward writes a due down.
	volk.must(403, first(volk.do("POST", "/api/dues", map[string]any{"title": "Voda"})), "a house writes a due")
	for _, bad := range []map[string]any{
		{"amount": "40"},
		{"title": "Voda", "amount": "40.555"},
		{"title": "Voda", "amount": "štirideset"},
		{"title": "Voda", "amount": "-5"},
		{"title": "Voda", "due_on": "30. 9."},
		{"title": "Voda", "houses": []any{}},
		{"title": "Voda", "houses": []any{parkID}},
		{"title": "Voda", "collector_id": parkID},
	} {
		if code := first(steward.do("POST", "/api/dues", bad)); code != 400 {
			t.Fatalf("%v: want 400 got %d", bad, code)
		}
	}
	steward.must(201, first(steward.do("POST", "/api/dues", map[string]any{"title": "Voda, Q3", "amount": "40,50", "currency": "€", "due_on": "2026-09-30"})), "water")
	d := dueByTitle(t, volk, "Voda, Q3")
	if d["owed"] != 2.0 || d["paid"] != 0.0 || d["amount_cents"] != 4050.0 || d["expected_cents"] != 8100.0 || d["collected_cents"] != 0.0 {
		t.Fatalf("a new due: %v", d)
	}
	if d["collector_id"] != sID || owedRow(d, parkID) != nil || owedRow(d, vID) == nil {
		t.Fatalf("the steward collects, every house and no common place owes: %v", d)
	}
	id := fmt.Sprint(d["id"])

	// A house does not mark itself; the collector does.
	volk.must(403, first(volk.do("PUT", "/api/dues/"+id+"/houses/"+fmt.Sprint(vID), map[string]any{"paid": true})), "a house marks itself")
	steward.must(400, first(steward.do("PUT", "/api/dues/"+id+"/houses/"+fmt.Sprint(vID), map[string]any{})), "no paid")
	steward.must(404, first(steward.do("PUT", "/api/dues/"+id+"/houses/"+fmt.Sprint(parkID), map[string]any{"paid": true})), "a house the due does not ask")
	steward.must(204, first(steward.do("PUT", "/api/dues/"+id+"/houses/"+fmt.Sprint(vID), map[string]any{"paid": true})), "mark paid")
	d = dueByTitle(t, volk, "Voda, Q3")
	if row := owedRow(d, vID); d["paid"] != 1.0 || d["collected_cents"] != 4050.0 || row["paid_at"] == nil || row["paid_by_name"] != "S" {
		t.Fatalf("after one payment: %v", d)
	}

	// A paid house stays on the due until it is marked unpaid.
	steward.must(409, first(steward.do("PUT", "/api/dues/"+id, map[string]any{"houses": []any{sID}})), "drop a paid house")
	steward.must(204, first(steward.do("PUT", "/api/dues/"+id+"/houses/"+fmt.Sprint(vID), map[string]any{"paid": false})), "unmark")
	steward.must(204, first(steward.do("PUT", "/api/dues/"+id, map[string]any{"houses": []any{sID}, "amount": "", "title": "Voda, Q3 popravek"})), "narrow the due")
	d = dueByTitle(t, volk, "Voda, Q3 popravek")
	if d["owed"] != 1.0 || d["amount_cents"] != nil || d["expected_cents"] != nil || owedRow(d, vID) != nil {
		t.Fatalf("after the correction: %v", d)
	}
	volk.must(403, first(volk.do("PUT", "/api/dues/"+id, map[string]any{"title": "x"})), "a house edits a due")

	// A due collected by an ordinary house: that house marks, the steward may too.
	steward.must(201, first(steward.do("POST", "/api/dues", map[string]any{"title": "Cesta", "amount": 25, "currency": "EUR", "collector_id": vID})), "road")
	road := dueByTitle(t, volk, "Cesta")
	rid := fmt.Sprint(road["id"])
	volk.must(204, first(volk.do("PUT", "/api/dues/"+rid+"/houses/"+fmt.Sprint(sID), map[string]any{"paid": true})), "the collector marks")
	volk.must(204, first(volk.do("PUT", "/api/dues/"+rid+"/houses/"+fmt.Sprint(vID), map[string]any{"paid": true})), "the collector marks itself")
	steward.must(204, first(steward.do("PUT", "/api/dues/"+rid+"/houses/"+fmt.Sprint(vID), map[string]any{"paid": false})), "a steward marks on a due another house collects")
	volk.must(204, first(volk.do("PUT", "/api/dues/"+rid+"/houses/"+fmt.Sprint(vID), map[string]any{"paid": true})), "the collector marks again")
	road = dueByTitle(t, volk, "Cesta")
	if road["paid"] != 2.0 || road["collected_cents"] != 5000.0 {
		t.Fatalf("the road, settled: %v", road)
	}

	// The agent of the collecting house reads the ledger and marks.
	key, _ := mintKey(t, volk, "agent")
	if isErr, text := call(t, srv, key, "list_dues", nil); isErr || !strings.Contains(text, "Cesta") {
		t.Fatalf("list_dues: %v %s", isErr, text)
	}
	if isErr, text := call(t, srv, key, "mark_due", map[string]any{"id": road["id"], "house_id": sID, "paid": false}); isErr {
		t.Fatalf("mark_due as the collector: %s", text)
	}
	if isErr, _ := call(t, srv, key, "mark_due", map[string]any{"id": d["id"], "house_id": sID, "paid": true}); !isErr {
		t.Fatal("mark_due on a due another house collects")
	}

	// A house that leaves: both its rows stay without a name, paid and unpaid.
	_, sova, _ := steward.do("POST", "/api/houses", map[string]any{"name": "Sova"})
	sovaID := sova["id"].(float64)
	steward.must(201, first(steward.do("POST", "/api/dues", map[string]any{"title": "Zavarovanje", "amount": "10"})), "insurance")
	steward.must(201, first(steward.do("POST", "/api/dues", map[string]any{"title": "Smeti", "amount": "5"})), "rubbish")
	ins := dueByTitle(t, steward, "Zavarovanje")
	steward.must(204, first(steward.do("PUT", "/api/dues/"+fmt.Sprint(ins["id"])+"/houses/"+fmt.Sprint(sovaID), map[string]any{"paid": true})), "sova pays")
	steward.must(204, first(steward.do("DELETE", "/api/houses/"+fmt.Sprint(sovaID), nil)), "sova leaves")
	ins = dueByTitle(t, steward, "Zavarovanje")
	if ins["owed"] != 3.0 || ins["paid"] != 1.0 || ins["collected_cents"] != 1000.0 {
		t.Fatalf("the money Sova paid left with it: %v", ins)
	}
	if last := ins["houses"].([]any)[2].(map[string]any); last["house_id"] != nil || last["paid_at"] == nil {
		t.Fatalf("the departed house's row: %v", last)
	}
	rub := dueByTitle(t, steward, "Smeti")
	if rub["owed"] != 3.0 || rub["paid"] != 0.0 {
		t.Fatalf("deleting Sova forgave its unpaid row: %v", rub)
	}
	// Editing the due's houses does not forgive it either.
	steward.must(204, first(steward.do("PUT", "/api/dues/"+fmt.Sprint(rub["id"]), map[string]any{"houses": []any{sID}})), "smeti asks only the steward now")
	if rub = dueByTitle(t, steward, "Smeti"); rub["owed"] != 2.0 {
		t.Fatalf("an edit dropped the departed house's debt: %v", rub)
	}

	// The export carries the ledger.
	_, exp, _ := steward.do("GET", "/api/export", nil)
	if len(exp["dues"].([]any)) != 4 || exp["dues_owed"] == nil {
		t.Fatalf("export: %v", exp["dues"])
	}

	// A steward takes a due away, with its rows.
	volk.must(403, first(volk.do("DELETE", "/api/dues/"+rid, nil)), "a house deletes a due")
	steward.must(204, first(steward.do("DELETE", "/api/dues/"+rid, nil)), "delete")
	if row, _ := srv.st.One(t.Context(), `SELECT count(*) AS n FROM dues_owed WHERE due_id=?`, road["id"]); row["n"] != int64(0) {
		t.Fatalf("rows outlived their due: %v", row)
	}
}

// first is in features_test.go: the status of a client call, nothing else.
