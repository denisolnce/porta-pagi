package httpapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/denisolnce/porta-pagi/backend/internal/config"
	"github.com/denisolnce/porta-pagi/backend/internal/store"
)

// TestNewsFile: every note has an id that starts with its day, newest first,
// and says itself in every language the portal ships a dictionary for.
func TestNewsFile(t *testing.T) {
	if len(news) == 0 {
		t.Fatal("no Portal News at all")
	}
	seen := map[string]bool{}
	for i, e := range news {
		if seen[e.ID] || !strings.HasPrefix(e.ID, e.On+"-") {
			t.Errorf("note %q: a duplicate, or its id does not start with its day %s", e.ID, e.On)
		}
		seen[e.ID] = true
		if i > 0 && e.On > news[i-1].On {
			t.Errorf("note %q is newer than the one above it", e.ID)
		}
		// A push carries 120 characters of body: a note says itself in that,
		// or the lock screen cuts it mid-sentence.
		for _, lang := range []string{"en", "sl", "pt"} {
			if tx := e.Texts[lang]; tx.Title == "" || tx.Body == "" || len([]rune(tx.Body)) > 120 {
				t.Errorf("note %q: no %s title or body, or a body over 120 characters", e.ID, lang)
			}
		}
	}
}

// TestNewsRing: a roll's notes ring once, in one push; quiet hours hold them;
// a house that switched the kind off hears nothing; a village with no houses
// marks the past as told.
func TestNewsRing(t *testing.T) {
	srv, fake, steward, volk := newVillage(t) // 2026-09-04 10:00
	ctx := context.Background()
	for i, c := range []*client{steward, volk} {
		c.must(204, first(c.do("POST", "/api/push/subscribe", map[string]any{
			"endpoint": "https://push.example/n" + itoa(float64(i)), "lang": []string{"sl", "en"}[i], "keys": map[string]any{"p256dh": "p", "auth": "a"}})), "subscribe")
	}
	_, _, rows := volk.do("GET", "/api/news", nil)
	if len(rows) != len(news) || rows[0]["id"] != news[0].ID {
		t.Fatalf("the list: %v", rows)
	}

	srv.now = func() time.Time { return time.Date(2026, 9, 4, 22, 0, 0, 0, time.Local) }
	if n, err := srv.ringNews(ctx); err != nil || n != 0 {
		t.Fatalf("rang at night: %d %v", n, err)
	}
	srv.now = func() time.Time { return time.Date(2026, 9, 4, 7, 30, 0, 0, time.Local) }
	if n, err := srv.ringNews(ctx); err != nil || n != len(news) {
		t.Fatalf("the morning's round: %d %v", n, err)
	}
	waitFor(t, 2, fake)
	payloads := func() map[string]Payload {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		out := map[string]Payload{}
		for i, sub := range fake.sent {
			var p Payload
			json.Unmarshal(fake.payloads[i], &p)
			out[sub.Lang] = p
		}
		fake.sent, fake.payloads = nil, nil
		return out
	}
	got := payloads()
	if len(news) > 1 {
		if got["sl"].Title != "📰 Novice portala" || got["en"].Title != "📰 Portal News" || !strings.Contains(got["en"].Body, news[0].Texts["en"].Title+" · ") || got["en"].URL != "#/?news" {
			t.Fatalf("several notes in one push: %+v", got)
		}
	}
	if n, _ := srv.ringNews(ctx); n != 0 {
		t.Fatal("told twice")
	}

	// The next roll brings one note: it rings by its own title, and not for
	// the house that switched the kind off.
	srv.st.Exec(ctx, `DELETE FROM settings WHERE key=?`, "news:"+news[0].ID)
	volk.must(204, first(volk.do("PUT", "/api/me/prefs", map[string]any{"off": []any{"news"}})), "switch off")
	if n, _ := srv.ringNews(ctx); n != 1 {
		t.Fatalf("one new note: %d", n)
	}
	waitFor(t, 1, fake)
	got = payloads()
	if len(got) != 1 || got["sl"].Title != "📰 "+news[0].Texts["sl"].Title || got["sl"].Body != snippet(news[0].Texts["sl"].Body, 120) {
		t.Fatalf("one note: %+v", got)
	}

	// A village with nobody in it marks the past and rings nothing.
	st, _ := store.Open(t.TempDir())
	defer st.Close()
	bare := New(st, config.Config{Languages: []string{"sl", "en"}})
	bare.now = srv.now
	if n, err := bare.ringNews(ctx); err != nil || n != 0 {
		t.Fatalf("an empty village rang: %d %v", n, err)
	}
	if row, _ := st.One(ctx, `SELECT count(*) AS n FROM settings WHERE key LIKE 'news:%'`); row["n"].(int64) != int64(len(news)) {
		t.Fatalf("an empty village kept the past untold: %v", row)
	}
}
