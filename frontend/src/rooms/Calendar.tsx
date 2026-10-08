import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { api, type House, type Me } from "../api";
import { localeOf, useT } from "../i18n";
import { ahead as aheadOf, instances, isOver, useList, When } from "./shared";
import { EventCard, ICON, REPEATS } from "./EventCard";
import { SkyCard, skyIcon } from "./Almanac";
import { useFeature } from "../features";
import { DatePicker } from "../DatePicker";

const iso = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;

const blank = { title: "", kind: "event", starts_at: "", ends_at: "", place: "", notes: "", project_id: "", task_id: "", recur: "" };

// The village calendar: a month grid with the events sitting in their day, then
// the chosen day (or everything ahead) as cards you can sign up to. A series
// sits on every day it lands on; the sky — moons, equinoxes, solstices,
// eclipses — sits beside the events, written by the portal and only talked
// about, never answered. The sky is a switch (celestial): off, it is not
// fetched, and no mark or card of it is drawn.
export function Calendar({ me, houses }: { me: Me; houses: House[] }) {
  const { t, lang } = useT();
  const { items, reload } = useList("/events");
  const sky = useList(useFeature("celestial") ? "/almanac" : null);
  const [f, setF] = useState(blank);
  const birthday = f.kind === "birthday";
  const [open, setOpen] = useState(false);
  // Projects for the "belongs to" select; ?project=ID (from a project page)
  // opens the form with that project chosen.
  const projects = useList("/projects");
  const [params] = useSearchParams();
  const [tasks, setTasks] = useState<any[]>([]);
  useEffect(() => {
    if (!f.project_id) { setTasks([]); return; }
    api<any>(`/projects/${f.project_id}`).then((p) => setTasks((p.tasks || []).filter((x: any) => x.state === "open"))).catch(() => setTasks([]));
  }, [f.project_id]);
  useEffect(() => {
    const pid = params.get("project");
    if (pid && !open) { const d = defaultDay(); setF((x) => ({ ...x, project_id: pid, kind: "work", starts_at: d + "T09:00", ends_at: d + "T17:00" })); setOpen(true); }
    const dd = params.get("day");
    if (dd && /^\d{4}-\d{2}-\d{2}$/.test(dd)) { setDay(dd); setCursor(new Date(dd + "T00:00")); }
  }, [params]); // eslint-disable-line react-hooks/exhaustive-deps
  const [cursor, setCursor] = useState(() => new Date());
  const [day, setDay] = useState<string | null>(null);
  const locale = localeOf(lang);

  // The form opens already filled: the picked day, else today when the shown
  // month is this one, else the 1st of the shown month — 09:00 to 17:00. A
  // villager then changes the day, or nothing. Moving the start date drags the
  // end date along until the end is edited by hand.
  const [endTouched, setEndTouched] = useState(false);
  const defaultDay = () => {
    if (day) return day;
    const now = new Date();
    return cursor.getFullYear() === now.getFullYear() && cursor.getMonth() === now.getMonth() ? iso(now) : iso(new Date(cursor.getFullYear(), cursor.getMonth(), 1));
  };
  const openForm = () => {
    if (!open) {
      const d = defaultDay();
      setF({ ...f, starts_at: d + "T09:00", ends_at: d + "T17:00" });
      setEndTouched(false);
    }
    setOpen(!open);
  };
  const set = (k: string) => (e: React.ChangeEvent<any>) => {
    const v = e.target.value;
    // A birthday's date is the birth, not a day of this month at nine: the
    // field starts empty, and leaving the kind brings the usual hours back.
    if (k === "kind" && (v === "birthday") !== birthday) {
      const d = defaultDay();
      setF(v === "birthday" ? { ...f, kind: v, starts_at: "", ends_at: "", recur: "", project_id: "", task_id: "" } : { ...f, kind: v, starts_at: d + "T09:00", ends_at: d + "T17:00" });
      setEndTouched(false);
      return;
    }
    if (k === "starts_at" && !endTouched && v.length >= 10) {
      setF({ ...f, starts_at: v, ends_at: v.slice(0, 10) + (f.ends_at.slice(10) || "T17:00") });
      return;
    }
    if (k === "ends_at") setEndTouched(true);
    // A start dragged past a touched end would leave the end behind it.
    if (k === "starts_at" && f.ends_at && f.ends_at < v) {
      setF({ ...f, starts_at: v, ends_at: v });
      return;
    }
    setF({ ...f, [k]: v });
  };

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    await api("/events", { method: "POST", body: { ...f, project_id: f.project_id ? Number(f.project_id) : undefined, task_id: f.task_id ? Number(f.task_id) : undefined } });
    setF(blank);
    setOpen(false);
    reload();
  };

  // Monday-first grid, six rows so the layout never jumps between months.
  const cells = useMemo(() => {
    const first = new Date(cursor.getFullYear(), cursor.getMonth(), 1);
    const start = new Date(first);
    start.setDate(1 - ((first.getDay() + 6) % 7));
    return Array.from({ length: 42 }, (_, i) => {
      const d = new Date(start);
      d.setDate(start.getDate() + i);
      return d;
    });
  }, [cursor]);

  // What the grid shows: the events, a series as one copy per day it lands
  // on within the six weeks, and the sky beside them, marked as such.
  const shownItems = useMemo(() => {
    const evs = instances(items, iso(cells[0]), iso(cells[41]));
    const rows = sky.items.map((x: any) => ({ ...x, sky: true }));
    return [...evs, ...rows].sort((a, b) => a.starts_at.localeCompare(b.starts_at));
  }, [items, sky.items, cells]);

  // An event occupies every day from its start to its end.
  const byDay = useMemo(() => {
    const m: Record<string, any[]> = {};
    for (const ev of shownItems) {
      const from = new Date(ev.starts_at.slice(0, 10) + "T00:00");
      const to = new Date((ev.ends_at || ev.starts_at).slice(0, 10) + "T00:00");
      for (const d = new Date(from); d <= to; d.setDate(d.getDate() + 1)) (m[iso(d)] ||= []).push(ev);
    }
    return m;
  }, [shownItems]);

  const today = iso(new Date());
  // Without a picked day the list holds THIS MONTH's events that are still to
  // come — not everything ahead forever. The grid above already says which
  // month you are looking at, so the list can never grow past a month's worth,
  // and paging with ‹ › pages the list with it. What has passed leaves the
  // list and stays in its day cell.
  const monthKey = `${cursor.getFullYear()}-${String(cursor.getMonth() + 1).padStart(2, "0")}`;
  const inMonth = (ev: any) => ev.starts_at.slice(0, 7) <= monthKey && (ev.ends_at || ev.starts_at).slice(0, 7) >= monthKey;
  const shown = day ? byDay[day] || [] : shownItems.filter((ev) => inMonth(ev) && !isOver(ev));
  // Leaving the month drops the picked day with it — the list below belongs to
  // the month on screen, so a day from the month you just left must not hold it.
  const page = (by: number, to?: Date) => { setDay(null); setCursor(to || new Date(cursor.getFullYear(), cursor.getMonth() + by, 1)); };
  // Month-scoping must not hide that anything exists outside the month — Home
  // counts every event ahead, so the room has to account for the difference.
  // Whenever something ahead sits outside the shown month, a tail line says how
  // many and jumps to the closest one. Ahead is what Home counts: gatherings,
  // a series by its next day — never the sky or a birthday, which are always
  // somewhere ahead.
  const ahead = aheadOf(items);
  // A series stands ahead by its next day, but sits in this month whenever
  // one of its days still to come does: the Mondays of a weekly yoga are in
  // the list, and only a series whose days here have all passed points on.
  const inShown = (ev: any) => (ev.recur ? shownItems.some((x) => !x.sky && x.id === ev.id && inMonth(x) && !isOver(x)) : inMonth(ev));
  const outside = ahead.filter((ev) => !inShown(ev));
  const pointer = day ? null : outside[0] || null;
  const backwards = pointer ? pointer.starts_at.slice(0, 7) < monthKey : false;
  const weekdays = useMemo(() => {
    const base = new Date(2026, 0, 5); // a Monday
    return Array.from({ length: 7 }, (_, i) => {
      const d = new Date(base);
      d.setDate(5 + i);
      return d.toLocaleDateString(locale, { weekday: "short" }).slice(0, 2);
    });
  }, [locale]);

  return (
    <div className="parchment">
      <h2>🔔 {t("The village calendar")} <span className="sub">{t("calendar and work bees")}</span>
        <button style={{ marginLeft: "auto" }} onClick={openForm}>+ {t("Add an event")}</button>
      </h2>

      {open && (
        <form className="inline" onSubmit={save}>
          <div className="row">
            <label>{birthday ? t("Name") : t("Title")}<input value={f.title} onChange={set("title")} required maxLength={120} /></label>
            <label>{t("Kind")}<select value={f.kind} onChange={set("kind")}><option value="event">🔔 {t("event")}</option><option value="work">🤝 {t("work")}</option><option value="birthday">🎂 {t("birthday")}</option></select></label>
          </div>
          {birthday ? (<>
            <div className="row">
              <label>{t("Born")}<DatePicker time timeOptional anyMinute years required value={f.starts_at} onChange={(v) => setF({ ...f, starts_at: v })} placeholder={t("date of birth")} /></label>
            </div>
            <p className="small muted">{t("A date for the village to know, not a party: every house sees the date and the year, and the hour if you give it. Write someone's down only with their yes. It rings every house on the morning of the day.")}</p>
          </>) : (<>
          <div className="row">
            <label>{t("Starts")}<DatePicker time required value={f.starts_at} onChange={(v) => set("starts_at")({ target: { value: v } } as any)} /></label>
            <label>{t("Ends")}<DatePicker time min={f.starts_at} value={f.ends_at} onChange={(v) => set("ends_at")({ target: { value: v } } as any)} defaultTime="17:00" /></label>
            {/* The picker sits OUTSIDE the label on purpose. A label activates
                its first control, so a tap meant for the dropdown reached the
                text input instead, opened the keyboard and moved the page under
                the finger — the same trap the DatePicker documents. The select
                also shows the chosen house instead of snapping back to "📍…",
                so picking the same place twice is not a silent no-op. */}
            <span className="place-pick">
              <label>{t("Place")}
                <input value={f.place} onChange={set("place")} maxLength={120} placeholder={t("anywhere, or pick below")} />
              </label>
              <select aria-label={t("Place")} value={houses.some((h) => h.name === f.place) ? f.place : ""} onChange={(e) => setF({ ...f, place: e.target.value })}>
                <option value="">📍…</option>
                {houses.map((h) => <option key={h.id} value={h.name}>{h.crest} {h.name}</option>)}
              </select>
            </span>
            <label>{t("Repeats")}<select value={f.recur} onChange={set("recur")}>{REPEATS.map((r) => <option key={r.value} value={r.value}>{t(r.label)}</option>)}</select></label>
          </div>
          {/* Every project not finished, planned ones included: the work
              party that starts a project is exactly the event it needs. */}
          {projects.items.filter((p) => p.state !== "done").length > 0 && (
            <div className="row">
              <label>📋 {t("Belongs to a project")}<select value={f.project_id} onChange={(e) => setF({ ...f, project_id: e.target.value, task_id: "" })}><option value="">—</option>{projects.items.filter((p) => p.state !== "done").map((p) => <option key={p.id} value={p.id}>{p.title}</option>)}</select></label>
              {f.project_id && tasks.length > 0 && <label>{t("Task")}<select value={f.task_id} onChange={set("task_id")}><option value="">—</option>{tasks.map((x) => <option key={x.id} value={x.id}>{x.title}</option>)}</select></label>}
            </div>
          )}
          </>)}
          <label>{t("Notes")}<textarea value={f.notes} onChange={set("notes")} maxLength={2000} /></label>
          <div className="submit"><button className="primary" type="submit">{t("Save")}</button></div>
        </form>
      )}

      <div className="cal">
        <div className="cal-head">
          <button className="ghost" aria-label="‹" onClick={() => page(-1)}>‹</button>
          <strong>{cursor.toLocaleDateString(locale, { month: "long", year: "numeric" })}</strong>
          <button className="ghost" aria-label="›" onClick={() => page(1)}>›</button>
        </div>
        <div className="cal-grid">
          {weekdays.map((w, i) => <div key={"w" + i} className="cal-wd">{w}</div>)}
          {cells.map((d) => {
            const key = iso(d);
            const evs = byDay[key] || [];
            const other = d.getMonth() !== cursor.getMonth();
            return (
              <button key={key} className={"cal-day" + (other ? " other" : "") + (key === today ? " today" : "") + (key === day ? " picked" : "")}
                onClick={() => setDay(key === day ? null : key)}>
                <span className="n">{d.getDate()}</span>
                <span className={"marks m" + Math.min(evs.length, 4)}>{evs.slice(0, 4).map((e, i) => <span key={i}>{e.sky ? skyIcon(e) : ICON[e.kind]}</span>)}{evs.length > 4 ? <span className="more">+{evs.length - 4}</span> : null}</span>
              </button>
            );
          })}
        </div>
      </div>

      {day && <p className="small">{new Date(day + "T00:00").toLocaleDateString(locale, { weekday: "long", day: "numeric", month: "long" })} · <button className="ghost" onClick={() => setDay(null)}>{t("Show the whole month")}</button></p>}
      {shown.length === 0 && !pointer && <p className="muted" style={{ fontStyle: "italic" }}>{day ? t("Nothing on this day.") : t("Nothing planned. Ring the bell — add an event or call a work party, and houses sign up here.")}</p>}
      {pointer && (
        <p className="small">
          {shown.length === 0 ? t("Nothing left this month.") + " " : ""}
          {outside.length > 1 ? `${outside.length} ${t("outside this month")} · ` : ""}
          <button className="lesser" onClick={() => page(0, new Date(pointer.starts_at.slice(0, 10) + "T00:00"))}>
            {backwards ? t("Nearest") : t("Next")}: {ICON[pointer.kind]} {pointer.title} · <When iso={pointer.starts_at} /> {backwards ? "‹" : "›"}
          </button>
        </p>
      )}
      {shown.map((ev) => ev.sky
        ? <SkyCard key={"sky" + ev.id + "-" + (day || "")} row={ev} me={me} reload={sky.reload} />
        : <EventCard key={ev.id + "-" + (ev.occurs_on || "") + "-" + (day || "")} ev={ev} me={me} reload={reload} />)}
    </div>
  );
}
