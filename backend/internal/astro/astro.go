// Package astro computes a yearly almanac after Jean Meeus, "Astronomical
// Algorithms" (2nd ed.): new and full moons, equinoxes and solstices, the
// eclipses seen from one place, and sunrise and sunset.
package astro

import (
	"math"
	"time"
)

const (
	rad     = math.Pi / 180
	j2000   = 2451545.0
	earthKm = 6378.14
	auKm    = 149597870.7
)

func sind(x float64) float64 { return math.Sin(x * rad) }
func cosd(x float64) float64 { return math.Cos(x * rad) }

func norm360(x float64) float64 {
	x = math.Mod(x, 360)
	if x < 0 {
		x += 360
	}
	return x
}

// ch.7: the Unix epoch is JD 2440587.5.
func julian(t time.Time) float64 {
	return 2440587.5 + (float64(t.Unix())+float64(t.Nanosecond())/1e9)/86400
}

func fromJulian(jd float64) time.Time {
	s := (jd - 2440587.5) * 86400
	sec := math.Floor(s)
	return time.Unix(int64(sec), int64((s-sec)*1e9)).UTC().Round(time.Second)
}

func yearSpan(year int) (a, b float64) {
	return julian(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)), julian(time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC))
}

// deltaT is TT − UT in seconds, ch.10 with the Espenak–Meeus polynomials.
func deltaT(y float64) float64 {
	switch {
	case y >= 2005 && y < 2050:
		t := y - 2000
		return 62.92 + 0.32217*t + 0.005589*t*t
	case y >= 1986 && y < 2005:
		t := y - 2000
		return 63.86 + 0.3345*t - 0.060374*t*t + 0.0017275*t*t*t + 0.000651814*t*t*t*t + 0.00002373599*t*t*t*t*t
	case y >= 1961 && y < 1986:
		t := y - 1975
		return 45.45 + 1.067*t - t*t/260 - t*t*t/718
	case y >= 2050 && y < 2150:
		u := (y - 1820) / 100
		return -20 + 32*u*u - 0.5628*(2150-y)
	}
	u := (y - 1820) / 100
	return -20 + 32*u*u
}

func ut2tt(jd float64) float64  { return jd + deltaT(2000+(jd-j2000)/365.25)/86400 }
func tt2ut(jde float64) float64 { return jde - deltaT(2000+(jde-j2000)/365.25)/86400 }

// nutation gives Δψ and Δε in degrees, ch.22 (low-precision form).
func nutation(T float64) (dpsi, deps float64) {
	om := 125.04452 - 1934.136261*T + 0.0020708*T*T + T*T*T/450000
	ls := 280.4665 + 36000.7698*T
	lm := 218.3165 + 481267.8813*T
	dpsi = (-17.20*sind(om) - 1.32*sind(2*ls) - 0.23*sind(2*lm) + 0.21*sind(2*om)) / 3600
	deps = (9.20*cosd(om) + 0.57*cosd(2*ls) + 0.10*cosd(2*lm) - 0.09*cosd(2*om)) / 3600
	return dpsi, deps
}

// meanObliquity follows ch.22, eq. 22.2.
func meanObliquity(T float64) float64 {
	return 23.4392911111 - (46.8150*T+0.00059*T*T-0.001813*T*T*T)/3600
}

// sidereal is the mean sidereal time at Greenwich in degrees, ch.12.
func sidereal(jd float64) float64 {
	T := (jd - j2000) / 36525
	return norm360(280.46061837 + 360.98564736629*(jd-j2000) + 0.000387933*T*T - T*T*T/38710000)
}

// equatorial follows ch.13.
func equatorial(lambda, beta, eps float64) (alpha, delta float64) {
	alpha = math.Atan2(sind(lambda)*cosd(eps)-math.Tan(beta*rad)*sind(eps), cosd(lambda)) / rad
	delta = math.Asin(sind(beta)*cosd(eps)+cosd(beta)*sind(eps)*sind(lambda)) / rad
	return norm360(alpha), delta
}

type vec [3]float64

