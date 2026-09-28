package httpapi

import (
	"context"
	"net/http"
)

// Feature is one switch: a room or a behaviour a village turns on or off.
// The defaults are the village that built the portal, and most are off
// (docs/commercial.md, the Decided block). A switched-off feature is absent,
// not disabled: its routes answer 404, its MCP tools are not listed, its
// push kind is not offered, and the page does not draw its door.
type Feature struct {
	Key     string
	Default bool
	// Kind is the push kind the feature owns, "" for none.
	Kind string
}

// Features in the order the steward's settings page lists them. A new
// switch is one line here, one guard on its routes, one line in the page.
var Features = []Feature{
	{Key: "camp", Default: false, Kind: "camp"},
	{Key: "dues", Default: false},
	{Key: "roster", Default: false},
}

func featureByKey(key string) (Feature, bool) {
	for _, f := range Features {
		if f.Key == key {
			return f, true
		}
	}
	return Feature{}, false
}

// feature reports whether a switch is on in this village: the flipped row
// when one exists, else the default. An unknown key is off.
func (s *Server) feature(ctx context.Context, key string) bool {
	f, ok := featureByKey(key)
	if !ok {
		return false
	}
	row, err := s.st.One(ctx, `SELECT enabled FROM features WHERE key=?`, key)
	if err != nil || row == nil {
		return f.Default
	}
	return row["enabled"].(int64) == 1
}

// requireFeature wraps a handler so it exists only while its switch is on.
// Off answers 404, the same as a route that was never built.
func (s *Server) requireFeature(key string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.feature(r.Context(), key) {
			writeErr(w, http.StatusNotFound, "not in this village")
			return
		}
		next(w, r)
	}
}

// offKinds lists the push kinds whose feature is switched off: nobody is
// offered them, and nothing of theirs rings.
func (s *Server) offKinds(ctx context.Context) map[string]bool {
	off := map[string]bool{}
	for _, f := range Features {
		if f.Kind != "" && !s.feature(ctx, f.Key) {
			off[f.Kind] = true
		}
	}
	return off
}

// kinds is the push kinds this village offers: Kinds without the ones whose
// feature is off. The prefs page lists these and no other, and a prefs write
// touches only these, so a house's or a steward's choice about a switched-off
// kind waits intact for the day it comes back.
func (s *Server) kinds(ctx context.Context) []string {
	off := s.offKinds(ctx)
	out := []string{}
	for _, k := range Kinds {
		if !off[k] {
			out = append(out, k)
		}
	}
	return out
}

// listFeatures: every switch with its state, for any house — the page draws
// its doors from it. Who flipped a switch, and when, rides along.
func (s *Server) listFeatures(w http.ResponseWriter, r *http.Request) {
	rows, err := s.st.Rows(r.Context(), `SELECT f.key, f.enabled, f.set_at, h.name AS set_by FROM features f LEFT JOIN houses h ON h.id = f.set_by`)
	if err != nil {
		fail(w, err)
		return
	}
	set := map[string]map[string]any{}
	for _, row := range rows {
		set[row["key"].(string)] = row
	}
	out := []map[string]any{}
	for _, f := range Features {
		on := f.Default
		item := map[string]any{"key": f.Key, "default": f.Default}
		if row, ok := set[f.Key]; ok {
			on = row["enabled"].(int64) == 1
			item["set_by"] = row["set_by"]
			item["set_at"] = row["set_at"]
		}
		item["on"] = on
		out = append(out, item)
	}
	writeJSON(w, 200, out)
}

// putFeature: a steward flips one switch. {"on": true|false}.
func (s *Server) putFeature(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if _, ok := featureByKey(key); !ok {
		writeErr(w, 404, "no such switch")
		return
	}
	m, err := readJSON(r)
	if err != nil {
		writeErr(w, 400, "bad json")
		return
	}
	on, ok := m["on"].(bool)
	if !ok {
		writeErr(w, 400, "on must be true or false")
		return
	}
	v := 0
	if on {
		v = 1
	}
	if _, err := s.st.Exec(r.Context(), `INSERT INTO features(key, enabled, set_by, set_at) VALUES (?,?,?,datetime('now'))
		ON CONFLICT(key) DO UPDATE SET enabled=excluded.enabled, set_by=excluded.set_by, set_at=excluded.set_at`, key, v, houseFrom(r).ID); err != nil {
		fail(w, err)
		return
	}
	s.listFeatures(w, r)
}
