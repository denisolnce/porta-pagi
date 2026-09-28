import { useEffect, useState } from "react";
import { api, API, getToken } from "../api";
import { localeOf, useT } from "../i18n";
import { useFeatures, type FeatureRow } from "../features";
import { KIND_LABELS } from "./Notifications";
import { parse } from "./shared";

// What each switch adds, in one line. A switch with no line here shows its key.
const ABOUT: Record<string, string> = {
  camp: "Campground — who noticed a camper, who has the money",
  dues: "Dues ledger — who paid what, and when",
  roster: "Duty roster — turns at the village's chores, with counts",
};
// The rooms that count. The portal counts nothing between houses by default,
// so switching one of these on is a change of what the village is, and the
// steward is asked once more (docs/commercial.md § Decided).
const COUNTING = ["dues", "roster"];

type Prefs = { kinds: string[]; global_off: string[]; global_detail: { kind: string; set_by: string | null; set_at: string }[] };

// Village settings: what a steward decides for the village as a whole — its
// switches, the village-wide mutes, the export. Reached by ⚙️ in the top bar;
// the route exists only for a steward. What is about houses (creating them,
// invites, land, stewardship) stays in the Houses room.
export function Settings({ icons }: { icons: Record<string, string> }) {
  // The demo is one village for every visitor at once: a switch flipped by
  // one would take a room from all the others for the rest of the hour.
  const { t, lang, demo } = useT();
  const { list, reload } = useFeatures();
  // set_at is a UTC stamp; the day it names is the village's, not Greenwich's.
  const day = (iso: string) => parse(iso).toLocaleDateString(localeOf(lang), { day: "numeric", month: "short", year: "numeric" });
  const [prefs, setPrefs] = useState<Prefs | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [err, setErr] = useState("");
  const loadPrefs = () => api<Prefs>("/me/prefs").then(setPrefs).catch(() => {});
  useEffect(() => { loadPrefs(); }, []);

  const flip = async (f: FeatureRow) => {
    const on = !f.on;
    if (on && COUNTING.includes(f.key) && !confirm(`${t(ABOUT[f.key] || f.key)}\n\n${t("This adds counting to a portal that counts nothing by default. Switch it on?")}`)) return;
    // Off is one tap that takes a room from every house, the campground's
    // money in flight with it: say so first. Nothing is deleted.
    if (!on && !confirm(`${t(ABOUT[f.key] || f.key)}\n\n${t("Every house loses this room until a steward switches it back on. Nothing in it is deleted.")}`)) return;
    setBusy(f.key); setErr("");
    try {
      await api(`/features/${f.key}`, { method: "PUT", body: { on } });
      // A switch takes its push kind with it, so the mutes below change too.
      await Promise.all([reload(), loadPrefs()]);
    } catch { setErr(t("Something went wrong")); } finally { setBusy(null); }
  };
  // Who flipped it and when. A row with neither is the campground a village
  // already had when switches arrived; no row at all is the default.
  const who = (f: FeatureRow) => {
    if (f.set_at === undefined && f.set_by === undefined) return "";
    if (!f.set_at && !f.set_by) return t("The village had this before switches.");
    return `${f.on ? t("Switched on by") : t("Switched off by")} ${f.set_by || "?"}${f.set_at ? " · " + day(f.set_at) : ""}`;
  };

  // One switch mutes a kind for every house in the village.
  const toggleMute = async (k: string) => {
    if (!prefs) return;
    const off = prefs.global_off.includes(k) ? prefs.global_off.filter((x) => x !== k) : [...prefs.global_off, k];
    setPrefs({ ...prefs, global_off: off });
    await api("/prefs/global", { method: "PUT", body: { off } });
    loadPrefs(); // the who and when of a new mute come from the server
  };
  const mutedBy = (k: string) => { const d = prefs?.global_detail.find((x) => x.kind === k); return d ? `${t("Switched off for the whole village by")} ${d.set_by || "?"} · ${day(d.set_at)}` : ""; };

  const exportAll = async () => {
    const r = await fetch(`${API}/export`, { headers: { Authorization: "Bearer " + (getToken() || "") } });
    const b = await r.blob(); const u = URL.createObjectURL(b);
    const a = document.createElement("a"); a.href = u; a.download = "potok-export.json"; a.click(); URL.revokeObjectURL(u);
  };

  return (
    <div className="parchment">
      <h2>⚙️ {t("Village settings")}</h2>

      <h3>{t("Switches")}</h3>
      <p className="small">{t("A room that is off is absent: no door, no notifications, nothing for an agent. What it holds stays, and comes back when it is switched on.")}</p>
      {demo && <p className="small muted">{t("In the demo every switch stays on, so every visitor sees every room.")}</p>}
      <div className="kinds">
        {list.map((f) => (
          <div key={f.key} className="kind wide switch">
            <label className="kind"><input type="checkbox" checked={f.on} disabled={busy !== null || demo} onChange={() => flip(f)} /> {icons[f.key] ? icons[f.key] + " " : ""}{t(ABOUT[f.key] || f.key)}</label>
            {who(f) && <span className="small muted">{who(f)}</span>}
          </div>
        ))}
      </div>
      {err && <div className="err">{err}</div>}

      <h3>🔇 {t("Notifications")}</h3>
      {prefs && (
        <div className="kinds">
          <div className="small">🗝️ {t("Village-wide (steward): kinds switched off here reach nobody, whatever each house chose.")}</div>
          {prefs.kinds.map((k) => (
            <label key={k} className="kind" title={mutedBy(k)}><input type="checkbox" checked={!prefs.global_off.includes(k)} onChange={() => toggleMute(k)} /> {t(KIND_LABELS[k] || k)}</label>
          ))}
        </div>
      )}
      {prefs && prefs.global_detail.length > 0 && <div className="small">🔇 {prefs.global_detail.map((d) => `${t(KIND_LABELS[d.kind] || d.kind)}: ${d.set_by || "?"} · ${day(d.set_at)}`).join(" · ")}</div>}

      <h3>📦 {t("Export")}</h3>
      <p className="small">{t("Everything the village wrote, in one file. Take one before deleting a house.")}</p>
      <p><button type="button" onClick={exportAll}>📦 {t("Export everything")}</button></p>
    </div>
  );
}
