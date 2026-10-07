package httpapi

import (
	"context"
	"net/http"
	"time"
)

// A birthday is a date for the village to know, not a party to come to: an
// event of kind birthday, yearly, whose starts_at is the birth itself — the
// year, and the hour when somebody knows it, because a house that keeps a
// horoscope wants both. It belongs to no house: the house that wrote it down
// may take it away again, and nothing else on the card says whose it is.
// It takes no answers, rings nobody when it is written down, and rings every
// house on the morning of the day, under its own kind, which a house switches
// off like any other.

var recurWords = map[string]string{"weekly": "every week", "monthly": "every month", "yearly": "every year"}

func (s *Server) createBirthday(w http.ResponseWriter, r *http.Request, m map[string]any) {
	born := str(m, "starts_at")
	if _, err := time.Parse("2006-01-02", born); err != nil {
		if _, err := time.Parse("2006-01-02T15:04", born); err != nil {
			writeErr(w, 400, "starts_at is the birth: YYYY-MM-DD, or YYYY-MM-DDTHH:MM when the hour is known")
			return
		}
	}
	id, err := s.st.Exec(r.Context(), `INSERT INTO events(house_id, title, kind, starts_at, notes, recur) VALUES (?,?, 'birthday', ?,?, 'yearly')`,
		houseFrom(r).ID, str(m, "title"), born, str(m, "notes"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"id": id})
}

// occurrence reads the day an answer is for. A one-off event has one day, and
// its answer keeps the day empty; a series needs the day, and only one it
// lands on.
func (s *Server) occurrence(ev map[string]any, on string) (string, bool) {
	if str(ev, "recur") == "" {
		return "", true
	}
	return on, occursOn(str(ev, "starts_at"), str(ev, "recur"), on)
}

// ringBirthdays tells every house, once on the day, whose birthday it is. It
// waits for the morning: quiet hours keep it until 07:00, and the day is
// stamped in settings so a restart does not ring it twice.
func (s *Server) ringBirthdays(ctx context.Context) (int, error) {
	if s.quietHours() {
		return 0, nil
	}
	today := s.now().Format("2006-01-02")
	if row, err := s.st.One(ctx, `SELECT value FROM settings WHERE key='birthdays_rung'`); err != nil || (row != nil && row["value"] == today) {
		return 0, err
	}
	rows, err := s.st.Rows(ctx, `SELECT title, starts_at FROM events WHERE kind='birthday'`)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, b := range rows {
		if !occursOn(str(b, "starts_at"), "yearly", today) {
			continue
		}
		name := str(b, "title")
		s.notify("birthdays", 0, func(lang string) Payload {
			return Payload{Title: "🎂 " + name, Body: tr(lang, "birthday today"), URL: "#/tavern?day=" + today}
		})
		n++
	}
	_, err = s.st.Exec(ctx, `INSERT INTO settings(key, value) VALUES ('birthdays_rung', ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, today)
	return n, err
}
