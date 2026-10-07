package astro

import (
	"math"
	"time"
)

// Phase is a new or full moon.
type Phase struct {
	Full bool      // false = new moon
	At   time.Time // UTC
}

// Season is an equinox or a solstice.
type Season struct {
	Kind string    // "march_equinox", "june_solstice", "september_equinox", "december_solstice"
	At   time.Time // UTC
}

// lunation holds the ch.49 arguments for one k: an integer for a new moon, +0.5 for a full one.
type lunation struct{ k, T, jde, e, m, mp, f, om float64 }

func newLunation(k float64) lunation {
	T := k / 1236.85
	T2, T3, T4 := T*T, T*T*T, T*T*T*T
	return lunation{
		k:   k,
		T:   T,
		jde: 2451550.09766 + 29.530588861*k + 0.00015437*T2 - 0.000000150*T3 + 0.00000000073*T4,
		e:   1 - 0.002516*T - 0.0000074*T2,
		m:   norm360(2.5534 + 29.10535670*k - 0.0000014*T2 - 0.00000011*T3),
		mp:  norm360(201.5643 + 385.81693528*k + 0.0107582*T2 + 0.00001238*T3 - 0.000000058*T4),
		f:   norm360(160.7108 + 390.67050284*k - 0.0016118*T2 - 0.00000227*T3 + 0.000000011*T4),
		om:  norm360(124.7746 - 1.56375588*k + 0.0020672*T2 + 0.00000215*T3),
	}
}

func (l lunation) full() bool { return l.k != math.Floor(l.k) }

// lunations covers every new and full moon of a year, with a margin on both sides.
func lunations(year int) []lunation {
	k0 := math.Floor(float64(year-2000)*12.3685) - 1
	out := make([]lunation, 0, 32)
	for i := range 32 {
		out = append(out, newLunation(k0+float64(i)/2))
	}
	return out
}

var tab49A = [14][3]float64{
	{299.77, 0.107408, 0.000325},
	{251.88, 0.016321, 0.000165},
	{251.83, 26.651886, 0.000164},
	{349.42, 36.412478, 0.000126},
	{84.66, 18.206239, 0.000110},
	{141.74, 53.303771, 0.000062},
	{207.14, 2.453732, 0.000060},
	{154.84, 7.306860, 0.000056},
	{34.52, 27.261239, 0.000047},
	{207.19, 0.121824, 0.000042},
	{291.34, 1.844379, 0.000040},
	{161.72, 24.198154, 0.000037},
	{239.56, 25.513099, 0.000035},
	{331.55, 3.592518, 0.000023},
}

// phase is the true instant (JDE) of a new or full moon, ch.49.
func (l lunation) phase() float64 {
	m, mp, f, om, e := l.m, l.mp, l.f, l.om, l.e
	c := [7]float64{-0.40720, 0.17241, 0.01608, 0.01039, 0.00739, -0.00514, 0.00208}
	if l.full() {
		c = [7]float64{-0.40614, 0.17302, 0.01614, 0.01043, 0.00734, -0.00515, 0.00209}
	}
	s := c[0]*sind(mp) + c[1]*e*sind(m) + c[2]*sind(2*mp) + c[3]*sind(2*f) +
		c[4]*e*sind(mp-m) + c[5]*e*sind(mp+m) + c[6]*e*e*sind(2*m) -
		0.00111*sind(mp-2*f) - 0.00057*sind(mp+2*f) + 0.00056*e*sind(2*mp+m) -
		0.00042*sind(3*mp) + 0.00042*e*sind(m+2*f) + 0.00038*e*sind(m-2*f) -
		0.00024*e*sind(2*mp-m) - 0.00017*sind(om) - 0.00007*sind(mp+2*m) +
		0.00004*sind(2*mp-2*f) + 0.00004*sind(3*m) + 0.00003*sind(mp+m-2*f) +
		0.00003*sind(2*mp+2*f) - 0.00003*sind(mp+m+2*f) + 0.00003*sind(mp-m+2*f) -
		0.00002*sind(mp-m-2*f) - 0.00002*sind(3*mp+m) + 0.00002*sind(4*mp)
	for i, a := range tab49A {
		arg := a[0] + a[1]*l.k
		if i == 0 {
			arg -= 0.009173 * l.T * l.T
		}
		s += a[2] * sind(arg)
	}
	return l.jde + s
}

