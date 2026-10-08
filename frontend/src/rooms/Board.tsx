import { useEffect, useRef, useState } from "react";
import { api, type Me } from "../api";
import { useT } from "../i18n";
import { Crest, Empty, When, canEdit } from "./shared";
import { defaultAuthor, PostAs } from "./Thread";

// The message board half of the tavern. The calendar is the other half —
// Hall.tsx stacks them, because a village opens one room, not two.
//
// The write box stays at the top: a post is what the room is for. A reply is
// a form of its own under the post's replies, the shape a thread has
// (Thread.tsx) — the two used to share one box, so a reply dragged the
// villager back up to the top of the board and the box changed meaning
// under the thumb. Both submit rows read "Post as <name>" (owner's ask
// 2026-10-08).
export function Board({ me, items, reload }: { me: Me; items: any[]; reload: () => void }) {
  const { t } = useT();
  const [body, setBody] = useState("");
  const [author, setAuthor] = useState(() => defaultAuthor(me));
  const [replyTo, setReplyTo] = useState<number | null>(null);
  const [replyBody, setReplyBody] = useState("");
  const board = useRef<HTMLDivElement>(null);
  // The reply box appears under the post; focusing it brings it into view.
  // A draft moves with the box only while it is the same post's: opening a
  // reply under another post starts clean. Cancel keeps the words (a thread's
  // does too) and puts the focus back on the Reply button it left.
  const lastReply = useRef<number | null>(null);
  useEffect(() => {
    if (replyTo != null) {
      if (lastReply.current != null && lastReply.current !== replyTo) setReplyBody("");
      lastReply.current = replyTo;
      board.current?.querySelector<HTMLTextAreaElement>("form.reply textarea")?.focus();
    }
  }, [replyTo]);
  const cancelReply = (id: number) => { setReplyTo(null); board.current?.querySelector<HTMLButtonElement>(`[data-reply="${id}"]`)?.focus(); };

  const send = async (text: string, parent?: number) => {
    if (!text.trim()) return false;
    try { localStorage.setItem("potok.author", author); } catch { /* ignore */ }
    await api("/posts", { method: "POST", body: { body: text, author, parent_id: parent } });
    reload();
    return true;
  };
  const post = async (e: React.FormEvent) => { e.preventDefault(); if (await send(body)) setBody(""); };
  const reply = async (e: React.FormEvent) => { e.preventDefault(); if (await send(replyBody, replyTo!)) { setReplyBody(""); setReplyTo(null); } };
  const roots = items.filter((p) => !p.parent_id);
  const replies = (id: number) => items.filter((p) => p.parent_id === id).sort((a, b) => a.created_at.localeCompare(b.created_at));

  return (
    <div className="parchment" ref={board}>
      <h2>📜 {t("The board")} <span className="sub">{t("what must stay findable")}</span></h2>
      <form className="inline" onSubmit={post}>
        <label>{t("Write to the village")}<textarea value={body} onChange={(e) => setBody(e.target.value)} placeholder={t("What is on your mind")} maxLength={4000} /></label>
        <PostAs author={author} setAuthor={setAuthor} label={t("Post")} primary />
      </form>
      {roots.length === 0 && <Empty text={t("No posts yet. Be the first.")} />}
      {roots.map((p) => (
        <div key={p.id} className={"card" + (p.pinned ? " pinned" : "")}>
          <div className="head"><Crest crest={p.house_crest} color={p.house_color} /><span className="who">{p.house_name}{p.author && p.author !== p.house_name ? ` · ${p.author}` : ""}</span>{p.pinned ? <span className="tag alarm">📌 {t("Pinned")}</span> : null}<When iso={p.created_at} /></div>
          <div className="body">{p.body}</div>
          <div className="actions">
            <button className="ghost" data-reply={p.id} onClick={() => (replyTo === p.id ? cancelReply(p.id) : setReplyTo(p.id))}>↩ {t("Reply")}</button>
            {me.is_steward === 1 && <button className="ghost" onClick={() => api(`/posts/${p.id}`, { method: "PUT", body: { pinned: !p.pinned } }).then(reload)}>📌 {p.pinned ? t("Unpin") : t("Pin")}</button>}
            {canEdit(me, p) && <button className="ghost" onClick={() => confirm("?") && api(`/posts/${p.id}`, { method: "DELETE" }).then(reload)}>🗑 {t("Delete")}</button>}
          </div>
          {replies(p.id).map((r) => (
            <div key={r.id} className="card" style={{ marginLeft: "1.5rem", borderLeftColor: "var(--parch3)" }}>
              <div className="head"><Crest crest={r.house_crest} color={r.house_color} /><span className="who">{r.house_name}{r.author && r.author !== r.house_name ? ` · ${r.author}` : ""}</span><When iso={r.created_at} /></div>
              <div className="body">{r.body}</div>
              {canEdit(me, r) && <div className="actions"><button className="ghost" onClick={() => confirm("?") && api(`/posts/${r.id}`, { method: "DELETE" }).then(reload)}>🗑 {t("Delete")}</button></div>}
            </div>
          ))}
          {replyTo === p.id && (
            <form className="inline reply" onSubmit={reply} style={{ marginLeft: "1.5rem" }}>
              <label>{t("Reply")}<textarea value={replyBody} onChange={(e) => setReplyBody(e.target.value)} maxLength={4000} /></label>
              <PostAs author={author} setAuthor={setAuthor} label={t("Reply")} onCancel={() => cancelReply(p.id)} />
            </form>
          )}
        </div>
      ))}
    </div>
  );
}
