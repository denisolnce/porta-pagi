# Getting in, and staying in: the session problem

Brainstormed 2026-10-08 on the owner's ask, nothing built. The owner's
words: the WhatsApp webview and a desktop browser opening a link logged out
"yet again highlights that our code access for houses is a bit wonky and
uncomfortable for people … maybe a pincode that each user sets upon
initially using the first login link? then they use pincode to quickly log
in any time, like phone lockscreen code. what other login/session
persistence alleviating mechanisms can we brainstorm?"

**The villager job, in one line: "open the portal on whatever is in my
hand and be my house, without asking anyone."** Today that holds for the
one browser that opened the invite, and for nothing else in the hand.

**Said plainly first: the case the owner raised, a link tapped in WhatsApp
on an iPhone, is fixed by none of the options below.** WhatsApp's browser
there is a storage of its own, and iOS opens no link in the home-screen
app. What remains for it is what exists: pair that browser once, or the
install banner's "open in Safari". Android may do better by itself
(option G). The options fix the other doors — the quiet-week logout, the
laptop, the second phone — which is most of "wonky", not all of it.

## Where a session lives today, and how it dies

- A device token in `localStorage` (`api.ts`), sent as a bearer header,
  hashed in `devices`, **no expiry** — only `last_seen`. Nothing logs a
  phone out on purpose.
- **One storage per door.** Safari, the home-screen app, WhatsApp's webview
  and Chrome on the laptop are four storages on one person, and each is
  paired on its own. This is the whole "logged out again" feeling: not a
  session that ended, a door that never had one.
- **Safari forgets a site after seven days of Safari use without opening
  it** (its tracking prevention evicts script-written storage,
  `localStorage` included; the count is days Safari was used, so a phone in
  a drawer for a month keeps the portal and a phone used daily that skips it
  for a week loses it) unless the site is on the home screen. An iPhone that never installed the portal
  and had a quiet week opens the gate. The install banner is the fix, and
  it is the villager's to take.
- Clearing site data, a private window, a new phone: the gate, by design.
- The way back in is a steward's invite link (10 characters, 14 days,
  multi-use) or a pairing code from a phone that is in (6 digits, 15
  minutes, one use, burnt after five wrong guesses). Both are typed or
  tapped from WhatsApp, which lands in WhatsApp's browser.

## The owner's PIN idea, examined

A phone's lock code unlocks a device that already holds the key. It never
gets a new phone in. The need here is the second thing: a browser that has
nothing yet. A PIN used for that is a password, and a short one — ten houses
times ten thousand PINs, standing, against `/api/join`. The per-IP cap slows
one address; the pairing code survives the same arithmetic only because it
lives fifteen minutes and burns after five wrong guesses, and a standing PIN
can do neither. What survives of the idea is **a thing the house itself
holds and can type anywhere**, which is option D below, made long enough.

## Options

| Option | What the villager does | What it costs | What it breaks | Verdict |
| --- | --- | --- | --- | --- |
| A. **The token as a server-set cookie** (`HttpOnly`, `SameSite=Strict`, a year) | Nothing | `auth.go` reads the cookie beside the header; `api.ts` stops sending one; logout is a route. CSRF is covered by `SameSite=Strict` and the JSON content type the handlers already require. **Two more, each an invariant**: the roll must exchange what every phone holds — `api.ts` posts its `localStorage` token once and gets the cookie back, or the whole village is at the gate after the roll, the opposite of the ask; and `auth.go` must ignore the cookie at `/api/mcp`, because a cookie rides on every same-origin fetch and "the door takes a key and nothing else" (`CLAUDE.md` § A door for agents) would otherwise get the side door it forbids | Nothing seen. Whether Safari and the home-screen app share it is to be verified on a phone, not assumed | **First.** A server-set cookie is exempt from Safari's seven-day eviction, so an iPhone that never installed stays in |
| B. **The pairing code as a link too** | On a phone that is in: Houses → Add another device → *Copy link*, sent to the laptop (AirDrop, "message yourself", mail) and opened there | A second button beside the code; the gate takes `#/join/<code>` already | Nothing. Same fifteen minutes, one use | **With A.** No typing, and a link opened from WhatsApp Desktop lands in the system browser, logged in |
| C. **Reverse pairing with a QR** | The laptop's gate shows a QR; the phone's camera reads it, opens the portal logged in, and asks "let this browser in?" | A pending-pairings table, a QR encoder (a small dependency or ~150 lines), an approve page. iOS reads a QR with the camera app, so no scanner page is needed | Nothing; it is a second shape of today's code | **When laptops matter.** The WhatsApp Web pattern villagers already know |
| D. **A house passphrase** — a few words the house sets at first login, typed with the house name on any gate | Types "Zeleni Volk" and three words | A column, a form in the Houses room, a second mode on the gate. Three words of a two-thousand-word list are eight billion, and the join cap holds | **The invariant "no in-app password to reset"** (`CLAUDE.md` § The way back in). The reset exists — a steward rotates the invite and the house sets new words — but it is a password, and a thing written on the fridge | The owner's PIN made safe. A decision against an invariant, so the owner's |
| E. **The house's own standing link** — the invite shown to the house itself, not expiring, pinned in the house's own chat | Taps the pinned link in any browser | A longer life on `invites`, and the link shown in the Houses room to its house | A leak is now standing, not fourteen days; rotation by a steward is the brake, as today. A link tapped in WhatsApp still opens WhatsApp's browser | The cheapest. The owner weighs the leak |
| F. **Passkeys** (WebAuthn) | Face ID or a fingerprint; the laptop by the phone's QR, which the OS draws itself | A credentials table, the ceremony on the gate, CBOR and COSE verification — a Go dependency (`go-webauthn`) or some hundreds of lines, a decision in `CLAUDE.md` | Nothing in the model: a passkey is per person on a platform account, which is a device row each. WhatsApp's webview may not offer passkeys at all | **The solid end state**: nothing to remember, nothing to reset, nothing phishable, synced across one person's phones and laptops by Apple and Google. The heaviest, so after A–C |
| G. **Android link capture** | Nothing | Nothing, if true: an installed app on Android registers its scope, so a link tapped in WhatsApp may open the installed portal, logged in | iOS never does this | **Verify on an Android phone** before any of the above is built for WhatsApp's sake |
| H. **A way-in by email** through the mailer that feedback already uses | Types the house's address, taps the link in the mail | An address per house, a privacy line, the "no email to send" line in `CLAUDE.md` turned around | Email as identity, in a village that runs on WhatsApp. `later.md` keeps WhatsApp one-time codes at "v2 or never" for Meta's fee | No, for now |

## The ladder

1. **A and B**, together, small, no decision against an invariant: the
   quiet-week logout on iPhones goes, and a laptop is let in without typing.
2. **G verified** on one Android phone, because it may make the WhatsApp
   case free there.
3. **E** if the owner accepts a standing link, **D** if he wants the
   passphrase after all — one of the two, not both.
4. **C** when laptops are a habit, **F** as the end state once a dependency
   is accepted.

Whatever rung, **Remove in Your devices stays the brake**: every option
hands a browser the whole house, away notices included, and the house must
be able to take it back from a phone that is in.

## What was considered and refused

- A short PIN as the credential (above).
- A session that expires. Nothing is gained: a villager who is logged out
  for safety's sake is a villager who asks in the group for a code.
- A "remember me" box. Every session is remembered already; the problem is
  the doors, not the memory.
