package astro

import (
	"math"
	"time"
)

// Eclipse is one solar or lunar eclipse as seen from one place on Earth.
type Eclipse struct {
	Solar                  bool
	Kind                   string    // solar, as seen from the place: "partial", "annular", "total"; lunar: "partial", "total" (umbral phases only)
	Begin                  time.Time // solar: local first contact; lunar: start of the umbral (partial) phase
	Max                    time.Time // greatest eclipse (solar: local greatest; lunar: global greatest)
	End                    time.Time // solar: local last contact; lunar: end of the umbral phase
	TotalBegin, TotalEnd   time.Time // the total (or annular) phase, zero when the place has none
	Magnitude              float64   // solar: fraction of the sun's diameter covered at Max as seen from the place; lunar: umbral magnitude
	VisibleFrom, VisibleTo time.Time // the part of Begin..End with the body's centre above the horizon (altitude > 0°, geometric, no refraction)
	// SeenMax and SeenMagnitude are the greatest eclipse the place can see:
	// Max itself when the body is up then, else the edge of the visible part
	// nearest to it — the eclipse grows towards Max and shrinks after it.
	SeenMax       time.Time
	SeenMagnitude float64
}

// seen clamps the true maximum into the visible part.
func seen(mx, from, to float64) float64 { return min(max(mx, from), to) }

// Eclipses returns the eclipses of that UTC year that can be seen from lat,lon (degrees, north and east positive):
// some part of Begin..End with the sun (solar) or moon (lunar) above the horizon. Penumbral-only lunar eclipses are left out:
// nothing shows to the eye. Eclipses not seen from the place are left out.
func Eclipses(year int, lat, lon float64) []Eclipse {
	p := newPlace(lat, lon)
	a, b := yearSpan(year)
	var out []Eclipse
	for _, l := range lunations(year) {
		c, ok := l.eclipse()
		if !ok {
			continue
		}
		var e Eclipse
		if l.full() {
			e, ok = p.lunar(c)
		} else {
			e, ok = p.solar(c)
		}
		if mx := julian(e.Max); ok && mx >= a && mx < b {
			out = append(out, e)
		}
	}
	return out
}

// candidate is the ch.54 answer for one new or full moon.
type candidate struct{ jde, gamma, u, mp float64 }

// eclipse follows ch.54.
func (l lunation) eclipse() (candidate, bool) {
	if math.Abs(sind(l.f)) > 0.36 {
		return candidate{}, false
	}
	m, mp, e, om := l.m, l.mp, l.e, l.om
	f1 := l.f - 0.02665*sind(om)
	a1 := 299.77 + 0.107408*l.k - 0.009173*l.T*l.T
	jde := l.jde
	if l.full() {
		jde += -0.4065*sind(mp) + 0.1727*e*sind(m)
	} else {
		jde += -0.4075*sind(mp) + 0.1721*e*sind(m)
	}
	jde += 0.0161*sind(2*mp) - 0.0097*sind(2*f1) + 0.0073*e*sind(mp-m) - 0.0050*e*sind(mp+m) -
		0.0023*sind(mp-2*f1) + 0.0021*e*sind(2*m) + 0.0012*sind(mp+2*f1) + 0.0006*e*sind(2*mp+m) -
		0.0004*sind(3*mp) - 0.0003*e*sind(m+2*f1) + 0.0003*sind(a1) - 0.0002*e*sind(m-2*f1) -
		0.0002*e*sind(2*mp-m) - 0.0002*sind(om)
	p := 0.2070*e*sind(m) + 0.0024*e*sind(2*m) - 0.0392*sind(mp) + 0.0116*sind(2*mp) -
		0.0073*e*sind(mp+m) + 0.0067*e*sind(mp-m) + 0.0118*sind(2*f1)
	q := 5.2207 - 0.0048*e*cosd(m) + 0.0020*e*cosd(2*m) - 0.3299*cosd(mp) -
		0.0060*e*cosd(mp+m) + 0.0041*e*cosd(mp-m)
	gamma := (p*cosd(f1) + q*sind(f1)) * (1 - 0.0048*math.Abs(cosd(f1)))
	u := 0.0059 + 0.0046*e*cosd(m) - 0.0182*cosd(mp) + 0.0004*cosd(2*mp) - 0.0005*cosd(m+mp)
	return candidate{jde, gamma, u, mp}, true
}

