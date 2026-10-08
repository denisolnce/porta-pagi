import { useEffect, useRef, useState } from "react";
import { api, type Me } from "../api";
import { useT } from "../i18n";
import { Crest, When } from "./shared";

// One comment thread, wherever a room wants one: an event, a wish, an away
// notice, a number in the phone book. The board's own shape, one reply level. The name field starts filled
// with the label this phone joined under, so most people never type it.
// The box sits under the comments. The name sits to the right of Post,
// so the row reads "Post as <name>", with "Name (optional)" over the field.
//
// The button that opens one is "💬 Comments" and carries the count only when
// there is one to carry — the same in every room that has a thread.
export function Thread({ subject, id, me, onChanged }: { subject: "event" | "wish" | "away" | "contact" | "almanac"; id: number; me: Me; onChanged?: () => void }) {
  const { t } = useT();
  const [items, setItems] = useState<any[]>([]);
  const [body, setBody] = useState("");
  const [replyTo, setReplyTo] = useState<number | null>(null);
  const [author, setAuthor] = useState(() => defaultAuthor(me));
  const formRef = useRef<HTMLFormElement>(null);
  const load = () => api<any[]>(`/threads/${subject}/${id}`).then(setItems).catch(() => setItems([]));
  // Reply used to open the box above the thread, in view. The box is under
  // the comments now, so a reply focuses it and the browser brings it up.
  useEffect(() => {
    if (replyTo != null) formRef.current?.querySelector("textarea")?.focus();
  }, [replyTo]);
  useEffect(() => { load(); }, [subject, id]); // eslint-disable-line react-hooks/exhaustive-deps

  const post = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!body.trim()) return;
    try { localStorage.setItem("potok.author", author); } catch { /* ignore */ }
    await api(`/threads/${subject}/${id}`, { method: "POST", body: { body, author, parent_id: replyTo ?? undefined } });
    setBody(""); setReplyTo(null); load(); onChanged?.();
  };
  const del = (cid: number) => confirm("?") && api(`/comments/${cid}`, { method: "DELETE" }).then(() => { load(); onChanged?.(); });
  const roots = items.filter((c) => !c.parent_id);
  const replies = (cid: number) => items.filter((c) => c.parent_id === cid);

  // A render function, not a component — see Market.tsx: as a component it
  // remounted every comment card on each keystroke in the reply box.
  const one = (c: any, reply?: boolean) => (
    <div key={c.id} className="card comment" style={reply ? { marginLeft: "1.4rem" } : undefined}>
      <div className="head"><Crest crest={c.house_crest} color={c.house_color} /><span className="who">{c.author || c.house_name}</span><When iso={c.created_at} /></div>
      <div className="body">{c.body}</div>
      <div className="actions">
        {!reply && <button className="ghost" onClick={() => setReplyTo(c.id)}>↩ {t("Reply")}</button>}
        {(c.house_id === me.id || me.is_steward === 1) && <button className="ghost" onClick={() => del(c.id)}>🗑</button>}
      </div>
    </div>
  );

  return (
    <div className="comments">
      {roots.length === 0 && <p className="small muted" style={{ fontStyle: "italic" }}>{t("No comments yet.")}</p>}
      {roots.map((c) => (<div key={c.id}>{one(c)}{replies(c.id).map((r) => one(r, true))}</div>))}
      <form className="inline" onSubmit={post} ref={formRef}>
        <label>{replyTo ? t("Reply") : t("Comment")}<textarea value={body} onChange={(e) => setBody(e.target.value)} maxLength={2000} /></label>
        <PostAs author={author} setAuthor={setAuthor} label={replyTo ? t("Reply") : t("Post")} onCancel={replyTo ? () => setReplyTo(null) : undefined} />
      </form>
    </div>
  );
}

// The submit row of anything a house writes under a name: "Post as <name>",
// the one shape in a thread, on the board and under a board post's reply.
// `primary` is the board's own post, the one action its form is for.
export function PostAs({ author, setAuthor, label, primary, onCancel }: { author: string; setAuthor: (s: string) => void; label: string; primary?: boolean; onCancel?: () => void }) {
  const { t } = useT();
  return (
    <div className="submit post-as">
      {onCancel && <button type="button" className="ghost" onClick={onCancel}>✕</button>}
      <button className={primary ? "primary" : "lesser"} type="submit">{label}</button>
      <span className="as-word">{t("as")}</span>
      <label className="as-name">{t("Name (optional)")}<input value={author} onChange={(e) => setAuthor(e.target.value)} maxLength={40} /></label>
    </div>
  );
}

// The name to start with: what this person typed last, else the label this
// phone joined under ("Ana's phone" → "Ana").
export function defaultAuthor(me: Me): string {
  try {
    const saved = localStorage.getItem("potok.author");
    if (saved) return saved;
  } catch { /* ignore */ }
  const label = (me as any).device_label as string | undefined;
  if (!label) return "";
  return label.replace(/[’']s\s+(phone|telefon).*$/i, "").replace(/\s*(phone|telefon|naprava|laptop|prenosnik|iphone|android)\s*$/i, "").trim();
}
