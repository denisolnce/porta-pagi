package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