// lunar takes phases and magnitude from ch.54 and the moon's altitude at the place.
func (p place) lunar(c candidate) (Eclipse, bool) {
	mag := (1.0128 - c.u - math.Abs(c.gamma)) / 0.5450
	if mag <= 0 {
		return Eclipse{}, false
	}
	n := 0.5458 + 0.0400*cosd(c.mp)
	semi := func(r float64) float64 { return 60 / n * math.Sqrt(r*r-c.gamma*c.gamma) / 1440 }
	mx := tt2ut(c.jde)
	half := semi(1.0128 - c.u)
	begin, end := mx-half, mx+half
	e := Eclipse{Kind: "partial", Begin: fromJulian(begin), Max: fromJulian(mx), End: fromJulian(end), Magnitude: mag}
	if t := 0.4678 - c.u; math.Abs(c.gamma) < t {
		e.Kind = "total"
		h := semi(t)
		e.TotalBegin, e.TotalEnd = fromJulian(mx-h), fromJulian(mx+h)
	}
	alt := func(jd float64) float64 { f := p.at(jd); return f.altitude(f.moon()) }
	from, to, ok := above(alt, begin, end)
	if !ok {
		return Eclipse{}, false
	}
	e.VisibleFrom, e.VisibleTo = fromJulian(from), fromJulian(to)
	// The moon's centre is γ from the shadow's axis at Max and moves n Earth
	// radii an hour across it, which is what the semidurations already say.
	sm := seen(mx, from, to)
	dist := math.Hypot(c.gamma, n*(sm-mx)*24)
	e.SeenMax, e.SeenMagnitude = fromJulian(sm), (1.0128-c.u-dist)/0.5450
	return e, true
}

// moonK is the moon's radius in Earth radii, ch.55.
const moonK = 0.272481

// solar follows the eclipse at the place itself, from topocentric sun and moon,
// over ±4 h around the ch.54 maximum.
func (p place) solar(c candidate) (Eclipse, bool) {
	if math.Abs(c.gamma) > 1.5433+c.u {
		return Eclipse{}, false
	}
	type disc struct{ sep, sun, moon float64 } // degrees
	at := func(jd float64) disc {
		f := p.at(jd)
		s, m := f.sun(), f.moon()
		return disc{angle(s, m), 959.63 / 3600 / (s.norm() * earthKm / auKm), math.Asin(moonK/m.norm()) / rad}
	}
	sep := func(jd float64) float64 { return at(jd).sep }
	contact := func(jd float64) float64 { d := at(jd); return d.sep - d.sun - d.moon }

	const step = 1.0 / 1440
	mid := tt2ut(c.jde)
	lo, hi := mid-240*step, mid+240*step
	tm, best := mid, math.Inf(1)
	for i := -240; i <= 240; i++ {
		t := mid + float64(i)*step
		if s := sep(t); s < best {
			tm, best = t, s
		}
	}
	tm = minimize(sep, max(tm-step, lo), min(tm+step, hi))
	if contact(tm) >= 0 {
		return Eclipse{}, false
	}
	begin, end := edge(contact, tm, -step, lo), edge(contact, tm, step, hi)
	d := at(tm)
	e := Eclipse{
		Solar: true, Kind: "partial",
		Begin: fromJulian(begin), Max: fromJulian(tm), End: fromJulian(end),
		Magnitude: (d.sun + d.moon - d.sep) / (2 * d.sun),
	}
	switch {
	case d.moon >= d.sun && d.sep <= d.moon-d.sun:
		e.Kind = "total"
	case d.moon < d.sun && d.sep <= d.sun-d.moon:
		e.Kind = "annular"
	}
	if e.Kind != "partial" {
		inner := func(jd float64) float64 { d := at(jd); return d.sep - math.Abs(d.moon-d.sun) }
		e.TotalBegin, e.TotalEnd = fromJulian(edge(inner, tm, -step, begin)), fromJulian(edge(inner, tm, step, end))
	}
	alt := func(jd float64) float64 { f := p.at(jd); return f.altitude(f.sun()) }
	from, to, ok := above(alt, begin, end)
	if !ok {
		return Eclipse{}, false
	}
	e.VisibleFrom, e.VisibleTo = fromJulian(from), fromJulian(to)
	sm := seen(tm, from, to)
	ds := at(sm)
	e.SeenMax, e.SeenMagnitude = fromJulian(sm), (ds.sun+ds.moon-ds.sep)/(2*ds.sun)
	return e, true
}
