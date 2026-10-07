package astro

import (
	"math"
	"testing"
	"time"
	_ "time/tzdata"
)

const (
	kocevjeLat, kocevjeLon   = 45.6458, 14.8496
	alentejoLat, alentejoLon = 37.65, -8.22
)

func utc(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04", s)
	if err != nil {
		panic(err)
	}
	return t
}

func within(a, b time.Time, d time.Duration) bool {
	diff := a.Sub(b)
	return diff <= d && -diff <= d
}

// Meeus example 47.a; tolerance 1e-4° and 1 km.
func TestMoonExample47a(t *testing.T) {
	jde := julian(time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC))
	if jde != 2448724.5 {
		t.Fatalf("JD = %v, want 2448724.5", jde)
	}
	l, b, d := moonPosition(jde)
	if math.Abs(l-133.162655) > 1e-4 || math.Abs(b+3.229126) > 1e-4 || math.Abs(d-368409.7) > 1 {
		t.Errorf("λ=%.6f β=%.6f Δ=%.1f, want 133.162655 −3.229126 368409.7", l, b, d)
	}
}

// Meeus example 49.a; tolerance 1e-4 d.
func TestPhaseExample49a(t *testing.T) {
	if got := newLunation(-283).phase(); math.Abs(got-2443192.65118) > 1e-4 {
		t.Errorf("JDE = %.5f, want 2443192.65118", got)
	}
}

// Meeus example 27.a; tolerance 1e-4 d. 2437837.38589 is the example's JDE0;
// with S = +635 the instant is 2437837.39245, 21h25m08s TD.
func TestSeasonExample27a(t *testing.T) {
	if got := seasonMean(1962, 1); math.Abs(got-2437837.38589) > 1e-4 {
		t.Errorf("JDE0 = %.5f, want 2437837.38589", got)
	}
	if got := seasonJDE(1962, 1); math.Abs(got-2437837.39245) > 1e-4 {
		t.Errorf("JDE = %.5f, want 2437837.39245", got)
	}
}

// Meeus example 25.a; tolerance 0.01°.
func TestSunExample25a(t *testing.T) {
	jde := 2448908.5
	T := (jde - j2000) / 36525
	_, deps := nutation(T)
	l, _ := sunApparent(jde)
	a, d := equatorial(l, 0, meanObliquity(T)+deps)
	if math.Abs(a-198.38083) > 0.01 || math.Abs(d+7.78507) > 0.01 {
		t.Errorf("α=%.5f δ=%.5f, want 198.38083 −7.78507", a, d)
	}
}

// Published 2026 almanac times; tolerance 3 minutes.
func TestAlmanac2026(t *testing.T) {
	want := []string{"2026-03-20 14:46", "2026-06-21 08:24", "2026-09-23 00:05", "2026-12-21 20:50"}
	for i, s := range Seasons(2026) {
		if !within(s.At, utc(want[i]), 3*time.Minute) {
			t.Errorf("%s at %s, want %s", s.Kind, s.At, want[i])
		}
	}
	phases := MoonPhases(2026)
	for _, w := range []struct {
		full bool
		at   string
	}{
		{true, "2026-01-03 10:03"}, {true, "2026-08-28 04:18"},
		{false, "2026-02-17 12:01"}, {false, "2026-08-12 17:37"},
	} {
		found := false
		for _, p := range phases {
			if p.Full == w.full && within(p.At, utc(w.at), 24*time.Hour) {
				found = true
				if !within(p.At, utc(w.at), 3*time.Minute) {
					t.Errorf("full=%v at %s, want %s", p.Full, p.At, w.at)
				}
			}
		}
		if !found {
			t.Errorf("no phase full=%v near %s", w.full, w.at)
		}
	}
	for i := 1; i < len(phases); i++ {
		if !phases[i].At.After(phases[i-1].At) || phases[i].Full == phases[i-1].Full {
			t.Errorf("phases out of order at %d: %v %v", i, phases[i-1], phases[i])
		}
	}
}

func eclipseOn(es []Eclipse, solar bool, day string) *Eclipse {
	for i, e := range es {
		if e.Solar == solar && e.Max.Format("2006-01-02") == day {
			return &es[i]
		}
	}
	return nil
}

