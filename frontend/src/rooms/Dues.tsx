import { useState } from "react";
import { api, type House, type Me } from "../api";
import { DatePicker } from "../DatePicker";
import { localeOf, useT } from "../i18n";
import { Crest, Empty, When, parse, stamp, useList } from "./shared";

// Dues: who paid the water this quarter. Behind the `dues` switch — one of
// the two rooms where the portal counts, because a village switched it on.
// A steward writes a due down and names the house that collects it; that
// house or a steward marks who paid. The totals are per due, collected
// against expected, and never a house's sum across dues. The portal records;
// it moves no money, so there is no field for an account or a link.

type Owed = { row_id: number; house_id: number | null; name: string | null; crest: string | null; color: string | null; paid_at: string | null; paid_by_name: string | null };
type Due = {
  id: number; title: string; amount_cents: number | null; currency: string; due_on: string | null; notes: string;
  collector_id: number | null; collector_name: string | null; collector_crest: string | null; collector_color: string | null;
  owed: number; paid: number; expected_cents?: number; collected_cents?: number; houses: Owed[];
};
type Form = { title: string; amount: string; currency: string; due_on: string; notes: string; collector_id: number; houses: number[] };

export function Dues({ me, houses }: { me: Me; houses: House[] }) {
  const { t, lang } = useT();
  const { items, err: loadErr, reload } = useList<Due>("/dues");
  const steward = me.is_steward === 1;
  const members = houses.filter((h) => h.kind === "house");
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState<number | null>(null);
  const [err, setErr] = useState("");

  // A currency the village already writes, so the form does not ask again.
  const blank = (): Form => ({ title: "", amount: "", currency: items[0]?.currency || "€", due_on: "", notes: "", collector_id: me.id, houses: members.map((h) => h.id) });
  const [f, setF] = useState<Form>(blank);

  const money = (cents: number, currency: string) => {
    const n = new Intl.NumberFormat(localeOf(lang), { minimumFractionDigits: cents % 100 ? 2 : 0, maximumFractionDigits: 2 }).format(cents / 100);
    return currency ? `${n} ${currency}` : n;
  };
  // paid_at is a UTC stamp; parse() shifts it, so the day is the village's.
  const day = (iso: string) => parse(iso).toLocaleDateString(localeOf(lang), { day: "numeric", month: "short" });
  const today = stamp().slice(0, 10);
  const body = (v: Form) => ({ ...v, amount: v.amount.trim() });
  const save = async (e: React.FormEvent) => {
    e.preventDefault(); setErr("");
    try {
      if (editing) await api(`/dues/${editing}`, { method: "PUT", body: body(f) });
      else await api("/dues", { method: "POST", body: body(f) });
      setAdding(false); setEditing(null); reload();
    } catch (x: any) { setErr(String(x.message)); }
  };
  const startEdit = (d: Due) => {
    setF({
      title: d.title, amount: d.amount_cents == null ? "" : String(d.amount_cents / 100), currency: d.currency, due_on: d.due_on || "", notes: d.notes,
      collector_id: d.collector_id ?? me.id, houses: d.houses.map((o) => o.house_id).filter((x): x is number => x != null),
    });
    setEditing(d.id); setAdding(false); setErr("");
  };
  const mark = (d: Due, o: Owed, paid: boolean) => api(`/dues/${d.id}/houses/${o.house_id}`, { method: "PUT", body: { paid } }).then(reload);
  const settle = (d: Due, o: Owed, method: "PUT" | "DELETE", paid?: boolean) => api(`/dues/${d.id}/rows/${o.row_id}`, { method, body: method === "PUT" ? { paid } : undefined }).then(reload);
  const del = (d: Due) => confirm(d.title + " — " + t("the due and who paid it. Only last night’s backup can bring it back.")) && api(`/dues/${d.id}`, { method: "DELETE" }).then(reload);

  const form = (
    <form className="inline" onSubmit={save}>
      <label>{t("What for")}<input value={f.title} onChange={(e) => setF({ ...f, title: e.target.value })} required maxLength={80} placeholder={t("e.g. Water, third quarter")} /></label>
      <div className="row">
        <label>{t("Per house")}<input value={f.amount} onChange={(e) => setF({ ...f, amount: e.target.value })} inputMode="decimal" maxLength={12} placeholder={t("no amount")} /></label>
        <label>{t("Currency")}<input value={f.currency} onChange={(e) => setF({ ...f, currency: e.target.value })} maxLength={8} /></label>
      </div>
      <div className="row">
        <label>{t("Pay by")}<DatePicker value={f.due_on} onChange={(v) => setF({ ...f, due_on: v })} placeholder={t("no date")} /></label>
        <label>{t("Collected by")}<select value={f.collector_id} onChange={(e) => setF({ ...f, collector_id: Number(e.target.value) })}>
          {members.map((h) => <option key={h.id} value={h.id}>{h.crest} {h.name}</option>)}
        </select></label>
      </div>
      <span className="small">{t("Houses asked")}</span>
      <div className="chips">
        {members.map((h) => {
          const on = f.houses.includes(h.id);
          // A house that has paid stays on the due (the backend says 409):
          // shown here before the tap, not after the save.
          const paid = !!items.find((d) => d.id === editing)?.houses.some((o) => o.house_id === h.id && o.paid_at);
          return <button type="button" key={h.id} className={"chip" + (on ? " on" : "")} aria-pressed={on} disabled={paid} title={paid ? t("paid — mark it unpaid first") : undefined}
            onClick={() => setF({ ...f, houses: on ? f.houses.filter((x) => x !== h.id) : [...f.houses, h.id] })}>{h.crest} {h.name}</button>;
        })}
      </div>
      <label>{t("Notes")}<textarea value={f.notes} onChange={(e) => setF({ ...f, notes: e.target.value })} maxLength={300} placeholder={t("how it is paid — cash to the collecting house, at the meeting")} /></label>
      {err && <p className="small" style={{ color: "var(--red)" }}>{err}</p>}
      <div className="submit">
        <button type="button" className="ghost" onClick={() => { setAdding(false); setEditing(null); }}>✕</button>
        <button className="primary" type="submit">🪙 {t("Save")}</button>
      </div>
    </form>
  );

  return (
    <div className="parchment">
      <h2>🪙 {t("Dues")} <span className="sub">{t("who paid what the village asked")}</span></h2>
      {steward && !adding && editing === null && <p><button onClick={() => { setF(blank()); setAdding(true); setErr(""); }}>+ {t("New due")}</button></p>}
      {adding && form}
      {loadErr && <Empty text={loadErr} />}
      {!loadErr && items.length === 0 && <Empty text={t("Nothing asked yet. A steward writes the first due down.")} />}
      {items.map((d) => {
        const settled = d.owed > 0 && d.paid === d.owed;
        const collects = steward || d.collector_id === me.id;
        const mine = d.houses.find((o) => o.house_id === me.id);
        if (editing === d.id) return <div key={d.id} className="card">{form}</div>;
        const list = (
          <ul className="due-houses">
            {d.houses.map((o, i) => (
              <li key={o.house_id ?? "gone" + i}>
                {o.house_id != null && <Crest crest={o.crest!} color={o.color!} />}
                <span className="due-house">
                  {o.house_id != null ? <span className="who">{o.name}</span> : <span className="muted">{t("a house that has left")}</span>}
                  {/* The day, not the hour: the day the money came is the fact.
                      Who marked it shows only when it was not the collector —
                      a steward marking in the collector's place is worth a word. */}
                  <span className="small">{o.paid_at ? <>✓ {day(o.paid_at)}{o.paid_by_name && o.paid_by_name !== d.collector_name ? ` · ${o.paid_by_name}` : ""}</> : t("not yet")}</span>
                </span>
                {collects && o.house_id != null && (o.paid_at
                  ? <button className="lesser" onClick={() => mark(d, o, false)}>{t("Undo")}</button>
                  : <button className="lesser" onClick={() => mark(d, o, true)}>✓ {t("Mark paid")}</button>)}
                {/* A house that has left: paid at the door, or forgiven — by
                    the collector's hand, never by deleting the house. */}
                {collects && o.house_id == null && (o.paid_at
                  ? <button className="lesser" onClick={() => settle(d, o, "PUT", false)}>{t("Undo")}</button>
                  : <>
                    <button className="lesser" onClick={() => settle(d, o, "PUT", true)}>✓ {t("Mark paid")}</button>
                    <button className="ghost" onClick={() => confirm(t("Forgive what a house that has left still owes on this due?")) && settle(d, o, "DELETE")}>{t("Forgive")}</button>
                  </>)}
              </li>
            ))}
          </ul>
        );
        return (
          <div key={d.id} className={"card due" + (settled ? " faded" : "")}>
            <div className="head">
              <strong>{d.title}</strong>
              {/* Plain until the date has passed: a due written this morning
                  is not a debt yet, and amber on it would say so. */}
              {mine && <span className={"tag" + (mine.paid_at ? " done" : d.due_on && d.due_on < today ? " taken" : "")}>{mine.paid_at ? "✓ " + t("you paid") : t("you have not paid")}</span>}
              {d.due_on && <span className="when">{t("by")} <When iso={d.due_on} /></span>}
            </div>
            <div className="small">
              {d.amount_cents != null && <>{money(d.amount_cents, d.currency)} {t("per house")} · </>}
              {d.collector_name ? <>{t("collected by")} <Crest crest={d.collector_crest!} color={d.collector_color!} /> {d.collector_name}</> : t("nobody collects it now")}
            </div>
            {/* The one total the portal keeps, and only in this room: this
                due's, collected against expected. Never a house's. */}
            <div className="due-sum">
              {d.paid} / {d.owed} {t("houses paid")}
              {d.expected_cents != null && <> · {money(d.collected_cents!, d.currency)} / {money(d.expected_cents, d.currency)}</>}
            </div>
            {d.notes && <div className="body small">{d.notes}</div>}
            {/* A settled due folds its houses away; opening it is on purpose. */}
            {settled ? <details><summary className="small">{t("Houses")}</summary>{list}</details> : list}
            {steward && <div className="actions">
              <button className="ghost" onClick={() => startEdit(d)}>✎ {t("Edit")}</button>
              <button className="ghost" onClick={() => del(d)}>🗑 {t("Delete")}</button>
            </div>}
          </div>
        );
      })}
    </div>
  );
}
