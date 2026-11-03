package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/denisolnce/porta-pagi/backend/internal/store"
)

// House is the identity attached to every authenticated request. All members
// of a house share it; the device row tells them apart.
type House struct {
	ID        int64
	Name      string
	IsSteward bool
	DeviceID  int64
	Agent     bool // the token is an MCP key, not a phone (mcp.go)
}

type ctxKey struct{}

func houseFrom(r *http.Request) *House {
	h, _ := r.Context().Value(ctxKey{}).(*House)
	return h
}

func randomToken(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// inviteCode is short and phone-typable: 10 chars, no ambiguous letters.
func inviteCode() string {
	const alphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	b := make([]byte, 10)
	rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// requireHouse resolves the bearer token to a house or answers 401.
//
// An agent key (devices.agent = 1) is a house login with one door: it opens
// POST /api/mcp, and reaches every other handler only through the dispatcher
// in mcp.go, which marks its sub-requests with viaMCP in the context. Without
// that fence a key would be a full login with a menu in front of it — curl with
// the same bearer could delete the house's phones or read the Watchtower raw.
// The mark is a context value, so no header from outside can forge it. An
// agent is never a steward, whatever its house is: the steward routes are the
// ones the UI guards with a confirm dialog, and an agent has none.
func (s *Server) requireHouse(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok, fromCookie, refused := s.tokenFrom(r)
		if refused != "" {
			writeErr(w, http.StatusForbidden, refused)
			return
		}
		if tok == "" {
			writeErr(w, http.StatusUnauthorized, "no token")
			return
		}
		row, err := s.st.One(r.Context(), `SELECT d.id AS device_id, d.agent, h.id, h.name, h.is_steward
			FROM devices d JOIN houses h ON h.id=d.house_id WHERE d.token_hash=?`, hashToken(tok))
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		if row == nil {
			// A dead cookie — a phone a steward removed, the demo's hourly
			// reset — would ride on every call for a year; take it off now.
			if fromCookie {
				s.clearSession(w)
			}
			writeErr(w, http.StatusUnauthorized, "unknown device")
			return
		}
		agent := row["agent"].(int64) == 1
		if agent && !(r.Method == "POST" && r.URL.Path == "/api/mcp") && r.Context().Value(viaMCP{}) == nil {
			writeErr(w, http.StatusForbidden, "this key speaks MCP only")
			return
		}
		h := &House{ID: row["id"].(int64), Name: row["name"].(string), IsSteward: row["is_steward"].(int64) == 1 && !agent, DeviceID: row["device_id"].(int64), Agent: agent}
		go s.st.Exec(context.Background(), `UPDATE devices SET last_seen=datetime('now') WHERE id=?`, h.DeviceID)
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, h)))
	}
}

// The session cookie. A phone keeps its token in localStorage and sends it
// as a bearer, as it always did; the same token also rides in this cookie,
// set by the server, because Safari evicts what page script wrote after
// seven days of Safari use without opening the site, and a cookie the
// server set it keeps. HttpOnly keeps it from page script, SameSite=Strict
// keeps another site from sending it, Secure follows PUBLIC_URL so the dev
// proxy on plain http still gets one. A year, re-issued by GET /api/me on
// every load; nothing renews a cookie but the server sending it again.
// The header wins when both arrive, and /api/mcp reads the header only:
// the cookie rides on every same-origin fetch, and the agent door must stay
// closed to a phone's session (CLAUDE.md § A door for agents).
//
// SameSite fences a *site*, not an origin: the demo and the landing are
// sibling subdomains of one site, a village may sit beside its homestead's
// pages, and a cloud domain would make every village a sibling of every
// other. A sibling page's fetch with a text/plain body sends no preflight,
// the browser attaches the cookie, and readJSON decodes the body whatever
// its type — so a write by cookie is taken only when the request's Origin
// is the portal's own, PUBLIC_URL's. A browser always sends Origin on a
// POST, PUT or DELETE; a request without one is not a browser's and the
// cookie was not ambient. Reads need no such check: a sibling cannot read
// the answer without CORS, and there is none.
const sessionCookie = "potok"

func (s *Server) tokenFrom(r *http.Request) (tok string, fromCookie bool, refused string) {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		if t := strings.TrimPrefix(auth, "Bearer "); t != "" {
			return t, false, ""
		}
	}
	if r.URL.Path == "/api/mcp" {
		return "", false, ""
	}
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return "", false, ""
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if o := r.Header.Get("Origin"); o != "" && !sameOrigin(o, s.cfg.PublicURL) {
			return "", true, "a write by cookie is taken from " + s.cfg.PublicURL + " only, not from " + o
		}
	}
	return c.Value, true, ""
}

