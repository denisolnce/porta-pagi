package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSiteFiles: a village's own backdrop is a row that serves in the built-in
// file's place, public like the gate it stands behind, with an ETag; only an
// image is taken; and the shell carries the village's first language.
func TestSiteFiles(t *testing.T) {
	srv, _, _, _ := newVillage(t)
	if err := srv.ImportSite("backdrop", strings.NewReader("not a picture")); err == nil {
		t.Fatal("text was taken as a backdrop")
	}
	if err := srv.ImportSite("icon", strings.NewReader("\x89PNG\r\n\x1a\n")); err == nil {
		t.Fatal("an unknown name was taken")
	}
	png := "\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 64)
	if err := srv.ImportSite("backdrop", strings.NewReader(png)); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/backdrop.jpg", nil))
	etag := rec.Header().Get("ETag")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" || rec.Body.String() != png || etag == "" {
		t.Fatalf("backdrop: %d %q %d bytes", rec.Code, rec.Header().Get("Content-Type"), rec.Body.Len())
	}
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/backdrop.jpg", nil)
	req.Header.Set("If-None-Match", etag)
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != 304 {
		t.Fatalf("a phone that holds the backdrop fetched it again: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), `<html lang="sl">`) {
		t.Fatalf("the shell without the village's language: %q", rec.Body.String())
	}
}

// TestSiteUpload: a steward replaces the gate picture from the page and goes
// back to the built-in one; a house that is not a steward cannot, and the
// server takes only an image.
func TestSiteUpload(t *testing.T) {
	srv, _, steward, house := newVillage(t)
	put := func(c *client, ct, body string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/api/site/backdrop", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", ct)
		srv.Handler().ServeHTTP(rec, req)
		return rec.Code
	}
	jpeg := "\xff\xd8\xff\xe0" + strings.Repeat("\x00", 64)
	if c := put(house, "image/jpeg", jpeg); c != 403 {
		t.Fatalf("a house that is not a steward replaced the gate picture: %d", c)
	}
	if c := put(steward, "text/plain", "hello"); c != 415 {
		t.Fatalf("text as a picture: %d", c)
	}
	if c := put(steward, "image/jpeg", "not a picture at all"); c != 400 {
		t.Fatalf("bytes that are not an image, labelled jpeg: %d", c)
	}
	if c := put(steward, "image/jpeg", jpeg); c != 204 {
		t.Fatalf("the steward's picture: %d", c)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/backdrop.jpg", nil))
	if rec.Body.String() != jpeg {
		t.Fatalf("the gate does not show the steward's picture: %d bytes", rec.Body.Len())
	}
	if code, _, list := steward.do("GET", "/api/site", nil); code != 200 || len(list) != 1 {
		t.Fatalf("list: %d %v", code, list)
	}
	if code, _, _ := steward.do("DELETE", "/api/site/backdrop", nil); code != 204 {
		t.Fatalf("back to the built-in picture: %d", code)
	}
	if _, _, list := steward.do("GET", "/api/site", nil); len(list) != 0 {
		t.Fatalf("the row stayed: %v", list)
	}
	srv.demoLocked = true
	if c := put(steward, "image/jpeg", jpeg); c != 403 {
		t.Fatalf("the demo took a visitor's picture: %d", c)
	}
}
