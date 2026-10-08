package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Weather for the home screen, from the provider WEATHER_PROVIDER names:
// open-meteo (the default — anywhere on earth from WEATHER_COORDS, no key,
// CC BY 4.0, so the panel names Open-Meteo.com) or arso (the Slovenian
// environment agency, by the place name in WEATHER_LOCATION — this village).
//
// The backend fetches and trims it instead of the page framing a widget: an
// iframe would hand the provider every villager's address and the portal URL
// on a logged-in page, and would need a hole in the CSP. Here the provider
// sees one server, once every half hour.
//
// Both adapters emit the same shape. The icon is one of the keys below, which
// the page turns into a glyph; the text is a dictionary key for Open-Meteo
// (the page translates it) and ARSO's own Slovenian words for ARSO.
//
// Days: each provider gives what it has, up to maxWeatherDays, never padded —
// Open-Meteo 15 (its API stops at 16), ARSO 10 (what its own page shows). A
// day carries a link only where the provider has a browser page for that day:
// ARSO has one, Open-Meteo is an API with no page per place or per day, so
// its days carry no link and the page draws them unclickable.

// maxWeatherDays: the owner's ask (2026-10-08), fifteen days to page through.
const maxWeatherDays = 15

// Icon keys. A provider maps its own names onto these; the page knows only these.
const (
	icClear        = "clear"
	icClearNight   = "clear-night"
	icMostlyClear  = "mostly-clear"
	icPartlyCloudy = "partly-cloudy"
	icMostlyCloudy = "mostly-cloudy"
	icOvercast     = "overcast"
	icFog          = "fog"
	icDrizzle      = "drizzle"
	icRain         = "rain"
	icSnow         = "snow"
	icThunder      = "thunder"
)

type weatherDay struct {
	Date string `json:"date"`
	Icon string `json:"icon"`
	Text string `json:"text"`
	Min  string `json:"min"`
	Max  string `json:"max"`
	Rain string `json:"rain"`
	Link string `json:"link,omitempty"` // the provider's page for this day; absent where it has none
}

type weatherOut struct {
	Place   string       `json:"place"`
	Now     string       `json:"now"`
	NowIcon string       `json:"now_icon"`
	NowText string       `json:"now_text"`
	Wind    string       `json:"wind"`
	Days    []weatherDay `json:"days"`
	Fetched string       `json:"fetched"`
	Source  string       `json:"source"`
}

type weatherCache struct {
	mu   sync.Mutex
	at   time.Time
	body []byte
}

var wcache weatherCache

func (s *Server) weather(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.WeatherOn() {
		// Not configured is not down: on a 404 the panel says this village
		// has no forecast, where a 503 would have it say the service did not
		// answer.
		writeErr(w, 404, "no weather configured")
		return
	}
	now := s.now()
	wcache.mu.Lock()
	// A copy from before midnight is not fresh after it, however young: its
	// first day is yesterday, and ARSO's day links count from today.
	fresh := wcache.body != nil && now.Sub(wcache.at) < 30*time.Minute && now.Format("2006-01-02") == wcache.at.Format("2006-01-02")
	body := wcache.body
	wcache.mu.Unlock()
	if !fresh {
		out, err := s.fetchWeather(r.Context())
		if err != nil {
			if body != nil { // serve the stale copy rather than nothing
				writeRaw(w, body, now)
				return
			}
			writeErr(w, 502, "weather unavailable")
			return
		}
		body, _ = json.Marshal(out)
		wcache.mu.Lock()
		wcache.at, wcache.body = now, body
		wcache.mu.Unlock()
	}
	writeRaw(w, body, now)
}

// writeRaw lets the browser keep the copy fifteen minutes, but not past
// midnight, for the same reason the cache above does not.
func writeRaw(w http.ResponseWriter, body []byte, now time.Time) {
	y, m, d := now.Date()
	age := min(900, int(time.Date(y, m, d+1, 0, 0, 0, 0, now.Location()).Sub(now).Seconds()))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age="+strconv.Itoa(age))
	w.Write(body)
}

// fetchWeather asks the configured provider. PUBLIC_URL rides in the
// User-Agent as the contact: what the provider sees if anyone looks at their
// logs, and an anonymous poller is the thing a service blocks first.
func (s *Server) fetchWeather(ctx context.Context) (*weatherOut, error) {
	if s.cfg.WeatherProvider == "arso" {
		return fetchARSO(ctx, s.cfg.WeatherLocation, s.cfg.PublicURL, s.now())
	}
	return fetchOpenMeteo(ctx, s.cfg.WeatherCoords, s.cfg.WeatherLocation, s.cfg.PublicURL, s.now())
}

func getJSON(ctx context.Context, u, contact string, into any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "porta-pagi/1 (village portal; "+contact+")")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, into)
}

// ---- Open-Meteo ------------------------------------------------------------
//
// https://open-meteo.com — a forecast for any lat,lon, WMO weather codes.
// Attribution is a licence term (CC BY 4.0): the panel names Open-Meteo.com.

