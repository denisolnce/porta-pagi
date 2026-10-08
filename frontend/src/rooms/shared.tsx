import { useCallback, useEffect, useState } from "react";
import { api } from "../api";
import { localeOf, useT } from "../i18n";

// useList: load a resource, expose reload. Rooms are lists with a form on top.
// A null path is a list behind a switch that is off: nothing is fetched and
// the list is empty.
export function useList<T = any>(path: string | null) {
  const [items, setItems] = useState<T[]>([]);
  const [err, setErr] = useState("");
  const reload = useCallback(() => (path ? api<T[]>(path).then(setItems).catch((e) => setErr(String(e.message))) : Promise.resolve(setItems([]))), [path]);
  useEffect(() => { reload(); }, [reload]);
  return { items, err, reload };
}

export function Crest({ crest, color }: { crest: string; color: string }) {
  return <span className="crest" style={{ backgroundColor: color }}>{crest}</span>;
}

// Two shapes reach When, and only one of them is UTC. A date or a time a
// villager typed is local wall clock — "2026-09-07T19:30", no seconds — and is
// already the hour they mean. A stamp SQLite wrote is "2026-09-07 19:30:00" —
// a space and seconds — and is always UTC, because datetime('now') ignores the
// container's TZ. Mark that one with a Z so the browser shifts it; a bare
// string reads as local and shows the village two hours early in summer.
export const parse = (iso: string) =>
  new Date(iso.length <= 10 ? iso + "T00:00" : iso[10] === " " ? iso.replace(" ", "T") + "Z" : iso);

export function When({ iso }: { iso?: string | null }) {
  const { lang } = useT();
  if (!iso) return null;
  const d = parse(iso);
  const opts: Intl.DateTimeFormatOptions = iso.length <= 10 ? { day: "numeric", month: "short" } : { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" };
  return <time dateTime={iso}>{d.toLocaleString(localeOf(lang), opts)}</time>;
}

export function Empty({ text }: { text: string }) {
  return <p className="muted" style={{ fontStyle: "italic" }}>{text}</p>;
}

// stamp: local "YYYY-MM-DDTHH:MM", the same shape every date in the app has,
// so events can be compared as plain strings without a timezone conversion.
export const stamp = (d = new Date()) => {
  const n2 = (v: number) => String(v).padStart(2, "0");
  return `${d.getFullYear()}-${n2(d.getMonth() + 1)}-${n2(d.getDate())}T${n2(d.getHours())}:${n2(d.getMinutes())}`;
};

// isOver: an event is over when its END has passed, not its start. An event
// with no end time has no known length, so it lasts until the end of its day —
// dropping a work party from the list one minute after it begins would hide it
// from exactly the person who is running late. A past event never disappears:
// it stays in its day cell in the calendar, one click away.
// It takes no clock argument on purpose: nothing passed one, and a second
// parameter is what makes `list.some(isOver)` hand in the array index.
export function isOver(ev: { starts_at: string; ends_at?: string | null }): boolean {
  const end = ev.ends_at || ev.starts_at;
  return (end.length <= 10 ? end + "T23:59" : end) < stamp();
}

// A series — weekly, monthly, yearly — lands on its first day's weekday, day of
// the month or date. A monthly 31st lands on a shorter month's last day and a
// yearly 29 February on the 28th, so no month or year goes without it. The
// backend asks the same question (recur.go) before it takes an answer.
const utc = (day: string) => Date.UTC(+day.slice(0, 4), +day.slice(5, 7) - 1, +day.slice(8, 10));
const dayOf = (ms: number) => new Date(ms).toISOString().slice(0, 10);
const lastDay = (y: number, m: number) => new Date(Date.UTC(y, m + 1, 0)).getUTCDate();
export function landsOn(start: string, recur: string, day: string): boolean {
  const s = start.slice(0, 10);
  if (day < s) return false;
  const [y, m, d] = [+day.slice(0, 4), +day.slice(5, 7) - 1, +day.slice(8, 10)];
  const sd = +s.slice(8, 10);
  switch (recur) {
    case "weekly": return Math.round((utc(day) - utc(s)) / 864e5) % 7 === 0;
    case "monthly": return d === Math.min(sd, lastDay(y, m));
    case "yearly": return m === +s.slice(5, 7) - 1 && d === Math.min(sd, lastDay(y, m));
    default: return day === s;
  }
}

// instances lays the events out over the days from..to (YYYY-MM-DD): a one-off
// is itself, a series becomes one copy per day it lands on, moved to that day
// with its hours and its length kept. A copy carries occurs_on, the day its
// answers are for, and series, the row as written — the form edits that. A
// birthday's copy is the whole day; the birth itself stays in series.
export function instances(items: any[], from: string, to: string): any[] {
  const out: any[] = [];
  for (const ev of items) {
    if (!ev.recur) { out.push(ev); continue; }
    const start = ev.starts_at.slice(0, 10);
    const span = ev.ends_at ? parse(ev.ends_at).getTime() - parse(ev.starts_at).getTime() : 0;
    for (let ms = Math.max(utc(from), utc(start)); ms <= utc(to); ms += 864e5) {
      const day = dayOf(ms);
      if (!landsOn(start, ev.recur, day)) continue;
      const starts_at = ev.kind === "birthday" ? day : day + ev.starts_at.slice(10);
      const ends_at = ev.kind !== "birthday" && ev.ends_at ? stamp(new Date(parse(starts_at).getTime() + span)) : null;
      out.push({ ...ev, starts_at, ends_at, occurs_on: day, series: ev });
    }
  }
  return out.sort((a, b) => a.starts_at.localeCompare(b.starts_at));
}

// ahead: the gatherings still to come — events and work parties, a series by
// its next day only. What Home counts and the tavern peek lists; a birthday or
// the sky is not something to come to.
export function ahead(items: any[]): any[] {
  const today = stamp().slice(0, 10);
  const year = dayOf(utc(today) + 400 * 864e5);
  const gatherings = items.filter((ev) => ev.kind !== "birthday");
  const all = instances(gatherings, today, year).filter((ev) => !isOver(ev));
  const seen = new Set<number>();
  return all.filter((ev) => !ev.recur || (!seen.has(ev.id) && seen.add(ev.id)));
}

// answersFor: the answers given for this copy's day — a series is answered one
// day at a time, a one-off's answers carry no day.
export const answersFor = (ev: any): any[] =>
  (JSON.parse(ev.signup_list || "[]") as any[]).filter((x) => (x.occurs_on || "") === (ev.occurs_on || ""));

// house_id is nullable because a contact outlives the house that added it:
// nobody but a steward then matches, which is the intent.
export const canEdit = (me: { id: number; is_steward: number }, row: { house_id: number | null }) => row.house_id === me.id || me.is_steward === 1;