func sameOrigin(a, b string) bool {
	ua, err1 := url.Parse(a)
	ub, err2 := url.Parse(b)
	return err1 == nil && err2 == nil && ua.Host != "" && strings.EqualFold(ua.Scheme, ub.Scheme) && strings.EqualFold(ua.Host, ub.Host)
}

func (s *Server) sessionCookieFor(tok string, maxAge int) *http.Cookie {
	return &http.Cookie{Name: sessionCookie, Value: tok, Path: "/", MaxAge: maxAge, HttpOnly: true,
		SameSite: http.SameSiteStrictMode, Secure: strings.HasPrefix(s.cfg.PublicURL, "https://")}
}

func (s *Server) setSession(w http.ResponseWriter, tok string) {
	http.SetCookie(w, s.sessionCookieFor(tok, 365*24*3600))
}

func (s *Server) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, s.sessionCookieFor("", -1))
}

// logout takes the cookie off. No token needed: a browser clearing its own
// cookie harms nobody, and a dead one is exactly the case that calls it.
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	// The one write by cookie outside requireHouse, so the same Origin fence
	// here: a sibling page must not log a phone out of its cookie.
	if o := r.Header.Get("Origin"); o != "" && !sameOrigin(o, s.cfg.PublicURL) {
		writeErr(w, http.StatusForbidden, "a write by cookie is taken from "+s.cfg.PublicURL+" only, not from "+o)
		return
	}
	s.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) requireSteward(next http.HandlerFunc) http.HandlerFunc {
	return s.requireHouse(func(w http.ResponseWriter, r *http.Request) {
		if !houseFrom(r).IsSteward {
			writeErr(w, http.StatusForbidden, "stewards only")
			return
		}
		next(w, r)
	})
}

// newDevice mints a token for a house and stores its hash, with the hash of
// the PIN the device signs in by, when it has one ("" for none).
func (s *Server) newDevice(ctx context.Context, houseID int64, label, pin string) (string, error) {
	tok := randomToken(32)
	_, err := s.st.Exec(ctx, `INSERT INTO devices(house_id, token_hash, label, pin_hash) VALUES (?,?,?,?)`, houseID, hashToken(tok), label, nullIfEmpty(pin))
	return tok, err
}

// An invite is a house's own link, for any number of its people, for seven
// days: each opens it once and chooses a PIN (owner's decision 2026-11-02).
func (s *Server) newInvite(ctx context.Context, houseID int64) (store.Row, error) {
	if _, err := s.st.Exec(ctx, `DELETE FROM invites WHERE house_id=?`, houseID); err != nil {
		return nil, err
	}
	code := inviteCode()
	exp := time.Now().UTC().Add(7 * 24 * time.Hour).Format(time.RFC3339)
	if _, err := s.st.Exec(ctx, `INSERT INTO invites(code, house_id, expires_at) VALUES (?,?,?)`, code, houseID, exp); err != nil {
		return nil, err
	}
	return store.Row{"code": code, "expires_at": exp}, nil
}

// PINs (owner's decisions 2026-11-02; CLAUDE.md § Codes and PINs). Whoever
// opens a house's invite link chooses a PIN on that device. Later, in any
// browser — the home-screen app, a laptop, WhatsApp's own browser — the gate
// lists the houses that have a PIN, the person taps theirs and types it, and
// that browser is a new device of the house carrying the same PIN.
//
// A PIN is six digits, and the house names on the gate say where to guess,
// so three wrong PINs in a row lock the house: no PIN signs it in until
// someone of the house joins by its link again, which clears the count, as
// a right PIN before the lock does. A guesser gets three tries per house per
// link — about one chance in 170,000 for a house with two PINs. The cost,
// paid knowingly: anyone can lock every house in three taps each, and a
// villager who mistypes three times locks their own house; the browsers
// already signed in stay in, and the link is the way back for the rest.
//
// The hash folds in the house id, so the same PIN in two houses is two
// hashes; against a stolen database it protects nothing, and nothing else in
// that database is protected either.
var pinShape = regexp.MustCompile(`^[0-9]{6}$`)

const pinTries = 3

func pinHash(houseID int64, pin string) string {
	return hashToken("pin:" + strconv.FormatInt(houseID, 10) + ":" + pin)
}

