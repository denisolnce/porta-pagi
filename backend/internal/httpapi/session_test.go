package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/denisolnce/porta-pagi/backend/internal/config"
	"github.com/denisolnce/porta-pagi/backend/internal/store"
)

// cookieReq: a request carrying only the session cookie, no header.
func cookieReq(method, path, tok string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	if tok != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tok})
	}
	return req
}

// TestSessionCookie pins the cookie half of a session (auth.go): /api/me
// hands a bearer its cookie, the cookie alone opens every house route and
// never /api/mcp, a dead one is taken off on the 401, and DELETE
// /api/session takes a live one off.
func TestSessionCookie(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := New(st, config.Config{BootstrapCode: "letmein", Languages: []string{"sl", "en"}})
	h := srv.Handler()
	c := &client{t: t, h: h}
	code, obj, _ := c.do("POST", "/api/bootstrap", map[string]any{"code": "letmein", "name": "Stewards"})
	c.must(201, code, "bootstrap")
	tok := obj["token"].(string)

	// The bearer gets its cookie on /api/me, and nowhere else.
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("me: %d %s", rec.Code, rec.Body.String())
	}
	var set *http.Cookie
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == sessionCookie {
			set = ck
		}
	}
	if set == nil || set.Value != tok || !set.HttpOnly || set.SameSite != http.SameSiteStrictMode || set.MaxAge != 365*24*3600 || set.Path != "/" {
		t.Fatalf("me set no proper cookie: %+v", set)
	}
	if set.Secure {
		t.Fatal("Secure on a PUBLIC_URL that is not https — the dev proxy would get none")
	}
	req = httptest.NewRequest("GET", "/api/houses", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("a route other than /api/me set a cookie")
	}

	// The cookie alone is a session.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, cookieReq("GET", "/api/houses", tok))
	if rec.Code != 200 {
		t.Fatalf("cookie on /api/houses: %d %s", rec.Code, rec.Body.String())
	}
	// Except at the agent door, which reads the header only.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, cookieReq("POST", "/api/mcp", tok))
	if rec.Code != 401 || !strings.Contains(rec.Body.String(), "no token") {
		t.Fatalf("cookie at /api/mcp: want 401 no token, got %d %s", rec.Code, rec.Body.String())
	}

	// A dead cookie is taken off with the 401.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, cookieReq("GET", "/api/houses", "potok_dead"))
	if rec.Code != 401 {
		t.Fatalf("dead cookie: %d", rec.Code)
	}
	if cks := rec.Result().Cookies(); len(cks) != 1 || cks[0].Name != sessionCookie || cks[0].MaxAge >= 0 {
		t.Fatalf("dead cookie not cleared: %+v", cks)
	}

	// Logout takes a live one off, and needs no token to do it.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, cookieReq("DELETE", "/api/session", ""))
	if rec.Code != 204 {
		t.Fatalf("logout: %d", rec.Code)
	}
	if cks := rec.Result().Cookies(); len(cks) != 1 || cks[0].MaxAge >= 0 {
		t.Fatalf("logout cleared nothing: %+v", cks)
	}

	// Under https the cookie is Secure.
	srv2 := New(st, config.Config{BootstrapCode: "letmein", Languages: []string{"sl", "en"}, PublicURL: "https://vas.example.si"})
	if !srv2.sessionCookieFor("x", 1).Secure {
		t.Fatal("not Secure under an https PUBLIC_URL")
	}
	h2 := srv2.Handler()

	// A write by cookie is taken from the portal's own origin only: a sibling
	// subdomain is same-site for SameSite and would get the cookie attached.
	post := func(origin, cookie, bearer string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/posts", strings.NewReader(`{"body":"hi"}`))
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		rec := httptest.NewRecorder()
		h2.ServeHTTP(rec, req)
		return rec
	}
	if rec := post("https://demo.example.si", tok, ""); rec.Code != 403 || !strings.Contains(rec.Body.String(), "write by cookie") {
		t.Fatalf("sibling origin write by cookie: want 403, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := post("https://vas.example.si", tok, ""); rec.Code != 201 {
		t.Fatalf("own origin write by cookie: want 201, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := post("", tok, ""); rec.Code != 201 {
		t.Fatalf("no Origin (not a browser) write by cookie: want 201, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := post("https://demo.example.si", "", tok); rec.Code != 201 {
		t.Fatalf("bearer from anywhere: want 201, got %d %s", rec.Code, rec.Body.String())
	}
	// Logout is the one write by cookie outside requireHouse: same fence.
	req = httptest.NewRequest("DELETE", "/api/session", nil)
	req.Header.Set("Origin", "https://demo.example.si")
	rec = httptest.NewRecorder()
	h2.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("sibling origin logout: want 403, got %d", rec.Code)
	}

	// The header wins over the cookie, and /api/me then re-issues the
	// header's token: the demo switcher writes the next house to the drawer
	// and reloads while the cookie still holds the last one.
	c.token = tok
	code, obj, _ = c.do("POST", "/api/houses", map[string]any{"name": "Pisani Petelin", "crest": "🌳", "color": "#4a7c3f"})
	c.must(201, code, "create house")
	inv := obj["invite"].(map[string]any)["code"].(string)
	code, obj, _ = c.do("POST", "/api/join", map[string]any{"code": inv, "device": "phone"})
	c.must(201, code, "join")
	tokB := obj["token"].(string)
	req = httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokB)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tok})
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Pisani Petelin") {
		t.Fatalf("header did not win: %d %s", rec.Code, rec.Body.String())
	}
	set = nil
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == sessionCookie {
			set = ck
		}
	}
	if set == nil || set.Value != tokB {
		t.Fatalf("me re-issued the wrong token: %+v", set)
	}
}