type openMeteoDoc struct {
	Current struct {
		Temp  float64 `json:"temperature_2m"`
		Code  int     `json:"weather_code"`
		Wind  float64 `json:"wind_speed_10m"`
		IsDay int     `json:"is_day"`
	} `json:"current"`
	Daily struct {
		Time []string  `json:"time"`
		Code []int     `json:"weather_code"`
		Max  []float64 `json:"temperature_2m_max"`
		Min  []float64 `json:"temperature_2m_min"`
		Rain []float64 `json:"precipitation_sum"`
	} `json:"daily"`
}

func fetchOpenMeteo(ctx context.Context, coords, place, contact string, now time.Time) (*weatherOut, error) {
	lat, lon, ok := strings.Cut(strings.TrimSpace(coords), ",")
	if !ok {
		return nil, errNoWeather
	}
	q := url.Values{
		"latitude": {strings.TrimSpace(lat)}, "longitude": {strings.TrimSpace(lon)},
		"current":       {"temperature_2m,weather_code,wind_speed_10m,is_day"},
		"daily":         {"weather_code,temperature_2m_max,temperature_2m_min,precipitation_sum"},
		"timezone":      {"auto"}, // so the first day is the place's own today
		"forecast_days": {strconv.Itoa(maxWeatherDays)},
	}
	var doc openMeteoDoc
	if err := getJSON(ctx, "https://api.open-meteo.com/v1/forecast?"+q.Encode(), contact, &doc); err != nil {
		return nil, err
	}
	return fromOpenMeteo(&doc, coords, place, now)
}

// fromOpenMeteo trims the answer into the panel's shape. No day carries a
// link: Open-Meteo has no page to send a villager to.
func fromOpenMeteo(doc *openMeteoDoc, coords, place string, now time.Time) (*weatherOut, error) {
	if place == "" {
		place = strings.TrimSpace(coords)
	}
	out := &weatherOut{Fetched: now.UTC().Format(time.RFC3339), Source: "Open-Meteo.com", Place: place}
	out.NowIcon, out.NowText = wmo(doc.Current.Code, doc.Current.IsDay == 1)
	out.Now = fmt.Sprintf("%.0f", doc.Current.Temp)
	out.Wind = fmt.Sprintf("%.0f km/h", doc.Current.Wind)
	for i, date := range doc.Daily.Time {
		if i >= len(doc.Daily.Code) || i >= len(doc.Daily.Max) || i >= len(doc.Daily.Min) || len(out.Days) >= maxWeatherDays {
			break
		}
		day := weatherDay{Date: date, Min: fmt.Sprintf("%.0f", doc.Daily.Min[i]), Max: fmt.Sprintf("%.0f", doc.Daily.Max[i])}
		day.Icon, day.Text = wmo(doc.Daily.Code[i], true)
		if i < len(doc.Daily.Rain) && doc.Daily.Rain[i] >= 0.1 { // millimetres, only when worth saying
			day.Rain = strconv.FormatFloat(doc.Daily.Rain[i], 'f', 1, 64)
		}
		out.Days = append(out.Days, day)
	}
	if len(out.Days) == 0 {
		return nil, errNoWeather
	}
	return out, nil
}

// wmo maps a WMO weather code to an icon key and a dictionary key.
func wmo(code int, day bool) (icon, text string) {
	switch {
	case code == 0 && day:
		return icClear, "Clear sky"
	case code == 0:
		return icClearNight, "Clear sky"
	case code == 1:
		return icMostlyClear, "Mainly clear"
	case code == 2:
		return icPartlyCloudy, "Partly cloudy"
	case code == 3:
		return icOvercast, "Overcast"
	case code == 45 || code == 48:
		return icFog, "Fog"
	case code >= 51 && code <= 57:
		return icDrizzle, "Drizzle"
	case code >= 61 && code <= 67:
		return icRain, "Rain"
	case code >= 71 && code <= 77:
		return icSnow, "Snow"
	case code >= 80 && code <= 82:
		return icRain, "Rain showers"
	case code == 85 || code == 86:
		return icSnow, "Snow showers"
	case code >= 95:
		return icThunder, "Thunderstorm"
	}
	return "", ""
}

// ---- ARSO -------------------------------------------------------------------
//
// https://vreme.arso.gov.si — public data, attribution shown in the UI. The
// texts are ARSO's own Slovenian words and go to the page as they are.
//
// forecast24h carries eleven days, today and ten more, but ARSO's own page
// shows ten: it drops the days before today (the answer can still start
// with yesterday just after midnight) and keeps the first ten, because the
// eleventh is a stub — the 6-hour forecast ends at its midnight. That page
// has an address per day, /napoved/<place>/graf/<n>, n counted from today in
// that same list of ten; /graf/10 draws an empty page. So the panel takes
// the same ten days, and day n links to /graf/n. The address is a route of
// that page's script, not a published API (CLAUDE.md § No third party in
// the page says what was checked); the day.Link line is the one to cut.

