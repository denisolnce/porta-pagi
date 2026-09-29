package httpapi

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/denisolnce/porta-pagi/backend/internal/mailer"
)

// The feedback door: a house writes to whoever builds the portal, from the
// foot of Home. Nothing is stored — the message leaves as an email through
// Resend when RESEND_API_KEY and FEEDBACK_TO are set, and is written to the
// server log otherwise, so a village or the demo loses nothing before email
// is wired. The demo hands anyone a house, so this door is public there:
// a per-IP cap, and a village-wide daily one past which the message is
// logged, not emailed, and the house is told to try tomorrow — never told
// it went. Both caps live in this process's memory: per container, and a
// restart resets them.

// A section is one short line. The page offers the rooms and a few doors;
// anything else a client sends is taken as its own words.
const feedbackSectionMax = 60
const feedbackTextMax = 4000

func (s *Server) feedback(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	text, reply := str(m, "text"), str(m, "reply_to")
	section := strings.Join(strings.Fields(str(m, "section")), " ")
	if section == "" {
		section = "Something else"
	}
	switch {
	case text == "":
		writeErr(w, 400, "text is required")
		return
	case utf8.RuneCountInString(text) > feedbackTextMax:
		writeErr(w, 400, fmt.Sprintf("text is longer than %d characters", feedbackTextMax))
		return
	case utf8.RuneCountInString(section) > feedbackSectionMax:
		writeErr(w, 400, "section is too long")
		return
	case reply != "" && !plausibleEmail(reply):
		writeErr(w, 400, "reply address does not look like an email")
		return
	}
	lang := str(m, "lang")
	if !slices.Contains(s.cfg.Languages, lang) {
		lang = ""
	}
	if !s.tries.allow("feedback "+clientIP(r), 5, time.Hour) {
		writeErr(w, http.StatusTooManyRequests, "too many messages, try again later")
		return
	}

	msg := s.composeFeedback(houseFrom(r).Name, section, lang, reply, text)
	if s.cfg.ResendKey != "" && s.cfg.FeedbackTo != "" && !s.tries.allow("feedback", 50, 24*time.Hour) {
		// Over the village's daily cap: the log keeps the words, the Resend
		// quota is spared, and the house is told the truth — it did not go.
		log.Printf("FEEDBACK (not emailed: over 50 today) subject=%q reply_to=%q\n%s", msg.Subject, msg.ReplyTo, msg.Text)
		// 503, not 429: this house sent nothing too many; the day is full.
		writeErr(w, http.StatusServiceUnavailable, "the builder's mailbox is full for today, try tomorrow")
		return
	}
	why := ""
	if s.cfg.ResendKey == "" || s.cfg.FeedbackTo == "" {
		why = "not emailed: RESEND_API_KEY or FEEDBACK_TO unset"
	}
	if why != "" {
		log.Printf("FEEDBACK (%s) subject=%q reply_to=%q\n%s", why, msg.Subject, msg.ReplyTo, msg.Text)
		w.WriteHeader(204)
		return
	}
	send := s.mail
	if send == nil {
		send = func(ctx context.Context, m mailer.Message) error { return mailer.Send(ctx, s.cfg.ResendKey, m) }
	}
	if err := send(r.Context(), msg); err != nil {
		// The words go to the log too: the house was told to try again, and
		// the owner can still read what it wrote.
		log.Printf("FEEDBACK (could not send: %v) subject=%q reply_to=%q\n%s", err, msg.Subject, msg.ReplyTo, msg.Text)
		writeErr(w, http.StatusBadGateway, "could not send, try again later")
		return
	}
	w.WriteHeader(204)
}

func (s *Server) composeFeedback(house, section, lang, reply, text string) mailer.Message {
	village := s.cfg.VillageName
	if DemoMode {
		village += " (demo)"
	}
	row := func(k, v string) string { return fmt.Sprintf("%-10s %s\n", k+":", v) }
	var b strings.Builder
	b.WriteString(row("Village", village))
	b.WriteString(row("House", house))
	b.WriteString(row("Section", section))
	if lang != "" {
		b.WriteString(row("Language", lang))
	}
	if reply != "" {
		b.WriteString(row("Reply to", reply))
	} else {
		b.WriteString(row("Reply to", "none given, answer in the portal"))
	}
	b.WriteString("\n" + text + "\n")
	return mailer.Message{
		From:    s.emailFrom(),
		To:      s.cfg.FeedbackTo,
		Subject: fmt.Sprintf("[%s] %s · %s", village, section, house),
		Text:    b.String(),
		ReplyTo: reply,
	}
}

// emailFrom: EMAIL_FROM, or feedback@ the portal's own host, which is the
// domain a village already owns. Resend sends only from a verified domain,
// so an unverified one fails loudly in the log and the house is told.
func (s *Server) emailFrom() string {
	if s.cfg.EmailFrom != "" {
		return s.cfg.EmailFrom
	}
	host := "localhost"
	if u, err := url.Parse(s.cfg.PublicURL); err == nil && u.Hostname() != "" {
		host = u.Hostname()
	}
	return "Porta Pagi <feedback@" + host + ">"
}

// plausibleEmail: one bare address with a dotted domain, nothing else — it
// becomes a reply_to header, so a display name or a second address is refused.
func plausibleEmail(s string) bool {
	if len(s) > 254 {
		return false
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Name != "" || a.Address != s {
		return false
	}
	at := strings.LastIndex(s, "@")
	return at > 0 && strings.Contains(s[at+1:], ".")
}
