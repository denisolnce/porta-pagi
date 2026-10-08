import { useEffect, useRef, useState, type ReactNode } from "react";
import { Link, Navigate, useParams, useSearchParams } from "react-router-dom";
import { useT } from "./i18n";
import { useFeatures } from "./features";

// A link to one entry (CLAUDE.md § A link to one entry, docs/design-links.md).
// What a Share button, a comment push and an agent hand out is one shape,
// #/go/<kind>/<id>/<made>, and Go below is the one place that knows which
// room a kind lives in and whether that room is on. It redirects to the
// room's own address, a parameter on the room: #/tavern?event=12&made=….
//
// <made> is the UTC day the row was written, its created_at as stored. No
// table uses AUTOINCREMENT, so the id of a deleted newest entry comes back on
// the next one written; the room compares the day before it scrolls, and a
// row of another day is "not here any more". The sky has no created_at and
// its own day is its identity, so an almanac link carries starts_at's day; a
// codex section has neither and carries none.
type Kind = { room: string; param: string; feature?: string };
const KINDS: Record<string, Kind> = {
  event: { room: "/tavern", param: "event" },
  post: { room: "/tavern", param: "post" },
  almanac: { room: "/tavern", param: "almanac", feature: "celestial" },
  run: { room: "/market", param: "run" },
  need: { room: "/market", param: "need" },
  offer: { room: "/market", param: "offer" },
  tool: { room: "/shed", param: "tool" },
  wish: { room: "/shed", param: "wish" },
  section: { room: "/codex", param: "section" },
  project: { room: "/projects", param: "" }, // a page of its own: #/projects/ID
};

// The day a link carries for a row: see above.
export const madeOf = (row: { created_at?: string; starts_at?: string }, kind?: string) =>
  ((kind === "almanac" ? row.starts_at : row.created_at) || "").slice(0, 10);

// entryURL is absolute, as the invite link is, so it works pasted anywhere.
// A series is answered one day at a time, so its link names the day.
export function entryURL(kind: string, id: number, made: string, day?: string) {
  return `${location.origin}${location.pathname}#/go/${kind}/${id}${made ? "/" + made : ""}${day ? "?day=" + day : ""}`;
}

// The landing lines travel in the router's state, so the next navigation,
// which carries none, takes them away. "gone" is the one case that means
// deleted — a day in the link that is not the row's, or an unknown kind;
// "hidden" is a row the list no longer shows: a run the day after its
// cut-off, a need or a give-away seven days after done, an event after
// sixty days. The honest line for those says shown, not deleted.
export type LinkNote = "off" | "gone" | "hidden";

export function Go() {
  const { kind = "", id = "", made = "" } = useParams();
  const [params] = useSearchParams();
  const features = useFeatures().list;
  const k = KINDS[kind];
  if (!k || !/^\d+$/.test(id)) return <Navigate to="/" replace state={{ linkNote: "gone" as LinkNote }} />;
  if (k.feature && !features.find((f) => f.key === k.feature)?.on) return <Navigate to="/" replace state={{ linkNote: "off" as LinkNote }} />;
  const q = new URLSearchParams();
  if (k.param) q.set(k.param, id);
  if (/^\d{4}-\d{2}-\d{2}$/.test(made)) q.set("made", made);
  const day = params.get("day");
  if (day && /^\d{4}-\d{2}-\d{2}$/.test(day)) q.set("day", day);
  const path = k.param ? k.room : `${k.room}/${id}`;
  return <Navigate to={`${path}?${q}`} replace />;
}

// The faint line a landing leaves when it cannot open what it was sent to.
export function LinkLine({ note }: { note: LinkNote | null | undefined }) {
  const { t } = useT();
  if (!note) return null;
  return <p className="small muted link-note">{note === "off" ? t("That room is switched off in this village.") : note === "hidden" ? t("That entry is no longer shown here.") : t("That entry is not here any more — it may have been deleted.")}</p>;
}

