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
| A. **The token as a server-set cookie** (`HttpOnly`, `SameSite=Strict`, a year) | Nothing | **Built 2026-10-08** ("i see only the benefits so lets implement it"): `auth.go`, `TestSessionCookie`, the invariant in `CLAUDE.md` § A session is a cookie and a drawer. One change from the detail below: the drawer is kept, the cookie is the copy that survives | Nothing seen. Whether Safari and the home-screen app share it is to be verified on a phone, not assumed | **First.** A server-set cookie is exempt from Safari's seven-day eviction, so an iPhone that never installed stays in |
| B. **The pairing code as a link too** | On a phone that is in: Houses → Add another device → *Copy link*, sent to the laptop (AirDrop, "message yourself", mail) and opened there | A second button beside the code; the gate takes `#/join/<code>` already | Nothing. Same fifteen minutes, one use | **With A.** No typing, and a link opened from WhatsApp Desktop lands in the system browser, logged in |
| C. **Reverse pairing with a QR** | The laptop's gate shows a QR; the phone's camera reads it, opens the portal logged in, and asks "let this browser in?" | A pending-pairings table, a QR encoder (a small dependency or ~150 lines), an approve page. iOS reads a QR with the camera app, so no scanner page is needed | Nothing; it is a second shape of today's code | **When laptops matter.** The WhatsApp Web pattern villagers already know |
| D. **A house passphrase** — a few words the house sets at first login, typed with the house name on any gate | Types "Zeleni Volk" and three words | A column, a form in the Houses room, a second mode on the gate. Three words of a two-thousand-word list are eight billion, and the join cap holds | **The invariant "no in-app password to reset"** (`CLAUDE.md` § The way back in). The reset exists — a steward rotates the invite and the house sets new words — but it is a password, and a thing written on the fridge | The owner's PIN made safe. A decision against an invariant, so the owner's |
| E. **The house's own standing link** — the invite shown to the house itself, not expiring, pinned in the house's own chat | Taps the pinned link in any browser | A longer life on `invites`, and the link shown in the Houses room to its house | A leak is now standing, not fourteen days; rotation by a steward is the brake, as today. A link tapped in WhatsApp still opens WhatsApp's browser | The cheapest. The owner weighs the leak |
| F. **Passkeys** (WebAuthn) | Face ID or a fingerprint; the laptop by the phone's QR, which the OS draws itself | A credentials table, the ceremony on the gate, CBOR and COSE verification — a Go dependency (`go-webauthn`) or some hundreds of lines, a decision in `CLAUDE.md` | Nothing in the model: a passkey is per person on a platform account, which is a device row each. WhatsApp's webview may not offer passkeys at all | **The solid end state**: nothing to remember, nothing to reset, nothing phishable, synced across one person's phones and laptops by Apple and Google. The heaviest, so after A–C |
| G. **Android link capture** | Nothing | Nothing, if true: an installed app on Android registers its scope, so a link tapped in WhatsApp may open the installed portal, logged in | iOS never does this | **Verify on an Android phone** before any of the above is built for WhatsApp's sake |
| H. **A way-in by email** through the mailer that feedback already uses | Types the house's address, taps the link in the mail | An address per house, a privacy line, the "no email to send" line in `CLAUDE.md` turned around | Email as identity, in a village that runs on WhatsApp. `later.md` keeps WhatsApp one-time codes at "v2 or never" for Meta's fee | No, for now |
| I. **A native app**, iOS and Android, as a shell around the same web code (the owner asked 2026-10-08: "if we had a fullblown ios and android app, would that be solved well?") | Taps the link in WhatsApp; the app opens, signed in | Both platforms let an installed app claim a domain through a file the portal serves (Universal Links, App Links), so a link opens the app itself — **the only thing that solves the WhatsApp case on iOS**. The costs: an Apple developer account at 99 dollars a year and a review that needs a demo account (the demo village serves); a shell such as Capacitor — a dependency and a second build; push moved from web push to Apple's and Google's services inside it; and the claimed domains baked into the app, so every village on its own domain needs an app release, which argues for the cloud plan's one domain | Per-village naming on the home screen goes; one app for every village | The real answer to the iOS case, at the price of a second product. After A–C, and after the cloud plan decides the domain |

