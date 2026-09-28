import { useState } from "react";
import { api, ApiError } from "./api";
import { useT } from "./i18n";
import { useFeatures } from "./features";

// A message to whoever builds the portal, from the foot of Home beside the
// changelog: the changelog says what changed, this is how a villager asks for
// the next change without a WhatsApp message to the builder. The backend
// emails it or logs it and keeps nothing (feedback.go). The sections are the
// rooms Home shows and the doors that are not rooms; the English name is
// what travels, so the builder reads one language whatever the page speaks.
type Room = { icon: string; name: string; feature?: string };
const DOORS: [string, string][] = [["🗺️", "Map"], ["📜", "Codex"], ["🔔", "Notifications"], ["💭", "Something else"]];

export function Feedback({ rooms }: { rooms: readonly Room[] }) {
  const { t, lang } = useT();
  const { list } = useFeatures();
  // A room behind a switch that is off is absent from Home, so from here too.
  const on = (k?: string) => !k || !!list.find((f) => f.key === k)?.on;
  const sections: [string, string][] = [...rooms.filter((r) => on(r.feature)).map((r): [string, string] => [r.icon, r.name]), ...DOORS];
  const [section, setSection] = useState("Something else");
  const [reply, setReply] = useState("");
  const [text, setText] = useState("");
  const [state, setState] = useState<"" | "sending" | "sent" | "failed" | "busy" | "full" | "bad">("");

  const send = async (e: React.FormEvent) => {
    e.preventDefault();
    setState("sending");
    try {
      await api("/feedback", { method: "POST", body: { section, text, reply_to: reply, lang } });
      setText(""); setState("sent");
    } catch (err) {
      // The words stay in the box: a failed send must not cost the villager them.
      const s = err instanceof ApiError ? err.status : 0;
      setState(s === 429 ? "busy" : s === 503 ? "full" : s === 400 ? "bad" : "failed");
    }
  };

  return (
    <div className="parchment">
      <h2>✉️ {t("Feedback")} <span className="sub">{t("to whoever builds the portal")}</span></h2>
      <p className="small muted">{t("It goes to whoever builds the portal, not to the village.")} {t("Your house's name goes with it.")}</p>
      {state === "sent" ? (
        <p className="ok">{t("Sent. Thank you.")} <button className="lesser" onClick={() => setState("")}>{t("Write another")}</button></p>
      ) : (
        <form className="inline" onSubmit={send}>
          <label>{t("About")}
            <select value={section} onChange={(e) => setSection(e.target.value)}>
              {sections.map(([icon, name]) => <option key={name} value={name}>{icon} {t(name)}</option>)}
            </select>
          </label>
          <label>{t("Your email (optional)")}
            <input type="email" value={reply} onChange={(e) => setReply(e.target.value)} maxLength={254} autoComplete="email" />
            <span className="small muted">{t("Only used to answer this message.")}</span>
          </label>
          <label>{t("Message")}
            <textarea value={text} onChange={(e) => setText(e.target.value)} required maxLength={4000} rows={5} placeholder={t("what is wrong, what is missing, what you would change")} />
          </label>
          {state === "failed" && <div className="err">{t("Could not send, try again later.")}</div>}
          {state === "busy" && <div className="err">{t("Too many messages, try again later.")}</div>}
          {state === "full" && <div className="err">{t("The builder's mailbox is full for today. It did not go — try tomorrow.")}</div>}
          {state === "bad" && <div className="err">{t("Check the email address and the message.")}</div>}
          <div className="submit"><button className="primary" type="submit" disabled={state === "sending" || !text.trim()}>{state === "sending" ? t("Sending…") : `✉️ ${t("Send")}`}</button></div>
        </form>
      )}
    </div>
  );
}
