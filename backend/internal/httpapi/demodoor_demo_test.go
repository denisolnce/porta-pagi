//go:build demo

package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dennislapchenko/porta-pagi/backend/internal/config"
	"github.com/dennislapchenko/porta-pagi/backend/internal/store"
)

// TestDemoDoor (demo build only): a visitor lists the houses and enters one
// with no code; the reset leaves the village whole again.
func TestDemoDoor(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := New(st, config.Config{Languages: []string{"pt", "en"}, VillageName: "Sobreiro Torto"})
	srv.now = func() time.Time { return time.Date(2026, 9, 23, 10, 0, 0, 0, time.Local) }
	if err := srv.SeedDemo(srv.now()); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/demo/houses", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Casa da Eira") || strings.Contains(rec.Body.String(), "Largo") {
		t.Fatalf("houses: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/api/demo/enter", strings.NewReader(`{"id":1}`)))
	if rec.Code != 201 || !strings.Contains(rec.Body.String(), `"token"`) {
		t.Fatalf("enter: %d %s", rec.Code, rec.Body.String())
	}
	tok := strings.Split(strings.Split(rec.Body.String(), `"token":"`)[1], `"`)[0]
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Casa da Eira") {
		t.Fatalf("me as the visitor: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := st.Exec(t.Context(), `DELETE FROM posts`); err != nil {
		t.Fatal(err)
	}
	if err := srv.ResetDemo(); err != nil {
		t.Fatal(err)
	}
	if row, _ := st.One(t.Context(), `SELECT count(*) AS n FROM posts`); row["n"].(int64) < 20 {
		t.Fatalf("posts after the reset: %v", row)
	}
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), `content="1"`) {
		t.Fatal("the page does not say it is the demo")
	}
}
