import { useState } from "react";
import { api } from "./api";
import { useT } from "./i18n";

// A PIN is 4 to 12 digits (auth.go § PINs): the number pad on a phone, dots
// on the screen, and anything that is not a digit never lands in the field.
export function PinInput({ label, value, onChange, autoFocus }: { label: string; value: string; onChange: (v: string) => void; autoFocus?: boolean }) {
  return (
    <label>{label}
      <input type="password" inputMode="numeric" pattern="[0-9]{4,12}" minLength={4} maxLength={12} autoComplete="off" required autoFocus={autoFocus}
        value={value} onChange={(e) => onChange(e.target.value.replace(/\D/g, ""))} />
    </label>
  );
}

// Choosing a PIN asks twice, like a phone's lock screen: a typo here is a PIN
// nobody knows, and the way back from it is the house's link.
export function NewPin({ pin, setPin, again, setAgain }: { pin: string; setPin: (v: string) => void; again: string; setAgain: (v: string) => void }) {
  const { t } = useT();
  return (
    <div className="row">
      <PinInput label={t("Choose a PIN")} value={pin} onChange={setPin} />
      <PinInput label={t("The PIN again")} value={again} onChange={setAgain} />
    </div>
  );
}

export function pinProblem(pin: string, again: string, t: (s: string) => string) {
  if (!/^\d{4,12}$/.test(pin)) return t("A PIN is 4 to 12 digits.");
  if (pin !== again) return t("The two PINs differ.");
  return "";
}

// SetPin chooses or changes this device's PIN. A change moves every browser
// of the house that signed in with the old one (auth.go putPin).
export function SetPin({ has, onSaved }: { has: boolean; onSaved: () => void }) {
  const { t } = useT();
  const [pin, setPin] = useState("");
  const [again, setAgain] = useState("");
  const [err, setErr] = useState("");
  const save = async (e: React.FormEvent) => {
    e.preventDefault(); setErr("");
    const bad = pinProblem(pin, again, t);
    if (bad) { setErr(bad); return; }
    try {
      await api("/me/pin", { method: "PUT", body: { pin } });
      setPin(""); setAgain(""); onSaved();
    } catch { setErr(t("Something went wrong")); }
  };
  return (
    <form className="inline" onSubmit={save}>
      <NewPin pin={pin} setPin={setPin} again={again} setAgain={setAgain} />
      {err && <div className="err">{err}</div>}
      <div className="submit"><button className="primary" type="submit">{has ? t("Change PIN") : t("Save")}</button></div>
    </form>
  );
}

// PinPrompt: on Home, for a device signed in before the PIN existed, until it
// chooses one or says later. Without a PIN the home-screen app and every
// other browser need a new link from a steward, which is what the PIN ends.
export function PinPrompt() {
  const { t } = useT();
  const [later, setLater] = useState(() => { try { return localStorage.getItem("potok.pinLater") === "1"; } catch { return false; } });
  const [done, setDone] = useState(false);
  if (later) return null;
  const dismiss = () => { setLater(true); try { localStorage.setItem("potok.pinLater", "1"); } catch { /* private mode */ } };
  return (
    <div className="card banner">
      <div className="body">
        {done ? <><strong>🔑 {t("PIN saved.")}</strong> {t("On another phone or computer, or in the app on your home screen: tap your house and type it.")}</>
          : <><strong>🔑 {t("Choose a PIN")}</strong> — {t("then you sign in on any phone or computer, and in the app on your home screen, by tapping your house and typing it. Nobody has to send you a code again.")}
            <SetPin has={false} onSaved={() => setDone(true)} /></>}
      </div>
      <button className="ghost dismiss" aria-label={t("Dismiss")} onClick={dismiss}>✕</button>
    </div>
  );
}
