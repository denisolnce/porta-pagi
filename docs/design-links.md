# A link to one entry: events, posts, needs, tools, and most things

Researched 2026-10-08 on the owner's ask, not built. The owner's words:
"make events, projects, maybe even market orders and tools, and basically
most things a link-shareable thing … our frontend and its urls might have
to have a significant overhaul for this to work … this feature would help
with portability, if users could send particular entries in whichever room
to others."

**The villager job, in one line: "are you coming Saturday?" pasted into the
WhatsApp group with a link that opens that work party, its answer buttons
under the thumb** — and the same for "I drive to Kočevje Thursday, add
what you need", "the chainsaw is in the shed, here", "read the new codex
section". Today the villager types "look in the portal, under Market" and
the neighbour opens Home and hunts. The answer first, then the argument.

## Answer first

- **No overhaul.** The portal is hash-routed (`HashRouter`, `main.tsx`), and
  a hash link is already how every push, the invite link and `#/codex`
  travel. A link to an entry is one more query parameter on the room the
  entry lives in — `#/tavern?event=12`, `#/market?run=5`, `#/shed?tool=3`
  — which the rooms half do already (`?at=board`, `?project=`, `?day=`).
  Switching to path routing would break every notification link on every
  phone (`CLAUDE.md` § Old notification links must keep working) and need a
  fallback for every path in `static.go`, for nothing a villager sees.
- **One resolver route, `#/go/<kind>/<id>`**, is what a Share button, a
  push and an agent emit, and the one place that knows which room a kind
  lives in and whether that room's switch is on. It redirects to the room's
  own address above. Ten lines in `App.tsx`.
- **A link is an address, not an access.** Nothing is public: an outsider
  who opens it sees the gate, and the gate HTML names the village and
  nothing of the entry, so a WhatsApp unfurl says *Vas Mokri Potok* for
  every link. That stays so by construction — never put an entry's words
  into the public `index.html`. "Send to others" means the other houses,
  the house's other phones, and a house's agent.
