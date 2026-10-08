package httpapi

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"
)

func readFixture(t *testing.T, name string, into any) {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatal(err)
	}
}

// TestOpenMeteoDays: a recorded answer of 16 days (the API's most) comes out
// as 15, today first, and no day carries a link — Open-Meteo has no page.
func TestOpenMeteoDays(t *testing.T) {
	var doc openMeteoDoc
	readFixture(t, "weather-open-meteo.json", &doc)
	if len(doc.Daily.Time) != 16 {
		t.Fatalf("fixture has %d days, want the API's 16", len(doc.Daily.Time))
	}
	out, err := fromOpenMeteo(&doc, "37.65,-8.22", "Sobreiro Torto", time.Date(2026, 10, 8, 11, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Days) != maxWeatherDays {
		t.Fatalf("%d days, want %d", len(out.Days), maxWeatherDays)
	}
	if out.Days[0].Date != "2026-10-08" || out.Days[14].Date != "2026-10-22" {
		t.Errorf("days run %s..%s", out.Days[0].Date, out.Days[14].Date)
	}
	for _, d := range out.Days {
		if d.Link != "" {
			t.Errorf("%s carries a link %q", d.Date, d.Link)
		}
	}
	if d := out.Days[0]; d.Icon != icFog || d.Text != "Fog" || d.Max != "27" || d.Min != "13" || d.Rain != "" {
		t.Errorf("first day %+v", d)
	}
	if d := out.Days[14]; d.Icon != icDrizzle || d.Rain != "2.4" {
		t.Errorf("last day %+v", d)
	}
	if out.Place != "Sobreiro Torto" || out.Source != "Open-Meteo.com" || out.Now != "21" || out.NowIcon != icClear || out.Wind != "9 km/h" {
		t.Errorf("now %+v", out)
	}
	// The link stays out of the JSON, so the page reads a day as unclickable.
	if raw, _ := json.Marshal(out.Days[0]); containsKey(raw, "link") {
		t.Errorf("an empty link is in the JSON: %s", raw)
	}
}

// TestARSODays: ARSO answers eleven days; the panel takes the ten its own
// page shows, from today, and day n links to that page's /graf/n. Just
// after midnight the answer still starts with yesterday, which is dropped,
// and the links count from the new today.
func TestARSODays(t *testing.T) {
	var doc arsoDoc
	readFixture(t, "weather-arso.json", &doc)
	if n := len(doc.Forecast24h.Features[0].Properties.Days); n != 11 {
		t.Fatalf("fixture has %d days, want ARSO's 11", n)
	}
	lj, err := time.LoadLocation("Europe/Ljubljana")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		now         time.Time
		first, last string
	}{
		{time.Date(2026, 10, 8, 13, 0, 0, 0, lj), "2026-10-08", "2026-10-17"},
		{time.Date(2026, 10, 9, 0, 10, 0, 0, lj), "2026-10-09", "2026-10-18"},
	} {
		out, err := fromARSO(&doc, c.now)
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Days) != arsoDays || out.Days[0].Date != c.first || out.Days[len(out.Days)-1].Date != c.last {
			t.Errorf("on %s: %d days, %s..%s", c.now.Format("2006-01-02"), len(out.Days), out.Days[0].Date, out.Days[len(out.Days)-1].Date)
		}
		for i, d := range out.Days {
			if want := "https://vreme.arso.gov.si/napoved/Ko%C4%8Devje/graf/" + strconv.Itoa(i); d.Link != want {
				t.Errorf("day %d links %q, want %q", i, d.Link, want)
			}
		}
	}
	out, _ := fromARSO(&doc, time.Date(2026, 10, 8, 13, 0, 0, 0, lj))
	if d := out.Days[0]; d.Icon != icThunder || d.Text != "možnost neviht" || d.Max != "18" || d.Min != "12" || d.Rain != "21.4" {
		t.Errorf("first day %+v", d)
	}
	if d := out.Days[5]; d.Icon != icClear || d.Rain != "" { // "0" mm is not worth saying
		t.Errorf("sixth day %+v", d)
	}
	if out.Place != "Kočevje" || out.Source != "ARSO" || out.Now != "18" || out.NowIcon != icMostlyCloudy || out.Wind != "šibek J" {
		t.Errorf("now %+v", out)
	}
}

func containsKey(raw []byte, key string) bool {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	_, ok := m[key]
	return ok
}

// TestWeatherIcons: both providers land on the same small set of icon keys,
// and Open-Meteo's texts are dictionary keys.
func TestWeatherIcons(t *testing.T) {
	for _, c := range []struct {
		code int
		day  bool
		icon string
		text string
	}{
		{0, true, icClear, "Clear sky"}, {0, false, icClearNight, "Clear sky"}, {2, true, icPartlyCloudy, "Partly cloudy"},
		{45, true, icFog, "Fog"}, {55, true, icDrizzle, "Drizzle"}, {63, true, icRain, "Rain"}, {73, true, icSnow, "Snow"},
		{81, true, icRain, "Rain showers"}, {95, true, icThunder, "Thunderstorm"},
	} {
		if icon, text := wmo(c.code, c.day); icon != c.icon || text != c.text {
			t.Errorf("wmo(%d, %v) = %q %q", c.code, c.day, icon, text)
		}
		if _, ok := dicts["sl"][c.text]; !ok {
			t.Errorf("sl.json lacks %q", c.text)
		}
	}
	for name, want := range map[string]string{
		"clear_day": icClear, "clear_night": icClearNight, "partCloudy_day": icPartlyCloudy, "overcast_lightRA_day": icRain,
		"prevCloudy_modRASN_night": icSnow, "prevCloudy_day": icMostlyCloudy, "modCloudy_lightDZ_day": icDrizzle, "overcast_TS_day": icThunder, "FG_day": icFog, "slightCloudy_day": icMostlyClear, "": "",
	} {
		if got := arsoIcon(name); got != want {
			t.Errorf("arsoIcon(%q) = %q, want %q", name, got, want)
		}
	}
}
