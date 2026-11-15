package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestBuys: any house but the asker marks an ask bought, the asker hears who,
// and whether it was paid back is the buyer's alone — sent to the buyer only,
// set by the buyer only, kept past the month's window while unpaid.
func TestBuys(t *testing.T) {
	srv, fake, steward, volk := newVillage(t)
	_, o, _ := steward.do("POST", "/api/houses", map[string]any{"name": "Breza"})
	breza := &client{t: t, h: volk.h}
	_, j, _ := breza.do("POST", "/api/join", map[string]any{"pin": "654321", "code": o["invite"].(map[string]any)["code"]})
	breza.token = j["token"].(string)

	for _, bad := range []map[string]any{{}, {"link": "javascript:alert(1)"}, {"link": "amzn.eu/d/x"}, {"text": strings.Repeat("ž", 2001)}} {
		code, _, _ := volk.do("POST", "/api/buys", bad)
		volk.must(400, code, "bad ask")
	}
	code, obj, _ := volk.do("POST", "/api/buys", map[string]any{"link": "https://amzn.eu/d/abc12"})
	volk.must(201, code, "ask")
	id := int64(obj["id"].(float64))
	path := "/api/buys/" + itoa64(id)

	code, _, _ = volk.do("PUT", path, map[string]any{"state": "bought"})
	volk.must(400, code, "own ask")
	code, _, _ = volk.do("DELETE", path, nil)
	volk.must(204, code, "the asker deletes an open ask")
	_, obj, _ = volk.do("POST", "/api/buys", map[string]any{"text": "svedri za les 6 mm", "link": "https://amzn.eu/d/abc12"})
	id = int64(obj["id"].(float64))
	path = "/api/buys/" + itoa64(id)

	// The asker's phones hear who bought it, and who took it back, each in
	// its own language. Nobody else hears either.
	for _, l := range []string{"sl", "en"} {
		volk.do("POST", "/api/push/subscribe", map[string]any{"endpoint": "https://push.example/volk-" + l, "lang": l, "keys": map[string]any{"p256dh": "p", "auth": "a"}})
	}
	rings := func(what string, put map[string]any, sl, en string) {
		t.Helper()
		fake.mu.Lock()
		fake.sent, fake.payloads = nil, nil
		fake.mu.Unlock()
		code, _, _ := steward.do("PUT", path, put)
		steward.must(204, code, what)
		waitFor(t, 2, fake)
		time.Sleep(50 * time.Millisecond)
		fake.mu.Lock()
		defer fake.mu.Unlock()
		if len(fake.sent) != 2 {
			t.Fatalf("%s rang %d phones", what, len(fake.sent))
		}
		for i, sub := range fake.sent {
			var p Payload
			json.Unmarshal(fake.payloads[i], &p)
			want := map[string]string{"sl": sl, "en": en}[sub.Lang]
			if p.Title != want || p.Body != "svedri za les 6 mm" || p.URL != "#/market?at=buys" {
				t.Fatalf("%s push [%s]: %+v", what, sub.Lang, p)
			}
		}
	}
	rings("bought", map[string]any{"state": "bought"}, "🛍 S vam to kupi", "🛍 S bought this for you")
	rings("taken back", map[string]any{"state": "open"}, "🛍 S tega vendarle ne kupi", "🛍 S did not buy it after all")
	code, _, _ = steward.do("PUT", path, map[string]any{"state": "bought"})
	steward.must(204, code, "bought again")
	code, _, _ = breza.do("PUT", path, map[string]any{"state": "bought"})
	breza.must(409, code, "bought twice")
	code, _, _ = volk.do("DELETE", path, nil)
	volk.must(409, code, "a bought ask is the buyer's note")

	// Paid back: the buyer's mark, nobody else's — the asker included.
	for _, c := range []*client{volk, breza} {
		code, _, _ = c.do("PUT", path, map[string]any{"repaid": true})
		c.must(403, code, "someone else's note")
	}
	code, _, _ = steward.do("PUT", path, map[string]any{"repaid": true})
	steward.must(204, code, "repaid")
	row := func(c *client) map[string]any {
		_, _, list := c.do("GET", "/api/buys", nil)
		for _, x := range list {
			if int64(x["id"].(float64)) == id {
				return x
			}
		}
		return nil
	}
	if r := row(steward); r["repaid_at"] == nil {
		t.Fatalf("the buyer reads its note: %v", r)
	}
	// Bought, the ask leaves every list but the buyer's, the asker's too.
	for _, c := range []*client{volk, breza} {
		if r := row(c); r != nil {
			t.Fatalf("another house reads a bought ask: %v", r)
		}
	}

	// Taken back: open again, the note gone with it.
	code, _, _ = steward.do("PUT", path, map[string]any{"state": "open"})
	steward.must(204, code, "taken back")
	if r := row(steward); r["bought_at"] != nil || r["repaid_at"] != nil {
		t.Fatalf("taken back: %v", r)
	}

	// The buyer keeps an unpaid ask as long as it is unpaid, a ticked one a
	// month after the tick.
	steward.do("PUT", path, map[string]any{"state": "bought"})
	start := srv.now()
	srv.now = func() time.Time { return start.AddDate(0, 0, 31) }
	if row(steward) == nil {
		t.Fatal("an unpaid ask left the buyer's list")
	}
	steward.do("PUT", path, map[string]any{"repaid": true})
	srv.now = func() time.Time { return start.AddDate(0, 0, 62) }
	if row(steward) != nil {
		t.Fatal("a paid-back ask stays forever")
	}

	// A steward does not tick another house's note either.
	_, obj, _ = breza.do("POST", "/api/buys", map[string]any{"text": "žarnica"})
	bulb := "/api/buys/" + itoa64(int64(obj["id"].(float64)))
	volk.do("PUT", bulb, map[string]any{"state": "bought"})
	code, _, _ = steward.do("PUT", bulb, map[string]any{"repaid": true})
	steward.must(403, code, "a steward on a buyer's note")

	// A house that leaves: its open asks go, a bought one stays the buyer's.
	_, obj, _ = breza.do("POST", "/api/buys", map[string]any{"text": "open"})
	openID := obj["id"].(float64)
	_, obj, _ = breza.do("POST", "/api/buys", map[string]any{"text": "kupljeno"})
	keptID := int64(obj["id"].(float64))
	steward.do("PUT", "/api/buys/"+itoa64(keptID), map[string]any{"state": "bought"})
	_, _, houses := steward.do("GET", "/api/houses", nil)
	for _, h := range houses {
		if h["name"] == "Breza" {
			code, _, _ = steward.do("DELETE", "/api/houses/"+itoa64(int64(h["id"].(float64))), nil)
			steward.must(204, code, "delete house")
		}
	}
	_, _, list := steward.do("GET", "/api/buys", nil)
	kept := false
	for _, x := range list {
		if x["id"].(float64) == openID {
			t.Fatal("an open ask outlived its house")
		}
		if int64(x["id"].(float64)) == keptID {
			kept = x["house_id"] == nil && x["house_name"] == nil
		}
	}
	if !kept {
		t.Fatalf("the buyer's note went with the house: %v", list)
	}
	// Taken back, an ask whose house left goes, rather than wait as an open
	// card nobody asked for.
	code, _, _ = steward.do("PUT", "/api/buys/"+itoa64(keptID), map[string]any{"state": "open"})
	steward.must(204, code, "taken back, the asker gone")
	if r, _ := srv.st.One(t.Context(), `SELECT id FROM buys WHERE id=?`, keptID); r != nil {
		t.Fatal("an ownerless ask came back open")
	}
}