## A and B in detail (the owner asked, 2026-10-08)

**What a session is today.** The join answers with a token string. The page
puts it in `localStorage`, a drawer each browser keeps per site and only
page script writes, and sends it on every call as `Authorization: Bearer`.
The server keeps a hash of it in `devices`. Nothing expires it; it dies
when the drawer is emptied.

**A, the cookie.** The join answers with the same token as a cookie
instead: `Set-Cookie: potok=<token>; HttpOnly; Secure; SameSite=Strict;
Path=/; Max-Age=<a year>`. The browser then sends it by itself on every
call to this origin — the page's fetches default to `same-origin`
credentials, so `api.ts`, `photo.ts` and `push.ts` change nothing but
dropping the header. What that buys, and why each flag:

- *Server-set* is the point: Safari's seven-day eviction applies to what
  page script writes (`localStorage`, `document.cookie`) and not to a cookie
  the server set. The quiet-week logout on a non-installed iPhone goes.
- `HttpOnly`: page script cannot read it, so a script that got into the
  page cannot carry it off.
- `SameSite=Strict`: another site cannot make the browser send it, which is
  the CSRF surface a cookie opens and a header never had. A browser may
  withhold it on the first page load that arrives from a link — harmless
  either way, because `index.html` is public and the page's own calls
  after it are same-site and carry it.
- `Secure`: HTTPS only. The Vite dev proxy is plain `http://127.0.0.1`,
  which Chrome and Firefox treat as secure enough for the flag and Safari
  may not, so the flag follows the request's TLS, or dev in Safari is
  checked once.
- A year, under Chrome's 400-day cap. **A cookie is renewed only when the
  server sends it again**, so `/api/me` re-issues it on every load; without
  that line every phone lands at the gate a year to the day after the roll.
- `devices` is untouched: the cookie carries the token the hash already
  knows. No migration.

What it costs, each one small (as built: the re-issue on `/api/me` is the
exchange, so there is no `POST /api/session`, and the drawer is kept — a
browser that blocks cookies stays signed in by the header, and the CSP
already keeps page script to this origin, so `HttpOnly` loses little by
the token also sitting in `localStorage`): a route `DELETE /api/session`
for logout, because
`HttpOnly` means the page cannot clear it itself; the re-issue on
`/api/me` above; `auth.go` reading the cookie where there is no header,
**except at `/api/mcp`**, which reads the header only — the cookie rides
on every same-origin fetch, and the agent door must stay closed to a
phone's session; the demo door answering with the cookie too; tests keep
the header, which stays accepted. And one thing `HttpOnly` takes away: the
page can no longer tell gate from loading by looking in its drawer
(`getToken()` in `App.tsx`), so a fresh browser asks `/api/me` first and
sees the gate one round trip and one 401 later; the server clears the
cookie on that 401 so a dead one is not sent forever. Open and to be seen
on a phone, not assumed: whether the home-screen app on iOS shares
Safari's cookie jar (it has had its own storage since iOS 11.3; cookies
may or may not be in it).

**B, the pairing code as a link.** Houses → Add another device shows "883
559" for fifteen minutes, one use. `/api/join` already takes an invite and
a pairing code through the one route, and the gate already prefills a
code from `#/join/<code>`, which is how an invite link works. So the same
code as `https://<the portal>/#/join/883559` is a *Copy link* button beside
the code and nothing in the backend. The villager sends it to themselves —
AirDrop to the laptop, "message yourself" in WhatsApp and open it on
WhatsApp Desktop, which hands links to the system browser — and lands on
the gate with the code filled, names the device, taps Join. Same fifteen
minutes, same one use, same burn after five wrong guesses: the arithmetic
in `CLAUDE.md` § Codes does not move. On a phone the link has the same
trap as every link: tapped inside WhatsApp it opens WhatsApp's browser, so
B is for the laptop and for a browser the villager opens the link in
knowingly.

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
