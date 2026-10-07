package astro

import (
	"math"
	"time"
)

// sunApparent gives the sun's apparent longitude in degrees and its distance
// in AU, ch.25, with ch.22 nutation and aberration in place of the sin Ω shortcut.
func sunApparent(jde float64) (lambda, r float64) {
	T := (jde - j2000) / 36525
	l0 := 280.46646 + 36000.76983*T + 0.0003032*T*T
	m := 357.52911 + 35999.05029*T - 0.0001537*T*T
	e := 0.016708634 - 0.000042037*T - 0.0000001267*T*T
	c := (1.914602-0.004817*T-0.000014*T*T)*sind(m) + (0.019993-0.000101*T)*sind(2*m) + 0.000289*sind(3*m)
	r = 1.000001018 * (1 - e*e) / (1 + e*cosd(m+c))
	dpsi, _ := nutation(T)
	return norm360(l0 + c + dpsi - 20.4898/3600/r), r
}

// SunRiseSet gives sunrise and sunset (UTC) on the local calendar day that contains `day` in loc,
// for the sun's upper limb at the standard −0.833° altitude. ok=false in polar day or night.
func SunRiseSet(day time.Time, loc *time.Location, lat, lon float64) (rise, set time.Time, ok bool) {
	y, m, d := day.In(loc).Date()
	a := julian(time.Date(y, m, d, 0, 0, 0, 0, loc))
	b := julian(time.Date(y, m, d+1, 0, 0, 0, 0, loc))
	p := newPlace(lat, lon)
	h := func(jd float64) float64 { f := p.at(jd); return f.altitude(f.sun()) + 0.833 }
	n := int(math.Round((b - a) * 288))
	t := func(i int) float64 { return a + (b-a)*float64(i)/float64(n) }
	var r, s float64
	prev := h(a)
	for i := 1; i <= n; i++ {
		cur := h(t(i))
		if prev < 0 && cur >= 0 && r == 0 {
			r = root(h, t(i-1), t(i))
		}
		if prev >= 0 && cur < 0 && s == 0 {
			s = root(h, t(i-1), t(i))
		}
		prev = cur
	}
	if r == 0 || s == 0 {
		return time.Time{}, time.Time{}, false
	}
	return fromJulian(r), fromJulian(s), true
}