// doors lists the houses the gate offers: every house with at least one PIN,
// by name, with the tries it has left before it locks, so the gate says it
// before the first PIN is typed and a locked house says so before any.
// Public, like the gate picture: the owner's decision 2026-11-02, for now
// (docs/later.md). Never a common place, never a house nobody has opened the
// link of — its door would take no PIN.
func (s *Server) doors(w http.ResponseWriter, r *http.Request) {
	rows, err := s.st.Rows(r.Context(), `SELECT h.id, h.name, h.crest, h.color,
		max(0, ? - coalesce((SELECT misses FROM pin_tries t WHERE t.house_id=h.id), 0)) AS left
		FROM houses h
		WHERE h.kind='house' AND EXISTS (SELECT 1 FROM devices d WHERE d.house_id=h.id AND d.pin_hash IS NOT NULL)
		ORDER BY h.name`, pinTries)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, rows)
}

// login signs a browser in by a house and a PIN one of its devices chose.
// The new device takes the matched device's label, which is the best guess
// at who holds it, and its PIN, so the person's PIN outlives the browser it
// was chosen in. A wrong PIN answers with the tries left; a locked house is
// a 423 before the PIN is read, so a right PIN cannot open it either.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.tries.allow(clientIP(r), 12, 10*time.Minute) {
		writeErr(w, http.StatusTooManyRequests, "too many attempts, wait a few minutes")
		return
	}
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	id, _ := m["house_id"].(float64)
	houseID, pin := int64(id), str(m, "pin")
	if !pinShape.MatchString(pin) {
		writeErr(w, 400, "a PIN is 6 digits")
		return
	}
	ctx := r.Context()
	if h, err := s.st.One(ctx, `SELECT kind FROM houses WHERE id=?`, houseID); err != nil {
		fail(w, err)
		return
	} else if h == nil || h["kind"] != "house" {
		writeErr(w, 404, "no such house")
		return
	}
	// One login at a time, so two tries at once cannot both pass the count.
	s.pinMu.Lock()
	defer s.pinMu.Unlock()
	var misses int64
	if t, err := s.st.One(ctx, `SELECT misses FROM pin_tries WHERE house_id=?`, houseID); err != nil {
		fail(w, err)
		return
	} else if t != nil {
		misses = t["misses"].(int64)
	}
	if misses >= pinTries {
		writeJSON(w, http.StatusLocked, map[string]any{"error": "this house is locked: open its link again", "left": 0})
		return
	}
	dev, err := s.st.One(ctx, `SELECT label, pin_hash FROM devices WHERE house_id=? AND agent=0 AND pin_hash=? LIMIT 1`, houseID, pinHash(houseID, pin))
	if err != nil {
		fail(w, err)
		return
	}
	if dev == nil {
		misses++
		if _, err := s.st.Exec(ctx, `INSERT INTO pin_tries(house_id, misses) VALUES (?,?)
			ON CONFLICT(house_id) DO UPDATE SET misses=excluded.misses`, houseID, misses); err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "wrong PIN", "left": pinTries - misses})
		return
	}
	if _, err := s.st.Exec(ctx, `DELETE FROM pin_tries WHERE house_id=?`, houseID); err != nil {
		fail(w, err)
		return
	}
	tok, err := s.newDevice(ctx, houseID, str(dev, "label"), str(dev, "pin_hash"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"token": tok, "house_id": houseID})
}

// putPin sets this device's PIN, or changes it. A change moves every device
// of the house that carried the old PIN — the browsers it signed in — so the
// old PIN stops working everywhere at once. The cost, paid knowingly: two
// people of one house who chose the same PIN share it, and a change by one
// is a change for both.
func (s *Server) putPin(w http.ResponseWriter, r *http.Request) {
	h := houseFrom(r)
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	pin := str(m, "pin")
	if !pinShape.MatchString(pin) {
		writeErr(w, 400, "a PIN is 6 digits")
		return
	}
	old, err := s.st.One(r.Context(), `SELECT pin_hash FROM devices WHERE id=?`, h.DeviceID)
	if err != nil {
		fail(w, err)
		return
	}
	if was := str(old, "pin_hash"); was != "" {
		_, err = s.st.Exec(r.Context(), `UPDATE devices SET pin_hash=? WHERE house_id=? AND pin_hash=?`, pinHash(h.ID, pin), h.ID, was)
	} else {
		_, err = s.st.Exec(r.Context(), `UPDATE devices SET pin_hash=? WHERE id=?`, pinHash(h.ID, pin), h.DeviceID)
	}
	if err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}
