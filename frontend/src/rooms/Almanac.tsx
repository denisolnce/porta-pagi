import { useEffect, useState } from "react";
import type { Me } from "../api";
import { Share, madeOf } from "../links";
import { localeOf, useT } from "../i18n";
import { When, isOver } from "./shared";
import { Thread } from "./Thread";

// The almanac: the sky over the village, written by the portal once a year
// (almanac.go) — new and full moons, equinoxes, solstices, and the eclipses
// the village can see. Nobody creates, edits or answers a row; a house may
// talk about one. The words are the page's, per language: a row carries its
// kind, its title's English key (almanacTitles in almanac.go, the one map the
// comment's push words too), its time in the village's clock and a detail.

// The seasons turn the other way south of the equator, and so do their icons.
export function skyIcon(row: { kind: string; detail?: string }): string {
  const south = detailOf(row).south === true;
  const season: Record<string, [string, string]> = {
    march_equinox: ["🌱", "🍂"], june_solstice: ["☀️", "❄️"],
    september_equinox: ["🍂", "🌱"], december_solstice: ["❄️", "☀️"],
  };
  if (season[row.kind]) return season[row.kind][south ? 1 : 0];
  return { new_moon: "🌑", full_moon: "🌕", solar_eclipse: "🌚", lunar_eclipse: "🌘" }[row.kind] || "✨";
}

const detailOf = (row: { detail?: string }): any => {
  try { return JSON.parse(row.detail || "{}"); } catch { return {}; }
};

// One line about each full moon of the northern year; the south gets the line
// every full moon shares, because these are the north's months.
const FULL_MOON_MONTH = [
  "The highest full moon of winter: it rides across the sky the whole long night, and on snow it is almost bright enough to read by.",
  "Late winter's full moon. The days are already longer by two or three minutes each day.",
  "The full moon nearest the March equinox rises almost due east and sets almost due west.",
  "Spring's full moon: from now until summer each full moon rides a little lower than the last.",
  "Summer's full moons stay low in the south, as the winter sun does.",
  "The lowest full moon of the year, on the shortest nights: it skims the southern horizon.",
  "A low summer moon over the warmest nights of the year.",
  "The August full moon often outshines the Perseid meteors, which peak around the 12th.",
  "The harvest moon: near the equinox it rises at almost the same hour several evenings running, so the light lasts past sunset.",
  "The hunter's moon, after the harvest moon: again it rises soon after sunset several evenings in a row.",
  "The nights are long now, and the full moon climbs high, as the sun did in summer.",
  "Near the winter solstice the full moon climbs highest and stays up longest of the whole year.",
];

const SEASON_LINE: Record<string, [string, string]> = {
  march_equinox: ["Astronomical spring begins: from today the days are longer than the nights.", "Astronomical autumn begins: from today the nights are longer than the days."],
  june_solstice: ["The sun stands highest at noon: the longest day and the shortest night. Astronomical summer begins.", "The sun stands lowest at noon: the shortest day and the longest night. Astronomical winter begins."],
  september_equinox: ["Astronomical autumn begins: from today the nights are longer than the days.", "Astronomical spring begins: from today the days are longer than the nights."],
  december_solstice: ["The sun stands lowest at noon: the shortest day and the longest night. Astronomical winter begins.", "The sun stands highest at noon: the longest day and the shortest night. Astronomical summer begins."],
};

const ECLIPSE_KIND: Record<string, string> = {
  "solar partial": "A partial eclipse of the sun.",
  "solar annular": "An annular eclipse of the sun: a ring of sun stays around the moon.",
  "solar total": "A total eclipse of the sun: day turns to dusk.",
  "lunar partial": "A partial eclipse of the moon: the earth's shadow takes a bite out of it.",
  "lunar total": "A total eclipse of the moon: it turns copper-red in the earth's shadow.",
};

const hm = (s?: string) => (s ? s.slice(11, 16) : "");

