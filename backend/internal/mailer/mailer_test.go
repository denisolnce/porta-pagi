package mailer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSend: the request is the one Resend documents — bearer key, an
// explicit agent, `to` as a list, empty optional fields left out — and a
// refusal comes back as an error with Resend's words in it.
func TestSend(t *testing.T) {
	var got map[string]any
	var auth, agent string
	status := 200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, agent = r.Header.Get("Authorization"), r.Header.Get("User-Agent")
		json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(status)
		w.Write([]byte(`{"message":"domain is not verified"}`))
	}))
	defer srv.Close()
	old := Endpoint
	Endpoint = srv.URL
	t.Cleanup(func() { Endpoint = old })

	m := Message{From: "V <f@x.test>", To: "o@x.test", Subject: "s", Text: "t"}
	if err := Send(context.Background(), "re_k", m); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer re_k" || !strings.HasPrefix(agent, "porta-pagi/") {
		t.Fatalf("headers: %q %q", auth, agent)
	}
	if to, _ := got["to"].([]any); len(to) != 1 || to[0] != "o@x.test" || got["from"] != "V <f@x.test>" || got["text"] != "t" {
		t.Fatalf("body: %v", got)
	}
	if _, ok := got["reply_to"]; ok {
		t.Fatalf("empty reply_to sent: %v", got)
	}
	if _, ok := got["html"]; ok {
		t.Fatalf("empty html sent: %v", got)
	}

	m.ReplyTo = "a@b.test"
	Send(context.Background(), "re_k", m)
	if got["reply_to"] != "a@b.test" {
		t.Fatalf("reply_to: %v", got)
	}

	status = 403
	err := Send(context.Background(), "re_k", m)
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "not verified") {
		t.Fatalf("refusal: %v", err)
	}
}
