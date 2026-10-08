import { useEffect, useState } from "react";
import { api } from "./api";
import { localeOf, useT } from "./i18n";
import { Empty } from "./rooms/shared";

// The panel at the foot of Home that answers "is the thing I asked for in
// yet", in two tabs (owner's pick 2026-10-08). Portal News, first and chosen
// by default: a note per change a villager would notice, in the village's
// languages, shipped inside the binary and rung once when the roll brings it
// (news.go). Development: what changed in the last 90 days, read off the git
// log at build time (scripts/changelog.mjs writes public/changelog.json) —
// the builder's own commit subjects, in English, grouped by day.
type Entry = { h: string; d: string; s: string };
type Log = { built: string | null; entries: Entry[] };

type Note = { id: string; on: string; texts: Record<string, { title: string; body: string }> };

export function Changelog() {
  const { t } = useT();
  const [tab, setTab] = useState<"news" | "dev">("news");
  return (
    <div className="parchment" id="news">
      <h2>🔨 {t("Changelog")}</h2>
      <div className="tabs">
        <button className={tab === "news" ? "active" : ""} onClick={() => setTab("news")}>📰 {t("Portal News")}</button>
        <button className={tab === "dev" ? "active" : ""} onClick={() => setTab("dev")}>🛠 {t("Development")}</button>
      </div>
      {tab === "news" ? <News /> : <Development />}
    </div>
  );
}

// A note in the reader's language, else the village's first that has it.
function News() {
  const { t, lang, languages } = useT();
  const [notes, setNotes] = useState<Note[] | null>(null);
  const [err, setErr] = useState(false);
  // Opened by a push (#/?news), the panel comes into view once the notes are
  // in: before that, Home's own lists are still pushing it down.
  useEffect(() => {
    api<Note[]>("/news").then((n) => {
      setNotes(n);
      if (location.hash.includes("?news")) requestAnimationFrame(() => document.getElementById("news")?.scrollIntoView());
    }).catch(() => setErr(true));
  }, []);
  const pick = (n: Note) => [lang, ...languages, "en"].map((l) => n.texts[l]).find((x) => x?.title);
  const day = (d: string) => new Date(d + "T00:00").toLocaleDateString(localeOf(lang), { day: "numeric", month: "long", year: "numeric" });
  return (<>
    {err && <Empty text={t("Could not load the news.")} />}
    {notes && notes.length === 0 && <Empty text={t("No news yet.")} />}
    {notes?.map((n) => {
      const x = pick(n);
      return x ? (
        <section key={n.id} className="cl-day">
          <h3>{x.title} <span className="small muted">· {day(n.on)}</span></h3>
          <p>{x.body}</p>
        </section>
      ) : null;
    })}
  </>);
}

function Development() {
  const { t, lang } = useT();
  const [log, setLog] = useState<Log | null>(null);
  const [err, setErr] = useState(false);
  useEffect(() => {
    fetch(`${import.meta.env.BASE_URL}changelog.json`, { cache: "no-cache" })
      .then((r) => (r.ok ? r.json() : Promise.reject(r.status)))
      .then(setLog)
      .catch(() => setErr(true));
  }, []);
  const day = (d: string) => new Date(d + "T00:00").toLocaleDateString(localeOf(lang), { weekday: "long", day: "numeric", month: "long", year: "numeric" });
  // Grouped by day whatever the log's order: a branch merged later brings
  // commits of a day already listed, and a day must show once.
  const byDay = new Map<string, Entry[]>();
  for (const e of log?.entries ?? []) byDay.set(e.d, [...(byDay.get(e.d) ?? []), e]);
  const days = [...byDay.entries()].sort((a, b) => b[0].localeCompare(a[0]));
  return (
    <>
      <p className="small muted">{t("what changed in the last 90 days, newest first")} · {t("The builder's notes, in English.")}{log?.built && <> · {t("Built from the change of")} {day(log.built)}.</>}</p>
      {err && <Empty text={t("Could not load the changelog.")} />}
      {log && log.entries.length === 0 && <Empty text={t("Nothing in the last 90 days.")} />}
      {days.map(([d, es]) => (
        <section key={d} className="cl-day">
          <h3>{day(d)}</h3>
          <ul>{es.map((e) => <li key={e.h}>{e.s}</li>)}</ul>
        </section>
      ))}
    </>
  );
}
