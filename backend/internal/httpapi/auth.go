package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
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

// pairingCode is six digits: short enough to read across a kitchen table,
// short-lived and single-use to make up for the small keyspace.
func pairingCode() string {
	b := make([]byte, 6)
	rand.Read(b)
	out := make([]byte, 6)
	for i := range b {
		out[i] = '0' + b[i]%10
	}
	return string(out)
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

// newDevice mints a token for a house and stores its hash.
func (s *Server) newDevice(ctx context.Context, houseID int64, label string) (string, error) {
	tok := randomToken(32)
	_, err := s.st.Exec(ctx, `INSERT INTO devices(house_id, token_hash, label) VALUES (?,?,?)`, houseID, hashToken(tok), label)
	return tok, err
}

func (s *Server) newInvite(ctx context.Context, houseID int64) (store.Row, error) {
	if _, err := s.st.Exec(ctx, `DELETE FROM invites WHERE house_id=?`, houseID); err != nil {
		return nil, err
	}
	code := inviteCode()
	exp := time.Now().UTC().Add(14 * 24 * time.Hour).Format(time.RFC3339)
	if _, err := s.st.Exec(ctx, `INSERT INTO invites(code, house_id, expires_at) VALUES (?,?,?)`, code, houseID, exp); err != nil {
		return nil, err
	}
	return store.Row{"code": code, "expires_at": exp}, nil
}