- **The sure payoffs come first**: a comment push opens the entry, not the
  room (`threads.go` sends `#/tavern` today, a project push already
  deep-links, and `sw.js` navigates the installed app); a tavern post or a
  comment can carry a link to an event or a run, read inside the app that
  holds the session; and an agent can hand its house a link ("here it is:
  …"), which the `initialize` instructions in `mcp.go` teach in one
  sentence, no new tool.
- **The WhatsApp group is the uncertain one, and the owner should hear it
  before picking a button.** A link tapped in the group opens WhatsApp's own
  browser, which holds no session (`isInApp` in `push.ts`; the install
  banner already tells a villager so), and on iOS no link ever opens the
  home-screen app. The first tap lands on the gate. The way through is the
  one the invite already takes: "open in browser", then that browser pairs
  once (a 6-digit code from a phone that is in) and keeps its token, so
  every later link opens the card. A cost paid once per browser, not per
  link — but a neighbour who never paid it reads "look under Market" faster.
- **A Share button on the card**, 🔗 beside 💬 Comments: the phone's share
  sheet (`navigator.share`, Android Chrome and iOS Safari both open WhatsApp
  from it), the clipboard on a laptop with "Link copied", the way the invite
  link is copied in the Houses room today.
- **Not on an away notice.** The portal does not help a notice travel; a
  villager who wants to tell the group types it. The owner decides.
- **One prerequisite**: a link pasted into a tavern post is not clickable.
  `Board.tsx` prints `p.body` as text. `#/codex` is documented as "its
  address for pasting into the Tavern" and does not work as one. Linkify
  `https://` runs in posts and comments, `noreferrer`, same-origin hash
  links opening in the app — a few lines in a `Prose`-like helper.

## What exists today

| Thing | Where | What it means for a link |
| --- | --- | --- |
| Hash routing | `main.tsx`, `App.tsx` `Routes` | Every address is `https://host/#/room?…`. The hash never reaches the server, so no CSP, no route in `static.go`, no backend change |
| Links already on phones | push `URL:` fields in `server.go`, `projects.go`, `threads.go`, `birthdays.go`, `news.go`; `sw.js` `notificationclick` navigates to them | The shape is proven in WhatsApp (the invite link, `#/join/<code>`) and in notifications. `#/bell` is kept alive for links sent before a merge — the same promise holds for any address shipped here |
| One per-entry route | `/projects/:id` (`Projects.tsx` `Project`, `GET /api/projects/{id}`) | The only kind with a page of its own. Its push already links to it |
| Query parameters rooms read | `Hall.tsx` `?at=board`; `Calendar.tsx` `?project=`, `?day=`; `App.tsx` and `Changelog.tsx` `?news` | The pattern a room follows to open one entry: read the parameter once, move the view, scroll the card into view |
| Cards | `className="card"` with `key={x.id}` in every room | None carries a DOM `id`; `?at=board` scrolls to a section. An entry link needs `id="event-12"` or a ref |
| The gate after login | `Gate` in `App.tsx`: `nav("/")` after join | A phone that is not yet in the village lands on Home and loses the entry. The demo door reloads the page, so the hash survives there |
| List windows | `listEvents` 60 days back; `listNeeds`, `listOffers` drop done rows after 7 days; `listPosts`, `listCamp` the last 500 | A link lives as long as the list shows the entry. An older one opens the room and nothing is there |
| Switches | `withoutOff` in `App.tsx` | A link into a switched-off room falls through to Home, silently, as the room's own address does today |
| The demo | `RunDemoReset` in `demodoor.go`, the hour's wipe | The seed writes into an empty database in one order, so ids repeat each hour and a demo link keeps opening the same entry. The landing could link straight into an event |

## Address options

| Option | Shape | Who changes | What breaks | Verdict |
| --- | --- | --- | --- | --- |
| A. A parameter on the room | `#/tavern?event=12`, `#/market?need=5` | Each room: one `useSearchParams` effect, an `id` on the card, scroll and open | An entry outside the list window shows nothing. The calendar must page to the event's month and pick its day (it has `?day=` for that) | **Recommend.** It is what three rooms do already |
| B. A page per entry | `#/events/12`, `#/tools/3`, like `/projects/:id` | A page component per kind, a `GET /api/<kind>/{id}` per kind (only projects has one), a back link | Fourteen pages for cards the rooms already draw. A window that applies to the list and not to the page is one more thing to explain | No, as the first step. Keep it for a kind where the window bites — events, whose 60-day cut is already an open decision in `later.md` |
| C. A resolver route | `#/go/event/12` → A's address | `App.tsx`, one map of kind → room and parameter | Nothing. If a room's switch is off the resolver can say so in one line instead of falling through to Home | **Recommend with A**: a push, a Share button and an agent emit one shape, and a room's address may change later without the links on phones dying |
| D. Path routing | `https://host/events/12` | The router, `static.go` fallback for every path, every push URL, `sw.js` | Every link already on a phone. The one-origin, embedded-frontend simplicity | No. This is the overhaul the owner feared, and it buys nothing |

## What a link does when it lands

- **Room on, entry in the list**: the room opens, pages its view to the entry
  where the room has a view (the calendar's month and day), scrolls the card
  into view and opens it — the thread open where the card has one, because a
  link in a conversation is usually about the conversation. A short
  highlight (the `.card` frame darkened for a second) says "this one" on a
  phone where the scroll is not obvious.
- **Room on, entry not in the list**: one faint line at the top of the room,
  *"That entry is not here any more — it may have been deleted."* No 404
  page: the room is still the right place, and the line is gone on the next
  tap.
- **Room off**: the resolver draws Home with one line, *"That room is
  switched off in this village."* A steward reads where to turn it on.
- **Not logged in**: the gate, as today, but after join the page goes to the
  entry, not Home — `Gate` keeps the hash it was opened on. The demo door
  already does, by reloading.
- **Tapped inside WhatsApp**: its webview, no session, the gate with the
  "you are inside WhatsApp's browser" banner above it. The villager opens
  the link in the browser that joined, or pairs this one once. On iOS the
  home-screen app is never the one a link opens; Safari is, with a token of
  its own once paired. The gate keeping the hash is what makes the detour
  land on the card and not on Home.
- **A reused id**: no table uses `AUTOINCREMENT`, so when the newest entry
  is deleted the next one written takes its id, and a stale link would open
  a different entry instead of the faint line. The resolver link carries
  the entry's creation day, `#/go/event/12/2026-10-08`, and the room
  compares it with `created_at` before it scrolls: a mismatch is "not here
  any more". No migration; `AUTOINCREMENT` would be a rebuild of every table.
- **A series** (a weekly work party): the link carries the day, `?event=12&day=2026-10-11`,
  because the answer buttons are per day (`occurs_on`). The Share button on
  an instance writes the day it was drawn on.

## Which kinds, ranked by how often a villager would paste one

| Kind | Address (option A) | The line a villager writes beside it | Note |
| --- | --- | --- | --- |
| Event, work party | `#/tavern?event=ID[&day=]` | "Are you coming Saturday?" | The strongest case: the answer buttons are what the link opens |
| Trip, order | `#/market?run=ID` | "I drive Thursday, add what you need" | The second: needs join a run on the card |
| Project | `#/projects/ID` | "Here is the roof plan, take a task" | Exists. Add the button |
| Post | `#/tavern?post=ID` | "The council notice, read it" | A pinned notice is the one the steward pastes |
| Tool | `#/shed?tool=ID` | "The chainsaw, in the shed" | |
| Need, give-away | `#/market?need=ID`, `?offer=ID` | "Anyone want these seeds?" | Done rows leave the list after 7 days, the link with them |
| Wish | `#/shed?wish=ID` | "Found a model, look" | |
| Codex section | `#/codex?section=ID` | "We agreed this, section 4" | `#/codex` is already the chip's address |
| Almanac row | `#/tavern?almanac=ID` | "Full moon Friday, who is up?" | Behind `celestial` |
| Birthday | `#/tavern?day=DATE` | "Ana's day, Tuesday" | Already works. No button: a person's data, and the card takes no answers |
| Camp row, due, duty | `#/camp?row=ID`, `#/dues?due=ID`, `#/roster?duty=ID` | rare | Counting rooms behind switches. Build when asked, the resolver makes each one a line |
| Contact | — | — | The number is the share, and the phone shares it. The card's three actions on one row at every phone width is measured (`CLAUDE.md` § the phone book); a fourth breaks it |
| Away notice | — | — | **Not offered**, owner's call. Burglary information travels by the villager's own typing, never by the portal's button. The push stays anonymous and so does the card |

## The Share button

- **Where**: a ghost 🔗 beside 💬 Comments in the card's action row, on the
  kinds above. Rooms with no Comments button (tools, posts) put it where
  Edit sits. It is `ghost` because it dismisses nothing and deletes
  nothing, but it is not an action on the entry either — `CLAUDE.md` §
  Buttons gives `ghost` to dismiss and delete only, so this is a fourth use
  or a plain button one size down (`lesser`). The owner picks.
- **What it does**: `navigator.share({ url })` where the browser has it —
  the system sheet, WhatsApp two taps away — else `navigator.clipboard` and
  "Link copied" for 1.5 s, the invite link's own behaviour in `Houses.tsx`.
  The text shared is the link alone: the villager writes the sentence.
- **The link is absolute**: `location.origin + location.pathname + "#/go/…"`,
  as `inviteLink` builds it, so it works pasted anywhere.
- **Strings**: "Share", "Link copied", the two landing lines above — in
  `sl.json` and `pt.json`, a Slovenian house to check them (`later.md`).

## What it costs

- `App.tsx`: the resolver route and its two lines of copy.
- Each room in the table: one effect reading its parameter, an `id` on the
  card, the creation-day check, the scroll and the open, the button. The
  calendar also moves its cursor and day. About the size of the `?project=` handling in
  `Calendar.tsx`, per room.
- `Gate`: remember the hash, navigate to it after join.
- `Board.tsx`, `Thread.tsx`: linkified bodies.
- `threads.go`, `server.go`: push URLs to `#/go/<kind>/<id>`;
  `banner_test.go` pins texts, not URLs, so nothing there moves.
- `mcp.go`: one sentence in the `initialize` instructions on the link
  shape. Tool count unchanged, `TestMCPSurface` untouched.
- A Portal News note: "a link to an event, a run or a tool, to paste into
  the group".
- No migration, no backend route, no dependency, no CSP change.

## What was considered and refused

- **A public preview** — an unfurl card with the event's title for WhatsApp.
  It needs an unauthenticated read of one entry, and `CLAUDE.md` § Nothing
  is public refuses the first such read on purpose (the camper self-check-in
  link, 2026-09-06). A title in a WhatsApp preview is also a title on a lock
  screen, which is the push line's whole argument.
- **A short code per entry** (`/s/ab12`) — a table, a route and a redirect
  for links that already fit a WhatsApp message.
- **A per-house share token** in the link, so an outsider could read one
  entry — that is a viewer role by the back door. Waits for the reduced
  role in `design-membership.md`, if ever.
- **Copying the entry's text** into the share, not only the link — the
  villager's own words say why they share; a dumped card reads as software.

## Open

- `ghost` or `lesser` for the button (above).
- Whether the away card gets one. Recommended no.
- Whether events get a page of their own (option B) once the 60-day window
  is decided.