// The lines under a row, in the reader's language.
function lines(row: any, t: (s: string) => string, locale: string): string[] {
  const d = detailOf(row);
  const south = d.south === true;
  switch (row.kind) {
    case "full_moon": {
      const out = [t(south ? "The moon is opposite the sun: it rises as the sun sets and is lit the whole night." : FULL_MOON_MONTH[+row.starts_at.slice(5, 7) - 1])];
      // The church reckons Easter by a table moon, so the backend computes
      // the day and hangs it on the full moon before it.
      if (d.easter) out.push(`${t("Easter this year:")} ${new Date(d.easter + "T00:00").toLocaleDateString(locale, { weekday: "long", day: "numeric", month: "long" })}.`);
      return out;
    }
    case "new_moon":
      return [t("The moon stands between the earth and the sun and cannot be seen: the darkest nights of the month, best for the stars.")];
    case "march_equinox":
    case "september_equinox":
    case "june_solstice":
    case "december_solstice": {
      const out = [t(SEASON_LINE[row.kind][south ? 1 : 0])];
      if (row.kind.endsWith("equinox")) out.unshift(t("Day and night are nearly equal everywhere on earth."));
      if (d.rise && d.set) out.push(`${t("Here the sun rises at")} ${d.rise} ${t("and sets at")} ${d.set}.`);
      return out;
    }
    case "solar_eclipse":
    case "lunar_eclipse": {
      const solar = row.kind === "solar_eclipse";
      const out = [t(ECLIPSE_KIND[(solar ? "solar " : "lunar ") + d.kind] || "")];
      const pct = Math.round((d.magnitude || 0) * 100);
      if (d.kind !== "total" && pct > 0) out.push(`${t("Greatest at")} ${hm(d.max)}: ${t(solar ? "the moon covers" : "the shadow covers")} ${pct} % ${t(solar ? "of the sun's width" : "of the moon's width")}.`);
      if (d.total_from) out.push(`${t(solar && d.kind === "annular" ? "The ring" : "Totality")}: ${hm(d.total_from)}–${hm(d.total_to)}.`);
      if (d.seen_from) {
        let seen = `${t("Seen from here")} ${hm(d.seen_from)}–${hm(d.seen_to)}`;
        if (row.ends_at && d.seen_to < row.ends_at) seen += `; ${t(solar ? "the sun sets during it" : "the moon sets during it")}`;
        else if (d.seen_from > row.starts_at) seen += `; ${t(solar ? "the sun rises during it" : "the moon rises during it")}`;
        out.push(seen + ".");
      }
      out.push(t(solar ? "Never look at the sun without eclipse glasses: sunglasses do not protect the eyes." : "Safe to watch with the naked eye."));
      return out.filter(Boolean);
    }
  }
  return [];
}

export function SkyCard({ row, me, reload, landed }: { row: any; me: Me; reload: () => void; landed?: boolean }) {
  const { t, lang } = useT();
  const [showComments, setShowComments] = useState(!!landed);
  useEffect(() => { if (landed) setShowComments(true); }, [landed]);
  const over = isOver(row);
  const eclipse = row.kind.endsWith("eclipse");
  return (
    <div className={"card sky" + (over ? " faded" : "")} style={{ borderLeftColor: over ? "var(--parch3)" : "var(--blue)" }} data-entry={"almanac-" + row.id}>
      <div className="head">
        <strong>{skyIcon(row)} {t(row.title || row.kind)}</strong>
        <span className="tag sky">{t("celestial")}</span>
        <span className="when"><When iso={row.starts_at} />{eclipse && row.ends_at ? <> → <When iso={row.ends_at} /></> : null}</span>
      </div>
      <div className="body small">{lines(row, t, localeOf(lang)).map((l, i) => <p key={i}>{l}</p>)}</div>
      <div className="actions">
        <button className="ghost" onClick={() => setShowComments(!showComments)}>💬 {t("Comments")}{row.comments ? ` (${row.comments})` : ""}</button>
        <Share kind="almanac" id={row.id} made={madeOf(row, "almanac")} />
      </div>
      {showComments && <Thread subject="almanac" id={row.id} me={me} onChanged={reload} />}
    </div>
  );
}