// MoonPhases returns every new and full moon whose instant falls in that UTC calendar year, in time order.
func MoonPhases(year int) []Phase {
	a, b := yearSpan(year)
	var out []Phase
	for _, l := range lunations(year) {
		if jd := tt2ut(l.phase()); jd >= a && jd < b {
			out = append(out, Phase{Full: l.full(), At: fromJulian(jd)})
		}
	}
	return out
}

// ch.27, tables 27.A (before year 1000) and 27.B.
var (
	tab27A = [4][5]float64{
		{1721139.29189, 365242.13740, 0.06134, 0.00111, -0.00071},
		{1721233.25401, 365241.72562, -0.05323, 0.00907, 0.00025},
		{1721325.70455, 365242.49558, -0.11677, -0.00297, 0.00074},
		{1721414.39987, 365242.88257, -0.00769, -0.00933, -0.00006},
	}
	tab27B = [4][5]float64{
		{2451623.80984, 365242.37404, 0.05169, -0.00411, -0.00057},
		{2451716.56767, 365241.62603, 0.00325, 0.00888, -0.00030},
		{2451810.21715, 365242.01767, -0.11575, 0.00337, 0.00078},
		{2451900.05952, 365242.74049, -0.06223, -0.00823, 0.00032},
	}
	tab27C = [24][3]float64{
		{485, 324.96, 1934.136}, {203, 337.23, 32964.467}, {199, 342.08, 20.186},
		{182, 27.85, 445267.112}, {156, 73.14, 45036.886}, {136, 171.52, 22518.443},
		{77, 222.54, 65928.934}, {74, 296.72, 3034.906}, {70, 243.58, 9037.513},
		{58, 119.81, 33718.147}, {52, 297.17, 150.678}, {50, 21.02, 2281.226},
		{45, 247.54, 29929.562}, {44, 325.15, 31555.956}, {29, 60.93, 4443.417},
		{18, 155.12, 67555.328}, {17, 288.79, 4562.452}, {16, 198.04, 62894.029},
		{14, 199.76, 31436.921}, {12, 95.39, 14577.848}, {12, 287.11, 31931.756},
		{12, 320.81, 34777.259}, {9, 227.73, 1222.114}, {8, 15.45, 16859.074},
	}
)

// seasonMean is JDE0 of the i-th equinox or solstice of a year (0 = March), ch.27.
func seasonMean(year, i int) float64 {
	c, y := tab27B[i], float64(year-2000)/1000
	if year < 1000 {
		c, y = tab27A[i], float64(year)/1000
	}
	return c[0] + y*(c[1]+y*(c[2]+y*(c[3]+y*c[4])))
}

// seasonJDE adds the periodic terms of ch.27 to JDE0.
func seasonJDE(year, i int) float64 {
	jde0 := seasonMean(year, i)
	T := (jde0 - j2000) / 36525
	w := 35999.373*T - 2.47
	dl := 1 + 0.0334*cosd(w) + 0.0007*cosd(2*w)
	var s float64
	for _, t := range tab27C {
		s += t[0] * cosd(t[1]+t[2]*T)
	}
	return jde0 + 0.00001*s/dl
}

// Seasons returns the year's two equinoxes and two solstices, in order.
func Seasons(year int) []Season {
	kinds := [4]string{"march_equinox", "june_solstice", "september_equinox", "december_solstice"}
	out := make([]Season, 4)
	for i, k := range kinds {
		out[i] = Season{Kind: k, At: fromJulian(tt2ut(seasonJDE(year, i)))}
	}
	return out
}

// Easter is Easter Sunday of the Gregorian calendar, ch.8. It follows the
// church's table moon, not the sky's, so it is computed, not read off a full moon.
func Easter(year int) time.Time {
	a, b, c := year%19, year/100, year%100
	d, e := b/4, b%4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i, k := c/4, c%4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	n := h + l - 7*m + 114
	return time.Date(year, time.Month(n/31), n%31+1, 0, 0, 0, 0, time.UTC)
}
