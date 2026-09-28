import { useCallback, useEffect, useState } from "react";
import { Navigate } from "react-router-dom";
import { api, type House, type Me } from "../api";
import { localeOf, useT } from "../i18n";
import { useFeature } from "../features";
import { DatePicker } from "../DatePicker";
import { Crest, Empty, When, parse } from "./shared";

// Duty roster, behind the `roster` switch: the village's recurring chores and
// the turns houses took. One of the two places the portal counts, and only
// because a village switched it on. A steward writes the duties down; a house
// writes its own turns and nobody else's. The table is done turns per house
// per duty over the last twelve months: houses in the village's order, never
// by count, and no sum across duties — a pump check is not a mowing. Nothing
// assigns a house to a date, nothing pushes, and a turn that did not happen
// is not "missed": after its day only its own house sees it.

type Turn = { id: number; duty_id: number; house_id: number | null; on_date: string; state: "planned" | "done"; note: string; house_name?: string; house_crest?: string; house_color?: string };
type Duty = { id: number; title: string; notes: string; every_days: number | null; retired_at: string | null; last_done: string | null; next_due: string | null; turns: Turn[] };
type RosterData = { today: string; since: string; duties: Duty[]; counts: { duty_id: number; house_id: number; n: number }[] };

export function Roster({ me, houses }: { me: Me; houses: House[] }) {
  const { t, lang } = useT();
  const on = useFeature("roster");
  const [data, setData] = useState<RosterData | null>(null);
  const [err, setErr] = useState("");
  const [open, setOpen] = useState(false);
  const [f, setF] = useState({ title: "", every_days: "", notes: "" });
  const steward = me.is_steward === 1;
  const reload = useCallback(() => api<RosterData>("/roster").then(setData).catch((e) => setErr(String(e.message))), []);
  useEffect(() => { if (on) reload(); }, [on, reload]);
  if (!on) return <Navigate to="/" replace />;

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    await api("/roster/duties", { method: "POST", body: { title: f.title, notes: f.notes, every_days: f.every_days ? Number(f.every_days) : null } }).catch((e) => setErr(e.message));
    setF({ title: "", every_days: "", notes: "" }); setOpen(false); reload();
  };

  const duties = data?.duties || [];
  const live = duties.filter((d) => !d.retired_at), retired = duties.filter((d) => d.retired_at);
  // Land does not take turns: a common place is not an account.
  const rows = houses.filter((h) => h.kind !== "common");
  const count = (house: number, duty: number) => data?.counts.find((c) => c.house_id === house && c.duty_id === duty)?.n || 0;

  return (
    <div className="parchment">
      <h2>🧹 {t("Duty roster")} <span className="sub">{t("chores and whose turn it was")}</span>
        {steward && <button style={{ marginLeft: "auto" }} onClick={() => setOpen(!open)}>+ {t("New duty")}</button>}
      </h2>
      {err && <p className="muted">{err}</p>}
      {open && (
        <form className="inline" onSubmit={create}>
          <div className="row">
            <label>{t("Title")}<input value={f.title} onChange={(e) => setF({ ...f, title: e.target.value })} required maxLength={120} placeholder={t("mow the common ground")} /></label>
            <label>{t("Every how many days")}<input type="number" min={1} max={366} value={f.every_days} onChange={(e) => setF({ ...f, every_days: e.target.value })} placeholder={t("empty: when needed")} /></label>
          </div>
          <label>{t("Notes")}<textarea value={f.notes} onChange={(e) => setF({ ...f, notes: e.target.value })} maxLength={2000} /></label>
          <div className="submit"><button className="primary" type="submit">{t("Create")}</button></div>
        </form>
      )}
      {data && duties.length === 0 && <Empty text={steward ? t("Write down the duties the village agreed on — the mowing, the snow, the pump.") : t("No duties yet. A steward writes down the ones the village agreed on.")} />}
      {data && live.map((d) => <DutyCard key={d.id} d={d} me={me} today={data.today} reload={reload} onErr={setErr} />)}

      {data && live.length > 0 && rows.length > 0 && (
        <>
          <h3 style={{ marginTop: "1rem" }}>{t("Turns in the last twelve months")}</h3>
          <div className="roster-table">
            <table>
              <thead>
                <tr><th />{live.map((d) => <th key={d.id} scope="col">{d.title}</th>)}</tr>
              </thead>
              <tbody>
                {rows.map((h) => (
                  <tr key={h.id}>
                    <th scope="row"><Crest crest={h.crest} color={h.color} /> <span className="name">{h.name}</span></th>
                    {live.map((d) => { const n = count(h.id, d.id); return <td key={d.id} className={n ? "" : "muted"}>{n}</td>; })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {/* The window's start carries its year: "28 Sep" alone reads as today. */}
          <p className="small muted">{t("Done turns since")} {parse(data.since).toLocaleDateString(localeOf(lang), { day: "numeric", month: "short", year: "numeric" })} · {t("A turn taken ahead counts once it is done.")}</p>
        </>
      )}

      {retired.length > 0 && (
        <details style={{ marginTop: "1rem" }}>
          <summary className="small">{t("Retired duties")} ({retired.length})</summary>
          {retired.map((d) => <DutyCard key={d.id} d={d} me={me} today={data!.today} reload={reload} onErr={setErr} />)}
        </details>
      )}
    </div>
  );
}

function DutyCard({ d, me, today, reload, onErr }: { d: Duty; me: Me; today: string; reload: () => void; onErr: (s: string) => void }) {
  const { t } = useT();
  const steward = me.is_steward === 1;
  const [form, setForm] = useState(false);
  const [day, setDay] = useState(today);
  const [note, setNote] = useState("");
  const [edit, setEdit] = useState(false);
  const [ef, setEf] = useState({ title: d.title, every_days: d.every_days ? String(d.every_days) : "", notes: d.notes });
  const call = (p: Promise<unknown>) => p.then(reload).catch((e) => onErr(String(e.message)));

  const plans = d.turns.filter((x) => x.state === "planned");
  const log = d.turns.filter((x) => x.state === "done");
  const mineToday = d.turns.some((x) => x.house_id === me.id && x.on_date === today);
  const lastBy = log.find((x) => x.on_date === d.last_done);
  const due = !d.retired_at && d.next_due !== null && d.next_due <= today;
  const ahead = day > today;

  const add = (e: React.FormEvent) => {
    e.preventDefault();
    call(api(`/roster/duties/${d.id}/turns`, { method: "POST", body: { on_date: day, done: !ahead, note } }));
    setForm(false); setNote(""); setDay(today);
  };
  const saveEdit = (e: React.FormEvent) => {
    e.preventDefault();
    call(api(`/roster/duties/${d.id}`, { method: "PUT", body: { title: ef.title, notes: ef.notes, every_days: ef.every_days ? Number(ef.every_days) : null } }));
    setEdit(false);
  };
  const who = (x: Turn) => x.house_id === null
    ? <span className="muted">{t("a house that has left")}</span>
    : <><Crest crest={x.house_crest || ""} color={x.house_color || ""} /> {x.house_name}</>;

  return (
    <div className={"card duty" + (d.retired_at ? " faded" : "")} style={{ borderLeftColor: due ? "var(--brass)" : undefined }}>
      <div className="head">
        <strong>{d.title}</strong>
        <span className="tag">{d.every_days ? `${t("every")} ${d.every_days} ${t("days")}` : t("when needed")}</span>
        {due && <span className="tag taken">{t("due")}</span>}
      </div>
      {d.notes && <div className="body small">{d.notes}</div>}
      <div className="small">
        {d.last_done ? <>{t("Last done")} <When iso={d.last_done} /> · {lastBy ? who(lastBy) : null}</> : t("Not done yet")}
        {d.next_due && !d.retired_at && <> · {t("next around")} <When iso={d.next_due} /></>}
      </div>

      {plans.map((x) => {
        const mine = x.house_id === me.id;
        const passed = x.on_date < today;
        return (
          <div key={x.id} className="turn small">
            🙋 {who(x)} · <When iso={x.on_date} />{x.note && <> · <em>{x.note}</em></>}
            {passed && <span className="muted"> · {t("did it happen?")}</span>}
            {mine && (
              <span className="actions">
                {x.on_date <= today && !d.retired_at && <button className="primary" onClick={() => call(api(`/roster/turns/${x.id}`, { method: "PUT", body: { done: true } }))}>✓ {t("Done")}</button>}
                <button className="ghost" onClick={() => call(api(`/roster/turns/${x.id}`, { method: "DELETE" }))}>{passed ? t("It did not happen") : t("Let it go")}</button>
              </span>
            )}
          </div>
        );
      })}

      {!d.retired_at && !form && (
        <div className="actions">
          {!mineToday && <button className="primary" onClick={() => call(api(`/roster/duties/${d.id}/turns`, { method: "POST", body: { done: true } }))}>✓ {t("Done today")}</button>}
          <button onClick={() => setForm(true)}>🙋 {t("Another day…")}</button>
        </div>
      )}
      {form && (
        <form className="inline" onSubmit={add}>
          <div className="row">
            <label>{t("Day")}<DatePicker required value={day} onChange={setDay} /></label>
            <label>{t("Note")}<input value={note} onChange={(e) => setNote(e.target.value)} maxLength={300} placeholder={t("a note, optional")} /></label>
          </div>
          {/* The day decides what the turn is: ahead is a turn taken, today or
              before is one done. The button says which. */}
          <div className="submit">
            <button type="button" className="ghost" onClick={() => setForm(false)}>✕</button>
            <button className="primary" type="submit">{ahead ? "🙋 " + t("I'll do it that day") : "✓ " + t("We did it that day")}</button>
          </div>
        </form>
      )}

      {log.length > 0 && (
        <details className="small">
          <summary>{t("Log")} ({log.length})</summary>
          {log.map((x) => (
            <div key={x.id} className="turn">
              ✓ <When iso={x.on_date} /> · {who(x)}{x.note && <> · <em>{x.note}</em></>}
              {x.house_id === me.id && <button className="ghost" title={t("Take this turn back")} onClick={() => confirm(t("Take this turn back?")) && call(api(`/roster/turns/${x.id}`, { method: "DELETE" }))}>🗑</button>}
            </div>
          ))}
        </details>
      )}

      {steward && edit && (
        <form className="inline" onSubmit={saveEdit}>
          <div className="row">
            <label>{t("Title")}<input value={ef.title} onChange={(e) => setEf({ ...ef, title: e.target.value })} required maxLength={120} /></label>
            <label>{t("Every how many days")}<input type="number" min={1} max={366} value={ef.every_days} onChange={(e) => setEf({ ...ef, every_days: e.target.value })} placeholder={t("empty: when needed")} /></label>
          </div>
          <label>{t("Notes")}<textarea value={ef.notes} onChange={(e) => setEf({ ...ef, notes: e.target.value })} maxLength={2000} /></label>
          <div className="submit"><button className="primary" type="submit">{t("Save")}</button></div>
        </form>
      )}
      {steward && (
        <div className="actions">
          <button onClick={() => setEdit(!edit)}>{t("Edit")}</button>
          {d.retired_at
            ? <button onClick={() => call(api(`/roster/duties/${d.id}`, { method: "PUT", body: { retired: false } }))}>{t("Bring back")}</button>
            : <button onClick={() => call(api(`/roster/duties/${d.id}`, { method: "PUT", body: { retired: true } }))}>{t("Retire")}</button>}
          {/* A duty somebody did is history: it can be retired, never deleted. */}
          {!d.last_done && <button className="ghost" onClick={() => confirm(d.title + "?") && call(api(`/roster/duties/${d.id}`, { method: "DELETE" }))}>🗑</button>}
        </div>
      )}
    </div>
  );
}
