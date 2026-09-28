package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestFeatures: a switch reads its default until a steward flips it, a
// house that is not a steward cannot, and a switched-off feature's guard
// answers 404 as if the route were never built.
func TestFeatures(t *testing.T) {
	srv, _, steward, house := newVillage(t)
	code, _, list := house.do("GET", "/api/features", nil)
	if code != 200 || fmt.Sprint(list) == "" {
		t.Fatalf("list: %d %v", code, list)
	}
	for _, f := range list {
		if f["key"] == "dues" && f["on"] != false {
			t.Fatalf("dues is off by default: %v", f)
		}
	}
	if c, _, _ := house.do("PUT", "/api/features/dues", map[string]any{"on": true}); c != 403 {
		t.Fatalf("a house that is not a steward flipped a switch: %d", c)
	}
	if c, _, _ := steward.do("PUT", "/api/features/nope", map[string]any{"on": true}); c != 404 {
		t.Fatalf("an unknown switch: %d", c)
	}
	if c, _, _ := steward.do("PUT", "/api/features/dues", map[string]any{"on": true}); c != 200 {
		t.Fatalf("the steward flips dues: %d", c)
	}
	if !srv.feature(t.Context(), "dues") {
		t.Fatal("dues stayed off after the steward switched it on")
	}
	rec := httptest.NewRecorder()
	srv.requireFeature("roster", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 404 {
		t.Fatalf("a switched-off feature answered %d", rec.Code)
	}
	if !srv.offKinds(t.Context())["camp"] {
		t.Fatal("camp is off in a new village, so its push kind is off too")
	}
}

// switchOn: the steward flips one switch on, as the settings page does. A new
// village has only the defaults, and most are off.
func switchOn(t *testing.T, steward *client, key string) {
	t.Helper()
	code, _, _ := steward.do("PUT", "/api/features/"+key, map[string]any{"on": true})
	steward.must(200, code, "switch "+key+" on")
}

// TestCampSwitch: in a new village the campground is absent — its routes are
// 404s, its MCP tools are neither listed nor callable, its push kind is not
// offered — and the steward's switch brings all three back. A house's own
// choice about the kind waits for it while it is off.
func TestCampSwitch(t *testing.T) {
	srv, fake, steward, volk := newVillage(t)
	steward.must(204, first(steward.do("POST", "/api/push/subscribe", map[string]any{"endpoint": "https://push.example/s", "lang": "sl", "keys": map[string]any{"p256dh": "p", "auth": "a"}})), "subscribe")
	// Nothing of a switched-off kind rings, whoever sends it; a kind that is
	// on still does.
	srv.notify("camp", 0, func(string) Payload { return Payload{Title: "camp"} })
	srv.notify("posts", 0, func(string) Payload { return Payload{Title: "posts"} })
	waitFor(t, 1, fake)
	fake.mu.Lock()
	sent := string(fake.payloads[0])
	fake.mu.Unlock()
	if !strings.Contains(sent, `"posts"`) {
		t.Fatalf("the send that went out: %s", sent)
	}
	for _, rt := range []struct{ method, path string }{
		{"GET", "/api/camp"}, {"POST", "/api/camp"}, {"PUT", "/api/camp/1"}, {"DELETE", "/api/camp/1"},
	} {
		if code, _, _ := volk.do(rt.method, rt.path, map[string]any{}); code != 404 {
			t.Fatalf("%s %s with camp off: %d", rt.method, rt.path, code)
		}
	}
	key, _ := mintKey(t, volk, "agent")
	listed := func() map[string]bool {
		_, out := rpc(t, srv, key, "tools/list", nil)
		names := map[string]bool{}
		for _, x := range out["result"].(map[string]any)["tools"].([]any) {
			names[x.(map[string]any)["name"].(string)] = true
		}
		return names
	}
	for _, name := range []string{"list_camp", "create_camp", "update_camp"} {
		if listed()[name] {
			t.Fatalf("%s listed with camp off", name)
		}
	}
	if _, out := rpc(t, srv, key, "tools/call", map[string]any{"name": "list_camp"}); out["error"] == nil {
		t.Fatalf("list_camp answered with camp off: %v", out)
	}
	kinds := func() string {
		_, obj, _ := volk.do("GET", "/api/me/prefs", nil)
		return fmt.Sprint(obj["kinds"])
	}
	if strings.Contains(kinds(), "camp") {
		t.Fatalf("prefs offer camp while it is off: %s", kinds())
	}

	// The house muted camp while it was on; the switch goes off, the house
	// saves its other choices, the switch comes back: the mute is still there.
	// The same for a steward's village-wide mute, and its who stays with it.
	switchOn(t, steward, "camp")
	volk.must(204, first(volk.do("PUT", "/api/me/prefs", map[string]any{"off": []string{"camp"}})), "mute camp")
	steward.must(204, first(steward.do("PUT", "/api/prefs/global", map[string]any{"off": []string{"camp"}})), "mute camp for the village")
	steward.must(200, first(steward.do("PUT", "/api/features/camp", map[string]any{"on": false})), "camp off")
	_, obj, _ := volk.do("GET", "/api/me/prefs", nil)
	if fmt.Sprint(obj["off"], obj["global_off"], obj["global_detail"]) != "[] [] []" {
		t.Fatalf("a switched-off kind still shows in prefs: %v", obj)
	}
	volk.must(204, first(volk.do("PUT", "/api/me/prefs", map[string]any{"off": []string{"needs"}})), "mute needs")
	steward.must(204, first(steward.do("PUT", "/api/prefs/global", map[string]any{"off": []string{"needs"}})), "mute needs for the village")
	switchOn(t, steward, "camp")
	_, obj, _ = volk.do("GET", "/api/me/prefs", nil)
	if off := fmt.Sprint(obj["off"]); !strings.Contains(off, "camp") || !strings.Contains(off, "needs") {
		t.Fatalf("a house's camp mute did not survive the switch: %s", off)
	}
	if goff := fmt.Sprint(obj["global_off"], obj["global_detail"]); !strings.Contains(goff, "camp") || !strings.Contains(goff, "needs") || !strings.Contains(goff, "set_by:S") {
		t.Fatalf("the village-wide camp mute did not survive the switch: %s", goff)
	}

	if !strings.Contains(kinds(), "camp") {
		t.Fatalf("prefs lack camp while it is on: %s", kinds())
	}
	for _, name := range []string{"list_camp", "create_camp", "update_camp"} {
		if !listed()[name] {
			t.Fatalf("%s missing with camp on", name)
		}
	}
	if code, _, _ := volk.do("GET", "/api/camp", nil); code != 200 {
		t.Fatalf("GET /api/camp with camp on: %d", code)
	}
}

func first(code int, _ map[string]any, _ []map[string]any) int { return code }