// Eclipses seen from Kočevje; tolerances named per check.
func TestEclipsesKocevje(t *testing.T) {
	es := Eclipses(2026, kocevjeLat, kocevjeLon)
	if e := eclipseOn(es, true, "2026-08-12"); e == nil {
		t.Error("solar eclipse of 2026-08-12 not listed")
	} else if e.Kind != "partial" || e.Magnitude <= 0.5 || e.Magnitude >= 1 {
		t.Errorf("2026-08-12: %s magnitude %.3f, want partial, 0.5–1", e.Kind, e.Magnitude)
	} else if !e.SeenMax.Equal(e.VisibleTo) || e.SeenMagnitude >= e.Magnitude || e.SeenMagnitude <= 0.5 {
		// The sun sets in Slovenia before the greatest eclipse: what the
		// village sees is the eclipse at sunset, smaller than the maximum.
		t.Errorf("2026-08-12 seen: max %s (sunset %s), magnitude %.3f of %.3f", e.SeenMax, e.VisibleTo, e.SeenMagnitude, e.Magnitude)
	}
	if e := eclipseOn(es, false, "2026-08-28"); e == nil {
		t.Error("lunar eclipse of 2026-08-28 not listed")
	} else {
		if e.Kind != "partial" || math.Abs(e.Magnitude-0.93) > 0.02 {
			t.Errorf("2026-08-28: %s magnitude %.3f, want partial 0.93 ± 0.02", e.Kind, e.Magnitude)
		}
		if !within(e.Max, utc("2026-08-28 04:13"), 5*time.Minute) {
			t.Errorf("2026-08-28: max %s, want 04:13 ± 5 min", e.Max)
		}
	}
	if e := eclipseOn(es, false, "2026-03-03"); e != nil {
		t.Errorf("lunar eclipse of 2026-03-03 listed, but the moon is down: %+v", *e)
	}
	if e := eclipseOn(es, true, "2026-02-17"); e != nil {
		t.Errorf("solar eclipse of 2026-02-17 listed, but it is not seen here: %+v", *e)
	}
	es = Eclipses(2025, kocevjeLat, kocevjeLon)
	if e := eclipseOn(es, false, "2025-09-07"); e == nil {
		t.Error("lunar eclipse of 2025-09-07 not listed")
	} else if e.Kind != "total" {
		t.Errorf("2025-09-07: %s, want total", e.Kind)
	}
}

// Kočevje on the June solstice; windows of 15 minutes.
func TestSunRiseSet(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Ljubljana")
	if err != nil {
		t.Fatal(err)
	}
	rise, set, ok := SunRiseSet(time.Date(2026, 6, 21, 12, 0, 0, 0, loc), loc, kocevjeLat, kocevjeLon)
	if !ok {
		t.Fatal("no sunrise or sunset")
	}
	r, s := rise.In(loc).Format("15:04"), set.In(loc).Format("15:04")
	if r < "05:05" || r > "05:20" || s < "20:50" || s > "21:05" {
		t.Errorf("rise %s set %s, want 05:05–05:20 and 20:50–21:05", r, s)
	}
}

func TestPrint2026(t *testing.T) {
	for _, s := range Seasons(2026) {
		t.Logf("%-18s %s", s.Kind, s.At.Format(time.DateTime))
	}
	for _, p := range MoonPhases(2026) {
		kind := "new"
		if p.Full {
			kind = "full"
		}
		t.Logf("%-4s %s", kind, p.At.Format(time.DateTime))
	}
	hm := func(t time.Time) string {
		if t.IsZero() {
			return "--:--:--"
		}
		return t.Format(time.TimeOnly)
	}
	for _, pl := range []struct {
		name     string
		lat, lon float64
	}{{"Kočevje", kocevjeLat, kocevjeLon}, {"Alentejo", alentejoLat, alentejoLon}} {
		t.Logf("eclipses 2026 from %s (%.4f, %.4f), UTC:", pl.name, pl.lat, pl.lon)
		for _, e := range Eclipses(2026, pl.lat, pl.lon) {
			body := "lunar"
			if e.Solar {
				body = "solar"
			}
			t.Logf("  %s %s %-7s mag %.3f  begin %s  max %s  end %s  total %s–%s  visible %s–%s",
				e.Max.Format(time.DateOnly), body, e.Kind, e.Magnitude,
				hm(e.Begin), hm(e.Max), hm(e.End), hm(e.TotalBegin), hm(e.TotalEnd), hm(e.VisibleFrom), hm(e.VisibleTo))
		}
	}
}

// Meeus ch.8's own examples, and 2019 and 2038, where the first full moon in
// the sky after the equinox is not Easter's moon.
func TestEaster(t *testing.T) {
	for year, want := range map[int]string{1818: "03-22", 1954: "04-18", 1991: "03-31", 1992: "04-19", 1993: "04-11", 2000: "04-23", 2019: "04-21", 2026: "04-05", 2038: "04-25"} {
		if got := Easter(year).Format("01-02"); got != want {
			t.Errorf("Easter %d: %s, want %s", year, got, want)
		}
	}
}
