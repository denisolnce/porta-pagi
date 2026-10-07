package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/denisolnce/porta-pagi/backend/internal/astro"
	"github.com/denisolnce/porta-pagi/backend/internal/store"
)

// The almanac: the sky over the village in its calendar — new and full
// moons, the equinoxes and solstices, and the eclipses the village can see.
// The backend writes a year once, and the next one with it so December pages
// into January; nobody creates, edits or answers a row, a house may only talk
// about one. Every time is the village's wall clock, the same shape an event
// has. A moon phase or an equinox is one instant for the whole planet and
// needs no place; an eclipse and the length of a day do. The place is the one
// the weather already names — WEATHER_COORDS, or the point ARSO gives for its
// place name — and without one the almanac leaves eclipses out rather than
// list one the village may never see.

// almanacTitles are the English keys the push of a comment words; the page
// words a row from its kind itself.
var almanacTitles = map[string]string{
	"new_moon": "New moon", "full_moon": "Full moon",
	"march_equinox": "March equinox", "june_solstice": "June solstice",
	"september_equinox": "September equinox", "december_solstice": "December solstice",
	"solar_eclipse": "Solar eclipse", "lunar_eclipse": "Lunar eclipse",
}

// RunCalendar keeps the year's sky written and rings the morning's birthdays:
// once at start, then every 30 minutes.
func (s *Server) RunCalendar(ctx context.Context) {
	t := time.NewTicker(30 * time.Minute)
	defer t.Stop()
	for {
		y := s.now().Year()
		for _, year := range []int{y, y + 1} {
			if err := s.fillAlmanac(ctx, year); err != nil {
				log.Printf("almanac %d: %v", year, err)
			}
		}
		if n, err := s.ringBirthdays(ctx); err != nil {
			log.Printf("birthdays: %v", err)
		} else if n > 0 {
			log.Printf("birthdays: %d rung", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

type almanacRow struct {
	kind, starts, ends string
	detail             map[string]any
}

// fillAlmanac writes one year's rows, once per place: settings holds
// almanac:<year> with the setting the place came from, and a village that
// names a place later — or another one — gets the year again, its eclipses
// and day lengths with it. A row keeps its id, so its thread stays; an
// eclipse the new place does not see goes, unless somebody talked about it.
// A place the village names but the provider does not answer for is an
// error, and the next round tries again: the year is not written without it.
func (s *Server) fillAlmanac(ctx context.Context, year int) error {
	key, where := fmt.Sprintf("almanac:%d", year), s.placeKey()
	if row, err := s.st.One(ctx, `SELECT value FROM settings WHERE key=?`, key); err != nil || (row != nil && row["value"] == where) {
		return err
	}
	lat, lon, placed, err := s.skyPlace(ctx)
	if err != nil {
		return err
	}
	return writeAlmanac(ctx, s.st, year, lat, lon, placed, key, where)
}

// writeAlmanac is the write itself, inside its own transaction or the
// caller's (the demo seeds inside one).
func writeAlmanac(ctx context.Context, st *store.Store, year int, lat, lon float64, placed bool, key, where string) error {
	write := func(tx *store.Store) error {
		var eclipses []any
		for _, r := range almanacYear(year, lat, lon, placed) {
			d, _ := json.Marshal(r.detail)
			if _, err := tx.Exec(ctx, `INSERT INTO almanac(kind, starts_at, ends_at, detail) VALUES (?,?,?,?)
				ON CONFLICT(kind, starts_at) DO UPDATE SET ends_at=excluded.ends_at, detail=excluded.detail`, r.kind, r.starts, nullIfEmpty(r.ends), string(d)); err != nil {
				return err
			}
			if strings.HasSuffix(r.kind, "eclipse") {
				eclipses = append(eclipses, r.starts)
			}
		}
		marks := strings.TrimSuffix(strings.Repeat("?,", len(eclipses)), ",")
		if marks == "" {
			marks = "''"
		}
		if _, err := tx.Exec(ctx, `DELETE FROM almanac WHERE kind LIKE '%eclipse' AND starts_at LIKE ? AND starts_at NOT IN (`+marks+`)
			AND NOT EXISTS (SELECT 1 FROM comments c WHERE c.subject='almanac' AND c.subject_id=almanac.id)`, append([]any{fmt.Sprintf("%d-%%", year)}, eclipses...)...); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO settings(key, value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, where)
		return err
	}
	if err := st.Tx(ctx, write); !errors.Is(err, store.ErrNestedTx) {
		return err
	}
	return write(st)
}

// placeKey names where the sky is computed for, by the setting it comes
// from: comparing it costs no call to ARSO.
func (s *Server) placeKey() string {
	if s.cfg.WeatherProvider == "arso" {
		return "arso:" + s.cfg.WeatherLocation
	}
	return "open-meteo:" + strings.ReplaceAll(s.cfg.WeatherCoords, " ", "")
}

// almanacYear computes the rows whose wall-clock day falls in year. A phase
// late on 31 December in UTC is already January here, so the neighbouring
// years are asked too and kept only when their local date is this year's.
func almanacYear(year int, lat, lon float64, placed bool) []almanacRow {
	wall := func(t time.Time) string { return t.In(time.Local).Round(time.Minute).Format("2006-01-02T15:04") }
	clock := func(t time.Time) string { return t.In(time.Local).Round(time.Minute).Format("15:04") }
	ours := func(t time.Time) bool { return t.In(time.Local).Year() == year }
	// The seasons' words are the north's unless the village is south of the
	// equator, where March brings autumn.
	base := func() map[string]any {
		if placed && lat < 0 {
			return map[string]any{"south": true}
		}
		return map[string]any{}
	}
	var out []almanacRow
	for _, y := range []int{year - 1, year, year + 1} {
		for _, p := range astro.MoonPhases(y) {
			if !ours(p.At) {
				continue
			}
			kind := "new_moon"
			if p.Full {
				kind = "full_moon"
			}
			out = append(out, almanacRow{kind, wall(p.At), "", base()})
		}
	}
	// Easter rides on the full moon before it, as the date itself: the
	// church reckons with a table moon, not the sky's, so the page states the
	// day rather than a rule that fails some years.
	easter := astro.Easter(year).Format("2006-01-02")
	best := -1
	for i, r := range out {
		if r.kind == "full_moon" && r.starts[:10] <= easter && r.starts[:10] >= astro.Easter(year).AddDate(0, 0, -8).Format("2006-01-02") {
			best = i
		}
	}
	if best >= 0 {
		out[best].detail["easter"] = easter
	}
	for _, se := range astro.Seasons(year) {
		d := base()
		if placed {
			if rise, set, ok := astro.SunRiseSet(se.At, time.Local, lat, lon); ok {
				d["rise"], d["set"] = clock(rise), clock(set)
			}
		}
		out = append(out, almanacRow{se.Kind, wall(se.At), "", d})
	}
	if !placed {
		return out
	}
	for _, e := range astro.Eclipses(year, lat, lon) {
		d := base()
		d["kind"] = e.Kind
		// What the village sees, not what the planet does: in Kočevje the
		// sun of 12 August 2026 sets ten minutes before the greatest eclipse.
		d["magnitude"] = math.Round(e.SeenMagnitude*100) / 100
		d["max"] = wall(e.SeenMax)
		d["seen_from"], d["seen_to"] = wall(e.VisibleFrom), wall(e.VisibleTo)
		if !e.TotalBegin.IsZero() {
			d["total_from"], d["total_to"] = wall(e.TotalBegin), wall(e.TotalEnd)
		}
		kind := "lunar_eclipse"
		if e.Solar {
			kind = "solar_eclipse"
		}
		out = append(out, almanacRow{kind, wall(e.Begin), wall(e.End), d})
	}
	return out
}

// skyPlace is where the village is, read off the weather's own setting.
// placed is false when the village names no place; err is a place named and
// not reached.
func (s *Server) skyPlace(ctx context.Context) (lat, lon float64, placed bool, err error) {
	if s.cfg.WeatherProvider == "arso" {
		if s.cfg.WeatherLocation == "" {
			return 0, 0, false, nil
		}
		lat, lon, err = arsoPlace(ctx, s.cfg.WeatherLocation, s.cfg.PublicURL)
		return lat, lon, err == nil, err
	}
	a, b, ok := strings.Cut(s.cfg.WeatherCoords, ",")
	if !ok {
		return 0, 0, false, nil
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(a), 64)
	lon, err2 := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false, nil
	}
	return lat, lon, true, nil
}

// arsoPlace asks ARSO where its place name is: the forecast carries the
// point, as [lon, lat]. A town some kilometres off moves an eclipse by
// seconds, which the almanac does not print.
func arsoPlace(ctx context.Context, loc, contact string) (float64, float64, error) {
	var doc struct {
		Forecast24h struct {
			Features []struct {
				Geometry struct {
					Coordinates []float64 `json:"coordinates"`
				} `json:"geometry"`
			} `json:"features"`
		} `json:"forecast24h"`
	}
	if err := getJSON(ctx, "https://vreme.arso.gov.si/api/1.0/location/?location="+url.QueryEscape(loc), contact, &doc); err != nil {
		return 0, 0, err
	}
	if f := doc.Forecast24h.Features; len(f) > 0 && len(f[0].Geometry.Coordinates) == 2 {
		return f[0].Geometry.Coordinates[1], f[0].Geometry.Coordinates[0], nil
	}
	return 0, 0, weatherErr("arso: no point for " + loc)
}

// listAlmanac: the sky from 60 days back, as the calendar lists events, with
// the count of comments on each row.
func (s *Server) listAlmanac(w http.ResponseWriter, r *http.Request) {
	since := s.now().AddDate(0, 0, -60).Format("2006-01-02")
	rows, err := s.st.Rows(r.Context(), `SELECT a.*, (SELECT count(*) FROM comments c WHERE c.subject='almanac' AND c.subject_id=a.id) AS comments
		FROM almanac a WHERE a.starts_at >= ? ORDER BY a.starts_at`, since)
	if err != nil {
		fail(w, err)
		return
	}
	// The title is the English key the page translates, from the one map the
	// comment's push words too.
	for _, row := range rows {
		row["title"] = almanacTitles[str(row, "kind")]
	}
	writeJSON(w, 200, rows)
}
