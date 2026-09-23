//go:build !demo

package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestDemoDoorAbsent: every village's binary is built without the demo tag,
// and in it the door with no code does not exist — not disabled, absent. A
// status answer says so too, so the page never shows the house switcher.
func TestDemoDoorAbsent(t *testing.T) {
	srv, _, _, _ := newVillage(t)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/demo/houses", nil))
	if rec.Code != 404 {
		t.Fatalf("the demo door answers in a village build: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/api/demo/enter", strings.NewReader(`{"id":1}`)))
	if rec.Code != 404 {
		t.Fatalf("the demo door answers in a village build: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/status", nil))
	if !strings.Contains(rec.Body.String(), `"demo":false`) {
		t.Fatalf("status in a village build: %s", rec.Body.String())
	}
}