const arsoDays = 10

type arsoFeatureSet struct {
	Features []struct {
		Properties struct {
			Title string `json:"title"`
			Days  []struct {
				Date     string              `json:"date"`
				Timeline []map[string]string `json:"timeline"`
			} `json:"days"`
		} `json:"properties"`
	} `json:"features"`
}

type arsoDoc struct {
	Forecast3h  arsoFeatureSet `json:"forecast3h"`
	Forecast24h arsoFeatureSet `json:"forecast24h"`
	Observation arsoFeatureSet `json:"observation"`
}

func fetchARSO(ctx context.Context, loc, contact string, now time.Time) (*weatherOut, error) {
	var doc arsoDoc
	if err := getJSON(ctx, "https://vreme.arso.gov.si/api/1.0/location/?location="+url.QueryEscape(loc), contact, &doc); err != nil {
		return nil, err
	}
	return fromARSO(&doc, now)
}

// fromARSO trims the answer into the panel's shape. now is the village's
// clock, and its date is today.
func fromARSO(doc *arsoDoc, now time.Time) (*weatherOut, error) {
	out := &weatherOut{Fetched: now.UTC().Format(time.RFC3339), Source: "ARSO"}
	first := func(fs arsoFeatureSet) map[string]string {
		if len(fs.Features) > 0 && len(fs.Features[0].Properties.Days) > 0 && len(fs.Features[0].Properties.Days[0].Timeline) > 0 {
			return fs.Features[0].Properties.Days[0].Timeline[0]
		}
		return nil
	}
	if len(doc.Forecast3h.Features) > 0 {
		out.Place = doc.Forecast3h.Features[0].Properties.Title
	}
	// Now: the observation if there is one, else the nearest 3-hour step.
	cur := first(doc.Observation)
	if cur == nil || cur["t"] == "" {
		cur = first(doc.Forecast3h)
	}
	if cur != nil {
		out.Now, out.NowIcon, out.NowText = cur["t"], arsoIcon(cur["clouds_icon_wwsyn_icon"]), cur["clouds_shortText_wwsyn_shortText"]
		if out.NowText == "" {
			out.NowText = cur["clouds_shortText"]
		}
		out.Wind = cur["ff_shortText"]
	}
	if len(doc.Forecast24h.Features) > 0 {
		p := doc.Forecast24h.Features[0].Properties
		today := now.Format("2006-01-02")
		for _, d := range p.Days {
			if d.Date < today { // ISO dates compare as strings
				continue
			}
			if len(d.Timeline) == 0 || len(out.Days) >= min(arsoDays, maxWeatherDays) {
				break
			}
			tl := d.Timeline[0]
			day := weatherDay{Date: d.Date, Icon: arsoIcon(tl["clouds_icon_wwsyn_icon"]), Min: tl["tnsyn"], Max: tl["txsyn"], Rain: tl["tp_24h_acc"]}
			day.Text = tl["clouds_shortText_wwsyn_shortText"]
			if day.Text == "" {
				day.Text = tl["clouds_shortText"]
			}
			// Millimetres, but only when it is worth saying.
			if v, err := strconv.ParseFloat(strings.TrimSpace(day.Rain), 64); err != nil || v < 0.1 {
				day.Rain = ""
			}
			if p.Title != "" {
				day.Link = "https://vreme.arso.gov.si/napoved/" + url.PathEscape(p.Title) + "/graf/" + strconv.Itoa(len(out.Days))
			}
			out.Days = append(out.Days, day)
		}
	}
	if out.Now == "" && len(out.Days) == 0 {
		return nil, errNoWeather
	}
	return out, nil
}

// arsoIcon maps ARSO's icon names — cloud cover, then the weather, then
// day/night, e.g. partCloudy_lightRA_day — onto the icon keys.
func arsoIcon(name string) string {
	i := strings.ToLower(name)
	switch {
	case strings.Contains(i, "ts"):
		return icThunder
	case strings.Contains(i, "sn"): // SN and RASN alike
		return icSnow
	case strings.Contains(i, "dz"):
		return icDrizzle
	case strings.Contains(i, "ra") || strings.Contains(i, "rain"):
		return icRain
	case strings.Contains(i, "fg"):
		return icFog
	case strings.Contains(i, "overcast"):
		return icOvercast
	case strings.Contains(i, "prevcloudy") || strings.Contains(i, "modcloudy") || strings.Contains(i, "mostcloudy"):
		return icMostlyCloudy // ARSO writes prevCloudy for "pretežno oblačno"
	case strings.Contains(i, "partcloudy"):
		return icPartlyCloudy
	case strings.Contains(i, "slightcloudy"):
		return icMostlyClear
	case strings.Contains(i, "clear_night"):
		return icClearNight
	case strings.Contains(i, "clear"):
		return icClear
	}
	return ""
}

type weatherErr string

func (e weatherErr) Error() string { return string(e) }

const errNoWeather = weatherErr("the weather provider returned nothing usable")
