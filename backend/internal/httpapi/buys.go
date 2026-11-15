package httpapi

import (
	"net/http"
	"net/url"
	"strings"
)

// The Market's wishlist (owner's ask 2026-11-15; buys in the code, because
// the Shed's wishlist of tools already has wishes). A house asks the
// village to buy a thing for it — one field, a few words or a link to an
// online shop. Any other house marks it bought, and the asker hears who; the
// ask then leaves every list but the buyer's (owner, 2026-11-15: no history).
// Whether it was paid back is the buyer's own note: repaid_at is read and
// written by the buyer alone, never summed, never pushed.

// buyWindow: days a ticked ask stays on the buyer's list after the tick.
const buyWindow = 30

func (s *Server) listBuys(w http.ResponseWriter, r *http.Request) {
	me := houseFrom(r).ID
	cut := s.now().UTC().AddDate(0, 0, -buyWindow).Format("2006-01-02 15:04:05")
	// Open asks, and what this house bought: unpaid for as long as it is
	// unpaid (a debt does not expire), ticked for a month after the tick.
	// A bought ask reaches its buyer and nobody else, not even the asker.
	rows, err := s.st.Rows(r.Context(), `SELECT x.id, x.house_id, x.text, x.link, x.bought_by, x.bought_at, x.repaid_at, x.created_at,`+houseJoin+`
		FROM buys x LEFT JOIN houses h ON h.id=x.house_id
		WHERE x.bought_at IS NULL OR (x.bought_by = ? AND (x.repaid_at IS NULL OR x.repaid_at >= ?))
		ORDER BY x.created_at DESC LIMIT 500`, me, cut)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, rows)
}

// webLink: an address the page may put in an href. Anything but http or https
// (javascript:, data:) would run in every villager's browser on a tap.
func webLink(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && !strings.ContainsAny(s, " \t\n")
}

// buyLabel: what a banner calls the thing — the asker's words, or the link
// without its scheme when the asker dropped a link alone.
func buyLabel(text, link string) string {
	if text != "" {
		return text
	}
	return strings.TrimPrefix(strings.TrimPrefix(link, "https://"), "http://")
}

func (s *Server) createBuy(w http.ResponseWriter, r *http.Request) {
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	text, link := strings.TrimSpace(str(m, "text")), strings.TrimSpace(str(m, "link"))
	switch {
	case text == "" && link == "":
		writeErr(w, 400, "a link or a few words")
		return
	case link != "" && (len(link) > 2000 || !webLink(link)):
		writeErr(w, 400, "link must be an http or https address")
		return
	case len([]rune(text)) > 2000: // words with a pasted address in them
		writeErr(w, 400, "too long")
		return
	}
	h := houseFrom(r)
	id, err := s.st.Exec(r.Context(), `INSERT INTO buys(house_id, text, link) VALUES (?,?,?)`, h.ID, text, link)
	if err != nil {
		fail(w, err)
		return
	}
	s.notify("buys", h.ID, func(lang string) Payload {
		return Payload{Title: "🛍 " + h.Name + tr(lang, " asks someone to buy"), Body: snippet(buyLabel(text, link), 140), URL: "#/market?at=buys"}
	})
	writeJSON(w, 201, map[string]any{"id": id})
}

// updateBuy: state=bought (any house but the asker, from open), state=open
// (the buyer takes it back, and its note with it), repaid=true|false (the
// buyer's own note). Nobody else touches a bought ask, a steward included:
// the note is the buyer's, and a mark it did not make is not its note.
func (s *Server) updateBuy(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	row, err := s.st.One(r.Context(), `SELECT house_id, text, link, bought_by, bought_at FROM buys WHERE id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if row == nil {
		writeErr(w, 404, "not found")
		return
	}
	buyer := row["bought_by"] != nil && row["bought_by"].(int64) == h.ID
	asker, hasAsker := row["house_id"].(int64)
	label := buyLabel(row["text"].(string), row["link"].(string))
	now := s.now().UTC().Format("2006-01-02 15:04:05")
	repaid, hasRepaid := m["repaid"].(bool)
	switch state := str(m, "state"); {
	case state == "bought":
		if hasAsker && asker == h.ID {
			writeErr(w, 400, "your own ask")
			return
		}
		n, err := s.st.ExecN(r.Context(), `UPDATE buys SET bought_by=?, bought_at=? WHERE id=? AND bought_at IS NULL`, h.ID, now, id)
		if err != nil {
			fail(w, err)
			return
		}
		if n == 0 {
			writeErr(w, 409, "already bought")
			return
		}
		if hasAsker {
			s.notifyHouse("buys", asker, func(lang string) Payload {
				return Payload{Title: "🛍 " + h.Name + tr(lang, " bought this for you"), Body: snippet(label, 140), URL: "#/market?at=buys"}
			})
		}
	case state == "open":
		if !buyer {
			writeErr(w, 403, "not yours")
			return
		}
		// An ask whose house has left is not asked any more: taken back, it
		// goes, as the open asks went with the house (032_buys.sql).
		q := `UPDATE buys SET bought_by=NULL, bought_at=NULL, repaid_at=NULL WHERE id=?`
		if !hasAsker {
			q = `DELETE FROM buys WHERE id=?`
		}
		if _, err := s.st.Exec(r.Context(), q, id); err != nil {
			fail(w, err)
			return
		}
		// The asker heard it was bought, and would wait for a parcel.
		if hasAsker {
			s.notifyHouse("buys", asker, func(lang string) Payload {
				return Payload{Title: "🛍 " + h.Name + tr(lang, " did not buy it after all"), Body: snippet(label, 140), URL: "#/market?at=buys"}
			})
		}
	case hasRepaid:
		if !buyer {
			writeErr(w, 403, "not yours")
			return
		}
		var at any
		if repaid {
			at = now
		}
		if _, err := s.st.Exec(r.Context(), `UPDATE buys SET repaid_at=? WHERE id=?`, at, id); err != nil {
			fail(w, err)
			return
		}
	default:
		writeErr(w, 400, "state bought or open, or repaid")
		return
	}
	w.WriteHeader(204)
}

// deleteBuy: the asker or a steward, and only while nobody bought it — once
// bought, deleting would wipe the buyer's note of what it is owed for.
func (s *Server) deleteBuy(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeErr(w, 400, "bad id")
		return
	}
	row, err := s.st.One(r.Context(), `SELECT house_id, bought_at FROM buys WHERE id=?`, id)
	if err != nil {
		fail(w, err)
		return
	}
	if row == nil {
		writeErr(w, 404, "not found")
		return
	}
	if row["bought_at"] != nil {
		writeErr(w, 409, "bought — the buyer takes it back first")
		return
	}
	h := houseFrom(r)
	if asker, _ := row["house_id"].(int64); asker != h.ID && !h.IsSteward {
		writeErr(w, 403, "not yours")
		return
	}
	if _, err := s.st.Exec(r.Context(), `DELETE FROM buys WHERE id=?`, id); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}