func rect(alpha, delta, r float64) vec {
	return vec{r * cosd(delta) * cosd(alpha), r * cosd(delta) * sind(alpha), r * sind(delta)}
}
func (a vec) sub(b vec) vec     { return vec{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func (a vec) dot(b vec) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func (a vec) norm() float64     { return math.Sqrt(a.dot(a)) }
func angle(a, b vec) float64 {
	c := vec{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
	return math.Atan2(c.norm(), a.dot(b)) / rad
}

// place is an observer at sea level; ρ sin φ′ and ρ cos φ′ follow ch.11.
type place struct{ lat, lon, rhoSin, rhoCos float64 }

func newPlace(lat, lon float64) place {
	u := math.Atan(0.99664719 * math.Tan(lat*rad))
	return place{lat, lon, 0.99664719 * math.Sin(u), math.Cos(u)}
}

// frame is the sky over a place at one UT instant, in Earth radii,
// equatorial coordinates of date.
type frame struct {
	jde, dpsi, eps float64
	obs, zenith    vec
}

func (p place) at(jd float64) frame {
	jde := ut2tt(jd)
	T := (jde - j2000) / 36525
	dpsi, deps := nutation(T)
	eps := meanObliquity(T) + deps
	theta := sidereal(jd) + dpsi*cosd(eps) + p.lon
	return frame{jde, dpsi, eps,
		vec{p.rhoCos * cosd(theta), p.rhoCos * sind(theta), p.rhoSin},
		vec{cosd(p.lat) * cosd(theta), cosd(p.lat) * sind(theta), sind(p.lat)}}
}

// sun and moon are topocentric apparent positions: ch.40 in rectangular form.
func (f frame) sun() vec {
	l, r := sunApparent(f.jde)
	a, d := equatorial(l, 0, f.eps)
	return rect(a, d, r*auKm/earthKm).sub(f.obs)
}

func (f frame) moon() vec {
	l, b, dist := moonPosition(f.jde)
	a, d := equatorial(l+f.dpsi, b, f.eps)
	return rect(a, d, dist/earthKm).sub(f.obs)
}

// altitude is geometric, without refraction, ch.13.
func (f frame) altitude(v vec) float64 { return math.Asin(f.zenith.dot(v)/v.norm()) / rad }

const tenthSecond = 0.1 / 86400

// root narrows a sign change of f inside [a, b].
func root(f func(float64) float64, a, b float64) float64 {
	neg := f(a) < 0
	for b-a > tenthSecond {
		m := (a + b) / 2
		if (f(m) < 0) == neg {
			a = m
		} else {
			b = m
		}
	}
	return (a + b) / 2
}

// edge walks from t, where f < 0, until f turns non-negative or the limit is passed.
func edge(f func(float64) float64, t, step, limit float64) float64 {
	for {
		next := t + step
		if (next-limit)*step > 0 {
			return limit
		}
		if f(next) >= 0 {
			return root(f, t, next)
		}
		t = next
	}
}

// minimize is a golden-section search for the least f in [a, b].
func minimize(f func(float64) float64, a, b float64) float64 {
	const g = 0.6180339887498949
	c, d := b-g*(b-a), a+g*(b-a)
	fc, fd := f(c), f(d)
	for b-a > tenthSecond {
		if fc < fd {
			b, d, fd = d, c, fc
			c = b - g*(b-a)
			fc = f(c)
		} else {
			a, c, fc = c, d, fd
			d = a + g*(b-a)
			fd = f(d)
		}
	}
	return (a + b) / 2
}

// above gives the first and last moment in [a, b] with f > 0, sampled by the minute.
func above(f func(float64) float64, a, b float64) (from, to float64, ok bool) {
	n := max(int(math.Ceil((b-a)*1440)), 1)
	t := func(i int) float64 { return a + (b-a)*float64(i)/float64(n) }
	first, last := -1, -1
	for i := 0; i <= n; i++ {
		if f(t(i)) > 0 {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return 0, 0, false
	}
	from, to = a, b
	if first > 0 {
		from = root(f, t(first-1), t(first))
	}
	if last < n {
		to = root(f, t(last), t(last+1))
	}
	return from, to, true
}
