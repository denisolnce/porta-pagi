package httpapi

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/denisolnce/porta-pagi/backend/internal/config"
	"github.com/denisolnce/porta-pagi/backend/internal/mailer"
	"github.com/denisolnce/porta-pagi/backend/internal/store"
)

// rowCount sums every table: the feedback door must leave it where it was.
func rowCount(t *testing.T, st *store.Store) int64 {
	tables, err := st.Rows(context.Background(), `SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		t.Fatal(err)
	}
	var n int64
	for _, tb := range tables {
		r, err := st.One(context.Background(), `SELECT count(*) AS n FROM "`+tb["name"].(string)+`"`)
		if err != nil {
			t.Fatal(err)
		}
		n += r["n"].(int64)
	}
	return n
}

func feedbackVillage(t *testing.T, cfg config.Config) (*Server, *store.Store, *client) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	cfg.BootstrapCode, cfg.VillageName, cfg.Languages, cfg.PublicURL = "x", "Testna vas", []string{"sl", "en"}, "https://portal.example.org"
	srv := New(st, cfg)
	c := &client{t: t, h: srv.Handler()}
	_, obj, _ := c.do("POST", "/api/bootstrap", map[string]any{"code": "x", "name": "Zeleni Hrast"})
	c.token = obj["token"].(string)
	return srv, st, c
}

// TestFeedbackEmail: with a key and an address the message goes to the
// sender with the village, the house, the section, the language and the
// words in it, the reply address as reply_to — and nothing lands in the DB.
func TestFeedbackEmail(t *testing.T) {
	srv, st, c := feedbackVillage(t, config.Config{ResendKey: "re_test", FeedbackTo: "owner@example.org"})
	var sent []mailer.Message
	srv.mail = func(_ context.Context, m mailer.Message) error { sent = append(sent, m); return nil }
	before := rowCount(t, st)

	code, _, _ := c.do("POST", "/api/feedback", map[string]any{"section": "Tavern", "text": "the calendar skips a week", "reply_to": "a@example.org", "lang": "sl"})
	c.must(204, code, "feedback with reply")
	if len(sent) != 1 {
		t.Fatalf("want one mail, got %d", len(sent))
	}
	m := sent[0]
	if m.To != "owner@example.org" || m.ReplyTo != "a@example.org" || m.From != "Porta Pagi <feedback@portal.example.org>" {
		t.Fatalf("addresses: %+v", m)
	}
	if !strings.Contains(m.Subject, "Testna vas") || !strings.Contains(m.Subject, "Tavern") || !strings.Contains(m.Subject, "Zeleni Hrast") {
		t.Fatalf("subject: %q", m.Subject)
	}
	for _, want := range []string{"Zeleni Hrast", "Tavern", "sl", "a@example.org", "the calendar skips a week"} {
		if !strings.Contains(m.Text, want) {
			t.Fatalf("text misses %q:\n%s", want, m.Text)
		}
	}

	// No reply address: none is sent, a language the village does not speak
	// is dropped, and EMAIL_FROM wins over the derived sender.
	srv.cfg.EmailFrom = "Porta Pagi <hello@example.org>"
	code, _, _ = c.do("POST", "/api/feedback", map[string]any{"section": "Map", "text": "hi", "lang": "xx"})
	c.must(204, code, "feedback without reply")
	if m := sent[1]; m.ReplyTo != "" || m.From != "Porta Pagi <hello@example.org>" || strings.Contains(m.Text, "xx") {
		t.Fatalf("second mail: %+v", m)
	}

	if after := rowCount(t, st); after != before {
		t.Fatalf("feedback wrote to the database: %d rows before, %d after", before, after)
	}

	// A send that fails is an error the page can show, and it is not retried.
	calls := 0
	srv.mail = func(context.Context, mailer.Message) error {
		calls++
		return errors.New("resend: 403 domain not verified")
	}
	code, obj, _ := c.do("POST", "/api/feedback", map[string]any{"text": "again"})
	c.must(502, code, "failed send")
	if calls != 1 || !strings.Contains(obj["error"].(string), "try again later") {
		t.Fatalf("failure: %d calls, %v", calls, obj)
	}
}

// TestFeedbackLog: without a key the message is written to the log in full
// and the house is told it went.
func TestFeedbackLog(t *testing.T) {
	srv, _, c := feedbackVillage(t, config.Config{FeedbackTo: "owner@example.org"})
	srv.mail = func(context.Context, mailer.Message) error { t.Fatal("no key, yet the sender was called"); return nil }
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	code, _, _ := c.do("POST", "/api/feedback", map[string]any{"section": "Codex", "text": "a typo in section two", "reply_to": "a@example.org"})
	log.SetOutput(os.Stderr)
	c.must(204, code, "feedback to the log")
	out := buf.String()
	for _, want := range []string{"FEEDBACK", "Zeleni Hrast", "Codex", "a typo in section two", "a@example.org"} {
		if !strings.Contains(out, want) {
			t.Fatalf("log misses %q:\n%s", want, out)
		}
	}
}

// TestFeedbackRefuses: the text is required and capped, the section is one
// short line, a reply address is one bare address, and a phone that keeps
// sending is stopped.
func TestFeedbackRefuses(t *testing.T) {
	srv, _, c := feedbackVillage(t, config.Config{})
	srv.mail = func(context.Context, mailer.Message) error { return nil }
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	for _, body := range []map[string]any{
		{"section": "Tavern", "text": "  "},
		{"section": "Tavern", "text": strings.Repeat("ž", 4001)},
		{"section": strings.Repeat("a", 61), "text": "hi"},
		{"text": "hi", "reply_to": "not an email"},
		{"text": "hi", "reply_to": "Someone <a@example.org>"},
		{"text": "hi", "reply_to": "a@example.org, b@example.org"},
		{"text": "hi", "reply_to": "a@localhost"},
	} {
		code, _, _ := c.do("POST", "/api/feedback", body)
		c.must(400, code, "refused body")
	}
	code, _, _ := c.do("POST", "/api/feedback", map[string]any{"text": strings.Repeat("ž", 4000)})
	c.must(204, code, "4000 characters is allowed")

	// The cap is per IP; the refusals above did not spend it.
	for i := 0; i < 4; i++ {
		code, _, _ := c.do("POST", "/api/feedback", map[string]any{"text": "hi"})
		c.must(204, code, "under the cap")
	}
	code, _, _ = c.do("POST", "/api/feedback", map[string]any{"text": "hi"})
	c.must(429, code, "over the cap")

	// Over the village's daily cap a message is logged, not emailed, and the
	// house is told so. The sender here would fail if called.
	srv.cfg.ResendKey, srv.cfg.FeedbackTo = "re_test", "owner@example.org"
	srv.mail = func(context.Context, mailer.Message) error { return errors.New("over the cap, yet sent") }
	for i := 0; i < 50; i++ {
		srv.tries.allow("feedback", 50, 24*time.Hour)
	}
	buf.Reset()
	req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(`{"text":"past the cap"}`))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-Forwarded-For", "198.51.100.7") // another phone: this one spent its own cap above
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	c.must(503, rec.Code, "over the village cap")
	if out := buf.String(); !strings.Contains(out, "over 50 today") || !strings.Contains(out, "past the cap") {
		t.Fatalf("village cap log:\n%s", out)
	}

	anon := &client{t: t, h: srv.Handler()}
	code, _, _ = anon.do("POST", "/api/feedback", map[string]any{"text": "hi"})
	anon.must(401, code, "no token")
}