// useLanding opens one entry a link points at, once per address: it waits
// for the room's list, finds the row by ?<param>= and checks ?made=, hands
// the row to the room (which pages its view, picks a tab, opens a thread),
// then scrolls the card that carries data-entry="<param>-<id>" into view and
// marks it for a moment. It answers with the landing line the room owes, or null.
export function useLanding(param: string, items: any[], loaded: boolean, onFound?: (row: any) => void, kind?: string): LinkNote | null {
  const [params] = useSearchParams();
  const id = Number(params.get(param) || 0);
  const made = params.get("made") || "";
  const key = id ? `${param}:${id}:${made}` : "";
  const done = useRef("");
  const [missing, setMissing] = useState<LinkNote | null>(null);
  const found = useRef(onFound);
  found.current = onFound;
  useEffect(() => {
    if (!key) { done.current = ""; setMissing(null); return; }
    if (!loaded || done.current === key) return;
    done.current = key;
    const row = items.find((x) => x.id === id);
    if (!row) { setMissing("hidden"); return; }
    if (made && madeOf(row, kind) !== made) { setMissing("gone"); return; }
    setMissing(null);
    found.current?.(row);
    // After the room has drawn what onFound changed: the card may sit in
    // another place once the month or the tab moved.
    setTimeout(() => reveal(`${param}-${id}`), 100);
  }, [key, loaded, items, id, made, param, kind]);
  return missing;
}

// reveal waits for the card to be drawn — a month paged, a tab picked, the
// wishlist unfolded — then scrolls to it and lights its frame for 1.5 s.
export function reveal(entry: string, tries = 20) {
  const el = document.querySelector<HTMLElement>(`[data-entry="${entry}"]`);
  if (!el) { if (tries > 0) setTimeout(() => reveal(entry, tries - 1), 100); return; }
  // The head of the card, not its middle: an open thread makes a card
  // taller than a phone's screen.
  el.scrollIntoView({ behavior: "smooth", block: "start" });
  el.classList.add("landed");
  setTimeout(() => el.classList.remove("landed"), 1500);
}

// The Share button: the phone's share sheet where the browser has one, else
// the clipboard and "Link copied", the invite link's own behaviour. Only the
// address goes: the villager writes the sentence. Ghost is the assistant's
// pick, pending the owner (CLAUDE.md § Buttons gives ghost to dismiss and
// delete only).
export function Share({ kind, id, made, day }: { kind: string; id: number; made: string; day?: string }) {
  const { t } = useT();
  const [copied, setCopied] = useState(false);
  const share = async () => {
    const url = entryURL(kind, id, made, day);
    // The system sheet on a phone, where WhatsApp is two taps away; the
    // clipboard on a laptop, where the sheet offers Mail and AirDrop and a
    // villager at WhatsApp Web wants to paste.
    if (navigator.share && matchMedia("(pointer: coarse)").matches) {
      try { await navigator.share({ url }); } catch { /* the sheet was closed */ }
      return;
    }
    navigator.clipboard?.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  return <button className="ghost" onClick={share}>🔗 {copied ? t("Link copied") : t("Share")}</button>;
}

// Linkify: a post or a comment is plain text, and no page makes a link out of
// text by itself. Only http and https runs become links. A link into this
// portal opens inside the app, in this tab; any other opens beside it, and
// noreferrer keeps the portal's address out of the request.
const URL_RUN = /https?:\/\/[^\s<>"]+/g;
export function Linkify({ text }: { text: string }) {
  const out: ReactNode[] = [];
  let last = 0;
  const here = location.origin + location.pathname;
  for (const m of text.matchAll(URL_RUN)) {
    // A sentence's own full stop or bracket is not part of the address.
    const url = m[0].replace(/[.,;:!?)\]'’]+$/, "");
    const at = m.index!;
    if (at > last) out.push(text.slice(last, at));
    if (url.startsWith(here + "#/")) out.push(<Link key={at} to={url.slice(here.length + 1)}>{url}</Link>);
    else out.push(<a key={at} href={url} target="_blank" rel="noreferrer noopener">{url}</a>);
    last = at + url.length;
  }
  if (last < text.length) out.push(text.slice(last));
  return <>{out}</>;
}
