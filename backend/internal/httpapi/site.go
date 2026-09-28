package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// A village may replace the pictures the page wears — today the backdrop
// behind the gate, frontend/public/backdrop.jpg — with its own: a steward
// uploads one from the settings page (PUT /api/site/backdrop) and can go back
// to the built-in one (DELETE), or it is entered from a file on the VM:
//
//	docker exec -i <container> /server site-import backdrop < picture.jpg
//
// The row lives in site_files; with no row, the picture built into the image
// serves, so every village starts with the same one. The backdrop is public
// (it is behind the gate, before any login), so it is served with no token,
// as the built-in file is.

var siteNames = map[string]bool{"backdrop": true}

const siteCap = 2 << 20

// ImportSite reads one picture from `in` and upserts its row. The type is
// read off the bytes, not the filename, and only an image is taken.
func (s *Server) ImportSite(name string, in io.Reader) error {
	if !siteNames[name] {
		return fmt.Errorf("name must be backdrop, got %q", name)
	}
	b, err := io.ReadAll(io.LimitReader(in, siteCap+1))
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	return s.storeSite(context.Background(), name, b)
}

// storeSite checks one picture's size and type and upserts its row.
func (s *Server) storeSite(ctx context.Context, name string, b []byte) error {
	if len(b) > siteCap {
		return fmt.Errorf("%s is over %d MB", name, siteCap>>20)
	}
	ct := http.DetectContentType(b)
	if ct != "image/jpeg" && ct != "image/png" && ct != "image/webp" {
		return fmt.Errorf("%s must be a jpeg, png or webp, this is %s", name, ct)
	}
	_, err := s.st.Exec(ctx, `INSERT INTO site_files (name, bytes, content_type, updated_at) VALUES (?,?,?,datetime('now'))
		ON CONFLICT(name) DO UPDATE SET bytes=excluded.bytes, content_type=excluded.content_type, updated_at=excluded.updated_at`, name, b, ct)
	return err
}

// serveSiteFile writes the village's own picture if it has one and reports
// whether it did; the caller falls back to the built-in file otherwise.
func (s *Server) serveSiteFile(w http.ResponseWriter, r *http.Request, name string) bool {
	meta, err := s.st.One(r.Context(), `SELECT content_type, updated_at FROM site_files WHERE name=?`, name)
	if err != nil || meta == nil {
		return false
	}
	etag := fmt.Sprintf(`"%s-%s"`, name, strings.ReplaceAll(fmt.Sprint(meta["updated_at"]), " ", "T"))
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, no-cache")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(304)
		return true
	}
	row, err := s.st.One(r.Context(), `SELECT bytes FROM site_files WHERE name=?`, name)
	if err != nil || row == nil {
		return false
	}
	w.Header().Set("Content-Type", meta["content_type"].(string))
	switch v := row["bytes"].(type) {
	case []byte:
		w.Write(v)
	case string:
		w.Write([]byte(v))
	}
	return true
}

// listSite: which pictures this village has replaced, and when — the
// settings page shows "your own picture since …" or "the built-in one".
func (s *Server) listSite(w http.ResponseWriter, r *http.Request) {
	rows, err := s.st.Rows(r.Context(), `SELECT name, updated_at FROM site_files`)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, rows)
}

// putSite: a steward replaces one picture with the request body (jpeg, png
// or webp, under 2 MB; the page shrinks it first). The demo refuses: one
// visitor's picture would greet every other visitor for the rest of the hour.
func (s *Server) putSite(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !siteNames[name] {
		writeErr(w, 404, "no such picture")
		return
	}
	if s.demoLocked {
		writeErr(w, 403, "the demo keeps its picture")
		return
	}
	body, _, ok := readPhoto(w, r)
	if !ok {
		return
	}
	if err := s.storeSite(r.Context(), name, body); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	w.WriteHeader(204)
}

// deleteSite: back to the picture built into the image.
func (s *Server) deleteSite(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !siteNames[name] {
		writeErr(w, 404, "no such picture")
		return
	}
	if s.demoLocked {
		writeErr(w, 403, "the demo keeps its picture")
		return
	}
	if _, err := s.st.Exec(r.Context(), `DELETE FROM site_files WHERE name=?`, name); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}
