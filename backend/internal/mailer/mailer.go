// Package mailer sends one email through Resend's HTTP API, stdlib only —
// the shape of the client truetothebook runs (its mailer.py): one POST, a
// bearer key, an explicit User-Agent, a 30 s timeout, no retry.
package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Endpoint is a variable so a test can point it at a local server.
var Endpoint = "https://api.resend.com/emails"

// Message is what Resend takes; an empty ReplyTo or HTML is left out.
type Message struct {
	From    string `json:"from"`
	To      string `json:"-"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html,omitempty"`
	ReplyTo string `json:"reply_to,omitempty"`
}

var client = &http.Client{Timeout: 30 * time.Second}

// Send posts one message. A non-2xx answer is an error carrying Resend's own
// words (a domain not verified, a bad key), so the log says what to fix.
func Send(ctx context.Context, key string, m Message) error {
	body, err := json.Marshal(struct {
		Message
		To []string `json:"to"`
	}{m, []string{m.To}})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	// Cloudflare in front of Resend refuses the default library agents.
	req.Header.Set("User-Agent", "porta-pagi/1.0")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("resend: %d %s", res.StatusCode, bytes.TrimSpace(msg))
	}
	return nil
}
