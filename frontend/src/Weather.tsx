import { useEffect, useState } from "react";
import { api, ApiError } from "./api";
import { localeOf, useT } from "./i18n";

// Weather on the home screen. The backend fetches the village's provider
// (ARSO or Open-Meteo) and trims it, so the provider never sees a villager's
// browser and no third-party frame sits on a logged-in page. Attribution
// stays visible: the data is theirs. The page knows no provider: the icon is
// a key from weather.go's small set, and the text is a dictionary key
// (Open-Meteo) or the provider's own words (ARSO), which t() passes through.

// link: the provider's page for that day, absent where the provider has none.
type Day = { date: string; icon: string; text: string; min: string; max: string; rain: string; link?: string };
type W = { place: string; now: string; now_icon: string; now_text: string; wind: string; days: Day[]; source: string; fetched: string };

const GLYPH: Record<string, string> = {
  "clear": "☀️", "clear-night": "🌙", "mostly-clear": "🌤️", "partly-cloudy": "⛅", "mostly-cloudy": "🌥️", "overcast": "☁️",
  "fog": "🌫️", "drizzle": "☔", "rain": "🌧️", "snow": "🌨️", "thunder": "⛈️",
};
const emoji = (icon: string) => GLYPH[icon] || "🌡️";

// Five days on screen, and ‹ › page by five, not by one: a tap shows the next
// five whole, so the up to fifteen a provider gives are three pages, two taps
// end to end, and no square slides under the thumb while the eye is on it.
const PAGE = 5;

// Rain in whole millimetres: "💧 21.6 mm" and even "💧 0.3 mm" are wider than
// a square on a 375 px phone, and a forecast's tenth of a millimetre tells a
// garden nothing. A trace under one millimetre reads "<1", because it is not 0.
const mm = (v: string) => { const n = parseFloat(v); return isNaN(n) ? v : n > 0 && n < 1 ? "<1" : String(Math.round(n)); };

export function Weather() {
  const { t, lang } = useT();
  const [w, setW] = useState<W | null>(null);
  const [start, setStart] = useState(0);
  const [err, setErr] = useState(false);
  const [off, setOff] = useState(false);
  // 404 is a village with no WEATHER_LOCATION: the panel says so, not a fault.
  useEffect(() => { api<W>("/weather").then(setW).catch((e) => (e instanceof ApiError && e.status === 404 ? setOff(true) : setErr(true))); }, []);

  // The chip above still lights, so the topper says why it is empty.
  if (off) return <div className="parchment weather"><p className="small muted">{t("No weather for this village.")}</p></div>;
  if (err) return <div className="parchment weather"><p className="small muted">{t("The weather service did not answer.")}</p></div>;
  if (!w) return <div className="parchment weather"><p className="small muted">{t("Loading…")}</p></div>;
  const locale = localeOf(lang);
  // The backend serves the last good copy when ARSO is down, so the panel must
  // say how old the reading is rather than promise it is fresh.
  const fetchedAt = new Date(w.fetched);
  const mins = Math.max(0, Math.round((Date.now() - fetchedAt.getTime()) / 60000));
  const stale = mins > 60;
  const age = mins < 60 ? `${mins} min` : fetchedAt.toLocaleString(locale, { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
  // Fifteen days hold two of each weekday, so a day says its date too. Joined
  // by a space, not the locale's comma: "pon., 26." is wider than a square on
  // a 375 px phone and wraps, "pon. 26." is not.
  const dayName = (d: string, i: number) => {
    if (i === 0) return t("today");
    const at = new Date(d + "T00:00");
    return `${at.toLocaleDateString(locale, { weekday: "short" })} ${at.toLocaleDateString(locale, { day: "numeric" })}`;
  };
  const shown = w.days.slice(start, start + PAGE);
  // The arrows stand on the days' own line, one at each side (owner's ask
  // 2026-10-08). An arrow is absent at its end of the forecast, never
  // wrapping round, so the end is seen; its empty column keeps the days still.
  const paged = w.days.length > PAGE;
  const arrow = (show: boolean, label: string, to: number, glyph: string) => show
    ? <button type="button" className="lesser w-arrow" aria-label={t(label)} onClick={() => setStart(to)}>{glyph}</button>
    : <span className="w-arrow" aria-hidden="true" />;

  return (
    <div className="parchment weather">
      <div className="w-now">
        <span className="w-icon">{emoji(w.now_icon)}</span>
        <span className="w-temp">{w.now}°</span>
        <span className="w-where">{w.place}<br /><span className="small">{t(w.now_text)}{w.wind ? ` · ${w.wind}` : ""}</span></span>
      </div>
      <div className={"w-row" + (paged ? " paged" : "")}>
        {paged && arrow(start > 0, "Earlier days", start - PAGE, "‹")}
        <div className="w-days">
          {shown.map((d, j) => {
            const inner = (<>
              <div className="small">{dayName(d.date, start + j)}</div>
              <div className="w-icon-sm">{emoji(d.icon)}</div>
              <div className="w-range"><strong>{d.max}°</strong> <span className="muted">{d.min}°</span></div>
              {d.rain && <div className="small w-rain">💧 {mm(d.rain)}<span className="unit"> mm</span></div>}
            </>);
            // A day the provider has a page for opens it beside the app, so an
            // installed portal does not navigate away, and without a referrer,
            // so the provider is not told the portal's address; one it has none
            // for is a framed box that does nothing (CLAUDE.md § The button frame).
            return d.link
              ? <a key={d.date} className="w-day" href={d.link} target="_blank" rel="noopener noreferrer" title={`${t("Details at")} ${w.source}`}>{inner}</a>
              : <div key={d.date} className="w-day">{inner}</div>;
          })}
        </div>
        {paged && arrow(start + PAGE < w.days.length, "Later days", start + PAGE, "›")}
      </div>
      <p className={"small w-src" + (stale ? " stale-note" : " muted")}>
        {/* CC BY asks for the licensor's name and a link; a link loads nothing until tapped. */}
        {t("Source")}: {w.source === "Open-Meteo.com" ? <a href="https://open-meteo.com" target="_blank" rel="noopener noreferrer">Open-Meteo.com</a> : w.source}. {t("Forecast for")} {w.place}, {t("not measured at the village.")}{" "}
        {stale ? t("This reading is over an hour old.") : t("Read")} {age}
      </p>
    </div>
  );
}
