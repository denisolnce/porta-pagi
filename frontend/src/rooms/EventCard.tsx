import { useState } from "react";
import { Link } from "react-router-dom";
import { api, type Me } from "../api";
import { localeOf, useT } from "../i18n";
import { Crest, When, answersFor, canEdit, isOver, parse } from "./shared";
import { Thread } from "./Thread";
import { DatePicker } from "../DatePicker";

// `alarm` is gone (2026-09-06) but old rows still render.
export const ICON: Record<string, string> = { event: "🔔", work: "🤝", birthday: "🎂", alarm: "🚨" };
export const REPEATS: { value: string; label: string }[] = [
  { value: "", label: "one-off" },
  { value: "weekly", label: "every week" },
  { value: "monthly", label: "every month" },
  { value: "yearly", label: "every year" },
];
const RSVP: { state: string; icon: string; label: string }[] = [
  { state: "yes", icon: "🙋", label: "I am coming" },
  { state: "no", icon: "❌", label: "I cannot come" },
  { state: "maybe", icon: "🤔", label: "Maybe" },
];

// One event, wherever it is shown. Three answers, never blended: a house that
// says no is not the same as a house that has not answered, and only "coming"
// is counted. A sign-up carries no note — "I bring the scythe" belongs in the
// comment thread below, which every house reads and can answer.
//
// `ev` may be one day of a series (instances in shared.tsx): its answers are
// that day's, and the form edits the series as written. A birthday is a date
// for the village to know, not a gathering: no answers, no place, and no
// crest, because it belongs to no house.
export function EventCard({ ev, me, reload, linkToTavern }: { ev: any; me: Me; reload: () => void; linkToTavern?: boolean }) {
  const { t, lang } = useT();
  const src = ev.series || ev;
  const birthday = ev.kind === "birthday";
  const [editing, setEditing] = useState(false);
  const [f, setF] = useState({ title: src.title, kind: src.kind, starts_at: src.starts_at, ends_at: src.ends_at || "", place: src.place || "", notes: src.notes || "", recur: src.recur || "" });
  const [showComments, setShowComments] = useState(false);

  const list = answersFor(ev);
  const mine = (list.find((x) => x.house_id === me.id)?.state as string | undefined) ?? null;
  const yes = list.filter((x) => x.state === "yes" && !x.stale).length;
  const by = (s: string) => list.filter((x) => x.state === s);
  const on = ev.occurs_on || "";
  const answer = (state: string) => api(`/events/${ev.id}/signup`, { method: "POST", body: { state, occurs_on: on || undefined } }).then(reload);
  const clear = () => api(`/events/${ev.id}/signup${on ? "?on=" + on : ""}`, { method: "DELETE" }).then(reload);
  const save = async () => {
    const body = birthday ? { title: f.title, starts_at: f.starts_at, notes: f.notes } : f;
    await api(`/events/${ev.id}`, { method: "PUT", body });
    setEditing(false); reload();
  };

  const anyStale = list.some((x) => x.stale);
  // Faded, kind colour off the edge, no answer buttons — the thread and the
  // answers given stay. CLAUDE.md § An answer is not a headcount says why.
  const over = isOver(ev);
  // A render function, not a component — see Market.tsx. Lowercase on purpose.
  const group = ({ state, icon }: { state: string; icon: string }) => {
    const g = by(state);
    if (!g.length) return null;
    return (
      <div key={state} className="small signers">
        <span className="state" title={t(RSVP.find((r) => r.state === state)!.label)}>{icon}</span>
        <span className="who">{g.map((sgn, i) => (
          <span key={sgn.house_id} className={sgn.stale ? "stale" : ""}>
            {i > 0 ? ", " : ""}{sgn.crest} {sgn.name}
          </span>
        ))}</span>
      </div>
    );
  };
  // The birth in words: the year, and the hour when somebody knows it.
  const born = () => {
    const d = parse(src.starts_at);
    const day = d.toLocaleDateString(localeOf(lang), { day: "numeric", month: "long", year: "numeric" });
    return src.starts_at.length > 10 ? `${day} · ${src.starts_at.slice(11, 16)}` : day;
  };
  const edge = over ? "var(--parch3)" : ev.kind === "alarm" ? "var(--red)" : ev.kind === "work" ? "var(--green)" : birthday ? "var(--plum)" : "var(--brass)";

  return (
    <div className={"card" + (over ? " faded" : "")} style={{ borderLeftColor: edge }}>
      {editing ? (
        <form className="inline" onSubmit={(e) => { e.preventDefault(); save(); }}>
          <div className="row">
            <label>{birthday ? t("Name") : t("Title")}<input value={f.title} onChange={(e) => setF({ ...f, title: e.target.value })} required maxLength={120} /></label>
            {!birthday && <label>{t("Kind")}<select value={f.kind} onChange={(e) => setF({ ...f, kind: e.target.value })}>{["event", "work"].map((k) => <option key={k} value={k}>{ICON[k]} {t(k)}</option>)}</select></label>}
          </div>
          {birthday ? (
            <div className="row">
              <label>{t("Born")}<DatePicker time timeOptional anyMinute years required value={f.starts_at} onChange={(v) => setF({ ...f, starts_at: v })} /></label>
            </div>
          ) : (
            <div className="row">
              <label>{t("Starts")}<DatePicker time required value={f.starts_at} onChange={(v) => setF({ ...f, starts_at: v, ends_at: f.ends_at && f.ends_at < v ? v : f.ends_at })} /></label>
              <label>{t("Ends")}<DatePicker time min={f.starts_at} value={f.ends_at} onChange={(v) => setF({ ...f, ends_at: v })} defaultTime="17:00" /></label>
              <label>{t("Repeats")}<select value={f.recur} onChange={(e) => setF({ ...f, recur: e.target.value })}>{REPEATS.map((r) => <option key={r.value} value={r.value}>{t(r.label)}</option>)}</select></label>
              <label>{t("Place")}<input value={f.place} onChange={(e) => setF({ ...f, place: e.target.value })} maxLength={120} /></label>
            </div>
          )}
          <label>{t("Notes")}<textarea value={f.notes} onChange={(e) => setF({ ...f, notes: e.target.value })} maxLength={2000} /></label>
          {src.recur && !birthday && <p className="small muted">{t("A change here applies to every date of it.")}</p>}
          <div className="submit"><button type="button" className="ghost" onClick={() => setEditing(false)}>✕</button><button className="primary" type="submit">{t("Save")}</button></div>
        </form>
      ) : (
        <>
          <div className="head">
            {!birthday && <Crest crest={ev.house_crest} color={ev.house_color} />}
            <strong>{ICON[ev.kind]} {linkToTavern ? <Link to={`/tavern?day=${ev.starts_at.slice(0, 10)}`} className="plain">{ev.title}</Link> : ev.title}</strong>
            {ev.kind !== "event" && <span className={"tag " + ev.kind}>{t(ev.kind)}</span>}
            {ev.recur && !birthday && <span className="tag">↻ {t(REPEATS.find((r) => r.value === ev.recur)?.label || "")}</span>}
            {ev.project_id && <Link to={`/projects/${ev.project_id}`} className="tag project-chip">📋 {ev.project_title}{ev.task_title ? ` · ${ev.task_title}` : ""}</Link>}
            <span className="when"><When iso={ev.starts_at} />{ev.ends_at ? <> → <When iso={ev.ends_at} /></> : null}</span>
          </div>
          {birthday && <div className="small">{t("Born")} {born()}</div>}
          {(ev.place || ev.notes) && <div className="body small">{ev.place && <span className="tag place">📍 {ev.place}</span>}{ev.place && ev.notes ? " " : ""}{ev.notes}</div>}
          {ev.edited_by_name && <div className="small muted">✎ {t("last edited by")} {ev.edited_by_name}</div>}
          {anyStale && <div className="small stale-note">⧗ {t("The time moved. Struck answers were given for the old one.")}</div>}
        </>
      )}

      {!birthday && RSVP.map(group)}
      <div className="actions">
        {!over && !birthday && RSVP.map((r) => (
          <button key={r.state} className={mine === r.state ? "primary" : "lesser"} onClick={() => (mine === r.state ? clear() : answer(r.state))} title={mine === r.state ? t("tap again to take it back") : ""}>
            {r.icon} {t(r.label)}{r.state === "yes" && yes > 0 ? ` (${yes})` : ""}
          </button>
        ))}
        <button className="ghost" onClick={() => setShowComments(!showComments)}>💬 {t("Comments")}{ev.comments ? ` (${ev.comments})` : ""}</button>
        {!editing && <button className="ghost" onClick={() => setEditing(true)}>✎ {t("Edit")}</button>}
        {canEdit(me, ev) && <button className="ghost" onClick={() => confirm(ev.recur && !birthday ? `${ev.title} — ${t("every date of it")}?` : ev.title + "?") && api(`/events/${ev.id}`, { method: "DELETE" }).then(reload)}>🗑 {t("Delete")}</button>}
      </div>
      {showComments && <Thread subject="event" id={ev.id} me={me} onChanged={reload} />}
    </div>
  );
}
