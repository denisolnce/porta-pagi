# CLAUDE.md

1. Don't assume. Don't hide confusion. Surface tradeoffs.
2. Minimum code that solves the problem. Nothing speculative.
3. Touch only what you must. Clean up only your own mess.
4. Define success criteria. Loop until verified (`task check` is the floor).

## What this is

Porta Pagi: the village portal one small Slovenian collective built for
itself and now offers to others — see `README.md` for the stack and how a
village runs it. The collective that built it runs it from its own deploy repo
(`mokri-potok-portal`), which holds that village's config, map seed and VM
notes; the decisions below were taken there, in that village's words, and
still hold here. It is a **social tool for a handful of houses**: every
feature must beat "scroll up in the WhatsApp group" or it does not ship.

## Invariants

- **Switches: a village turns some rooms on or off, one binary for all**
  (owner's decision 2026-09-23, the business plan's Decided block, kept privately in `porta-pagi-cloud/docs/commercial.md`). The list is
  `Features` in `backend/internal/httpapi/features.go`, each with a default;
  the defaults are the village that built the portal, and most are off. A
  row in `features` exists only for a switch a steward flipped, with who
  and when. **Off means absent**: `requireFeature` answers 404 on the
  feature's routes (inside `requireHouse`, so a key still meets the MCP
  fence first), a tool with its `Feature` is neither listed nor callable,
  its push kind is not offered in `GET /api/me/prefs` and rings nobody
  (`fanout`), and the page draws no door: a `ROOMS` entry with its
  `feature` has no building on Home, no nav item, and its routes fall
  through to Home (`withoutOff` in `App.tsx`). Off hides, never deletes:
  the rows stay, the export keeps them, and a house's or a steward's mute
  of that kind waits untouched for the switch to come back. **The steward's
  levers for the whole village live on one page**, `/settings` behind ⚙️ in
  the top bar (`rooms/Settings.tsx`, drawn and routed only for a steward,
  in the demo too): the switches with who flipped each and when, the
  village-wide notification mutes, and the export. Switching a counting room
  on asks once more, because it adds counting to a portal that counts
  nothing by default. What is about houses — creating them, invites, land,
  making stewards, deleting a house — stays in the Houses room. The switches today:
  `camp` (the Campground; on for a village that existed before switches,
  migration `025`), `dues` (a dues ledger) and `roster` (a duty roster with
  counts). The two counting rooms are the one place a total is allowed, and
  only because a village switched it on: the "do not add a total anywhere"
  rules below are the defaults, not exceptions to this one. The demo seeds
  every switch on.
- **A house is the account, with or without land.** Everyone in a house shares
  one identity; a device row per phone tells them apart. Never add per-person
  accounts without a decision recorded in the plan. **A member who lives here
  without land — renting a hut, using a vacant house part of the year — is an
  ordinary house whose parcel list is empty** (owner's decision 2026-09-06,
  option F in `docs/design-membership.md`). **No third kind of row**: a
  `kind='household'` would also make `ORDER BY kind, name` put them last in
  every list with nobody deciding it. What replaces the land on such a row is
  a mark on the map — `house_homes`, next bullet. **No sentence about where a
  house lives**: `houses.about` was dropped on 2026-09-08 (owner's decision),
  the mark is the answer. **The Houses room shows
  no parcel count**, in either block: a parcel list is a list, a count beside a
  name is a tally.
- **Marking a parcel somebody else holds says "we live here", never "this is
  ours"** (owner's decision 2026-09-07). In assign mode a tap on a free parcel
  assigns it as before; a tap on a parcel another house holds writes
  `house_homes` instead, so the crest joins that parcel on the map beside the
  holder's, and the Houses room reads *"Živi na zemlji hiše 🏠 …"* where a
  landholder has parcel numbers. It is never folded into `parcels`: a house
  that rents a hut holds no land and no list may say it does. The cost, paid
  knowingly: **a parcel no longer changes hands by assigning it to the new
  house** — clear it from the old one first.
  **The house itself marks where it lives** — `PUT /api/houses/{id}` takes
  `homes` from that house (a steward may write it too, because a steward keeps
  the map for houses that will not open a picker).
  A mark grants nothing: `homes` is not `house_parcels`, so a house cannot put
  itself on the map as a landholder, and a parcel it already holds is dropped
  rather than printed back at it. Marking a parcel nobody holds is allowed —
  living somewhere is a fact whether or not the app knows the owner yet.
  The 🗺 button in the Houses room that opens the picker **stays** (owner's
  decision 2026-09-08): dropping the sentence about where a house lives did not
  drop the mark, which is the answer that replaced it.
- **A stay ends by deleting the house** (owner's decision 2026-09-06), and only
  a steward can. **This is the one deletion the app allows, and the deliberate
  exception to "done is a state, never a deletion" below.** Deleting cascades
  through everything that house wrote — posts, its replies inside other houses'
  threads, events, sign-ups, needs, give-aways, away-notices, its tools, wishes,
  projects, tasks, camp rows — so **the confirm dialog names them, and says the
  honest thing about undo**: last night's `VACUUM INTO` backup can bring the
  house back, at the price of rolling the whole village back to it, so the
  dialog tells the steward to export first. Never make that button quieter, and
  never let it claim the loss is absolute — it is a day, not forever. Pictures
  the house put on projects stay with the project and lose only their house
  (owner's decision 2026-09-08). Going home
  for the season is not leaving: that is an away-notice, same as any house that
  winters elsewhere. The softer variant (`houses.left_at`) is designed and not
  built — `docs/design-membership.md` § Off-season, and ending a stay.
  A **common place** (`houses.kind = common`:
  event grounds, parking) shares the table because the map colours it, but it
  is land, not an account: no invite, no login, never offered where a house is
  meant (hand a task to, notifications, invite links). The UI keeps it in its
  own small block.
- **One origin.** The Go container serves the API *and* the built frontend
  (`static.go`, embedded at image build). The portal lives at its own domain,
  in `SITE.md` of the homestead repo. No second host and no CORS: in dev the
  Vite server proxies `/api` to the backend (`vite.config.ts`), so the page is
  same-origin there too.
- **No third party in the page.** Weather is fetched by the backend from the
  provider `WEATHER_PROVIDER` names — Open-Meteo by default, for any
  `WEATHER_COORDS`; ARSO by place name, this village — and trimmed
  (`weather.go`, cached 30 min), never framed. Both adapters emit one shape
  with a neutral icon key, so the page knows no provider. Both licences require
  naming the source, so the panel shows *Vir: ARSO* or *Vir: Open-Meteo.com*;
  the fetch sends a User-Agent with `PUBLIC_URL` as the contact. **The panel is a forecast for a town some
  kilometres away, not a measurement at the village** — the page says so, and
  it must never become a frost source for the homestead work. An iframe would hand
  the agency every villager's address on a logged-in page. Any future embed
  gets the same treatment or an argument written down here. **The fonts are
  served from this origin** (`frontend/public/fonts/`, SIL OFL, licences beside
  the files). **The page carries a CSP** (`static.go`, on `index.html` only):
  everything `'self'`, inline styles allowed because React paints house colours
  into style attributes, `blob:` images for photos and the export. The Go
  server sets it, not Caddy, so it ships and is tested with the frontend it
  describes; a village's edge proxy sets none on purpose.
  **A house colour is `#rrggbb` and nothing else** — CSS `background` also
  takes `url(...)`, so an unchecked colour is a beacon fired from every
  villager's browser; the backend answers anything else with a 400.
- **Nothing is public.** Every API route except `/api/healthz`, `/api/status`,
  `/api/bootstrap`, `/api/join` requires a bearer token. **No unauthenticated
  write surface, ever** — a camper self-check-in link from park4night was asked
  for on 2026-09-06 and refused for that reason. The one public image
  is `backdrop.jpg` behind the gate — a Google Maps screenshot; the owner
  keeps it as the default, in the demo and in his own village, knowing the
  licence is Google's and accepting the risk (owner's decision 2026-09-28).
  A steward replaces it from ⚙️ settings (`PUT /api/site/backdrop`, the page
  shrinks it to 2000 px) or goes back to the built-in one (`DELETE`); a
  server can also enter one by `/server site-import backdrop < picture`
  (`site.go`). Either way it is a row in `site_files` served in the built-in
  file's place, still without a token because the gate is before any login.
  The demo refuses both routes (`demoLocked`): one visitor's picture would
  greet every other visitor for the rest of the hour. Away-notices are
  burglary information: they never leave the logged-in app (no digests, no feeds).
  **An MCP key is the logged-in house** (owner's decision 2026-09-09): the
  model provider is a third party that house chose, the same argument that
  lets an away-notice sit in a WhatsApp group — so `list_away`, `create_away`,
  `update_away` and the `away` thread are on the agent surface (`mcp.go`).
  Say it the way the neighbour would hear it: **a neighbour's assistant may
  read your notice, as a neighbour's phone may.** The house that chose the
  provider is the reader, not the house that is away, so the village is owed
  the sentence — it belongs in the privacy note (`docs/later.md`), not only
  here. The
  "no digests, no feeds" half stands: nothing pushes or publishes an
  away-notice, an agent reads one only when its house asks, and the tool
  description tells it to keep what it read inside the conversation.
  **The away card carries the dates and the note its house wrote, and nothing
  else** (owner's decision 2026-09-08, when the dropped `houses.about` left the
  card a line short): the map mark was considered for that line and refused —
  the room answers who is away and until when, and a house's address is not
  part of the answer. It also carries a **thread**, like an event or a wish
  (owner's decision 2026-09-09) — "I fed them, key is back under the step" is a
  conversation and belongs where the notice is. **A card with no watcher says
  nothing** rather than printing an em dash: an empty line reads as a fault.
  **The Home tile counts who is away** (owner's decision 2026-09-09: every
  account is a villager, so a count on a logged-in page is not the leak a lock
  screen would be; narrow it when a reduced role arrives). It is two numbers,
  not one — out now, and told us they will be — because a house leaving in a
  fortnight is not away, and a tile that adds them together answers neither
  question. That split is the assistant's, not the owner's.
- **No tallies of favours.** "Taken by", "claimed by", "watched by" are
  acknowledgments. No counts, points, leaderboards, streaks. Ever.
- **Water on the map is one row and one component.** `map/Water.tsx` draws
  the `water` row of `map_files` — the watercourses chained out of an epsilon
  priority-flood + D8 model (the method of the homestead repo's
  `10-site/terrain-data/water.py`) run on ARSO DMR 1 m over **the whole village,
  E485000-488000 N44800-47600** — the box the map's own reset view opens on, so
  a course no longer stops in the middle of the page. Drawn: a catchment of
  **2 ha** or more and at least **250 m** long. That floor is set from below by
  the collective's own stream, which peaks at 2.36 ha — raise it and the plot's
  east boundary loses its water. **Named: 100 ha and above** (`stream_m2`), which
  is three courses and is the whole readability budget; the other 29 are drawn
  thin and **left unnamed on purpose**, because 1 m bare earth says how much
  land drains into a channel and never whether it runs. So there is **no
  *grapa* on the map any more** — the old rule named the plot's east course
  *potok* and the west one *grapa*, and at village scale the west course
  (0.34 ha, an eroded road) is below the floor and is not drawn at all.
  **The village is a closed karst basin** (outlet 525.8 m, floor 451.5 m, per
  `rimclose.py` in the homestead repo), so the fill is **capped at 2 m** and a
  real sink stays an outlet. Do not lift that cap: an uncapped priority-flood
  fills the basin to its rim, drowns every channel in one flat, and returns a
  confident hairball. **Do not put names on the lines.** OSM and the 2013
  survey disagree about this water — OSM calls the valley trunk *Reka* and puts
  *Mokri potok* 1.6 km west, `10-site/geodetic-survey-2013.md` reads *Mokri
  potok* on the trunk itself — and the owner has not picked. To take the water
  off the map, delete `Water.tsx` and the `<Water/>` line in `VillageMap.tsx` —
  nothing else knows about it. A village with no `water` row draws none and
  says nothing: the water is this village's model, a map is its parcels.
- **Cadastre is a view, not a source.** The `parcels` row of `map_files` is
  public GURS data for the village that built this, entered once from a file by
  `/server map-import` (`maps.go`) — **in the database like the photos, so the
  backup is the whole village and one image serves any village**; served only
  to a token, with an ETag so a phone holding it gets a 304. The import refuses
  anything but a FeatureCollection, because a bad upload would open every phone
  on an empty map. **A village with no `parcels` row sees what it needs**, in
  words, instead of a map. Parcel numbers show only for assigned parcels (and
  to stewards in assign mode). The caption under the map is **one line**: the
  snapshot date and the source, both typed at import and kept on the row.
  The "boundaries, not fences" sentence and the water-model disclaimer were cut
  on 2026-09-07 (owner's decision) — a caption nobody finishes reading protects
  nobody. The attribution itself stays, and an outline nobody holds is drawn at
  half stroke so the village reads out of the cadastre mesh.
  House ↔ parcel assignment lives in the DB only. Licence attribution for the
  GURS data: `TBD`, required before the portal moves to the collective's domain.
- **One thread implementation.** Comments live in `comments`, keyed by
  `(subject, subject_id)`, and are rendered by `Thread.tsx` everywhere: an
  event, a wish, an away notice, a contact. One reply level. The button that
  opens one says 💬 Comments and carries the count **only when there is one**,
  in every room. A new room that wants comments
  adds a subject, never a table. The push kind stays the room's own so no
  opt-out changes meaning. Who hears is the subject's own answer, in
  `tellThread`: an event tells the caller and everyone who answered yes or
  maybe, a wish tells the wisher and everyone who wants one, **a contact tells
  nobody at all** because the phone book has no push kind of its own, and **an
  away notice tells only the house that is away and its watcher, and tells them
  nothing** — the banner is the anonymous watchtower one, because a comment's
  snippet on a lock screen would say what the notice itself is not allowed to.
  The name field starts filled from the device label this phone joined under,
  and stays optional.
- **Wish options are findings, not votes.** Anyone may add "this model, this
  price, this link" to a wish. Never count, rank or mark a winner.
- **A past event leaves the list, never the calendar.** The list under the month
  grid holds **that month's** events that are not over — not everything ahead
  forever — so it cannot grow past a month's worth, and ‹ › pages it with the
  grid. What has passed stays in its day cell, one click away. `isOver` in
  `rooms/shared.tsx` decides: an event is over when its **end** has passed, and
  an event with no end lasts until the end of its day, because dropping a work
  party a minute after it starts hides it from whoever is running late. The Home
  badge, the tavern peek and a project's two event groups use the same function
  — never re-derive it with a date comparison, which counts this morning's
  finished event as ahead; the project page did exactly that until 2026-09-08.
  It takes no clock argument, so it cannot be handed an array index by
  `list.some`. **An event that is over wears `.faded`** and loses its kind
  colour from the left edge (`EventCard.tsx`), so the past reads as past in
  every room at once — a day cell, a project, the tavern. `.card.faded` is the
  one fade shared by a closed task, a done project, a handed camp row and a
  past event; a done need or give-away still fades further on its own
  (`Market.tsx`), and `.card.tool.out` is a tool on loan, which is a state and
  not a finished thing.
- **An answer is not a headcount.** A sign-up is `yes`, `no` or `maybe`, and
  silence is a fourth thing. **The house that creates an event answers `yes` for
  itself in the same request** — whoever calls a work party is at it, and a
  headcount of nobody beside an event somebody called reads as a failure that
  did not happen. It is an ordinary answer: the caller can change it or take it
  back like any house. Only `yes` is counted; never fold `maybe` into it.
  It carries **no note** — the sign-up note was removed on 2026-09-06 and the
  old ones moved into the event's thread (`013_quiet_and_notes.sql`), because a
  line only the signer can edit and nobody can answer is a worse comment. A
  POST without a valid state is a 400, never a yes.
  Moving an event's time bumps `events.time_version`, which marks every earlier
  answer stale — a headcount for a day that no longer exists is worse than none.
  **An event that is over shows no answer buttons** (owner's decision
  2026-09-09): the three go once `isOver`, because a headcount for a day that
  already happened counts nobody and a faded card you can still click reads as
  a room that did not notice. The answers already given stay listed with their
  crests, and the **thread stays open** — what a work party was like is written
  after it, not before. The backend still accepts a late POST, and must keep
  doing so: `over` is read once when the card renders and no clock re-renders
  it, so a page opened before the work party ended still carries the buttons
  afterwards, and that answer was given in good faith.
- **Any house may edit an event — provisional.** Owner's decision 2026-09-06:
  every account today is a villager. `events.edited_by` records who, and the
  room shows it. When a reduced "viewer" role arrives for volunteers, this
  narrows to the creator and stewards.
- **No tally of favours.** Work-bee sign-ups are a headcount for one day and
  die with the event. Tool loans record who holds a tool now, never how many
  times a house borrowed. Do not add a total anywhere. The **return reminder**
  (`remind.go`) nudges the holder after one day and then with doubling gaps
  (day 1, 3, 7, 15), read off two timestamps — never a counter, never the
  owner, never in quiet hours. A cap "after N reminders" would need a count;
  do not add one.
- **Photos stay in the database.** A tool photo, or a picture on a project, is
  shrunk in the browser and stored as a BLOB (`photos.go` reads and serves
  both), so the SQLite backup is the whole village. Lists and the export never
  carry the bytes; `GET /api/tools/{id}/photo` and `GET /api/photos/{id}` need
  a token. The export hardcodes the `tools`, `project_photos`, `map_files` and
  `site_files` column lists — a new column on any must be added there too, or it silently
  drops out of the exit path. The map rows go out as text, unlike the photos:
  a village leaving with the JSON and no backup keeps its cadastre. **Any house may put a picture on any project** — a project is
  the village's, like its events — and the full-size view says which house and
  when; the house that added it, the project's house or a steward takes it
  down. The input opens the chooser, not the camera: before-and-after pictures
  are already in the gallery. **The full-size view walks the strip** (owner's
  decision 2026-09-08): the arrow keys on a computer, a swipe on a phone, and
  the ‹ › buttons that make both discoverable. It never wraps around — each
  button is absent at its end of the strip, so the end is visible instead of
  looping back unannounced. A swipe must be **more sideways than vertical**,
  and the view sets `touch-action: none`, because a finger trying to scroll
  must neither move the strip nor drag the page behind it. The picture on
  screen stays until the next one has loaded: an empty frame reads as a
  closed or broken view on a slow phone.
- **Wishlist names are not votes.** Never sort, badge or count by how many
  houses want a thing. A wish ends when the wisher marks it arrived.
- **A project is planned, in progress or finished, and a new one begins
  planned** (owner's decision 2026-09-09). Its creator or a steward steps it
  forward or back, one step at a time in the UI, any step in the API. The DB
  value for in progress stays `open`, older than the planned state, so no row
  moved. The cost, paid knowingly: one more tap before Saturday's work party
  counts on Home. The creation push says "plans", not "starts", and is pinned
  in `banner_test.go`. Nothing pushes on a flip. **Only a project in
  progress counts on the Home badge**, and only its free tasks; a planned
  project's tasks may still be taken — the badge is what a plan stays off,
  not the hands. The event form offers every project that is not finished, a
  planned one included: the work party that starts a project is exactly the
  event it needs.
- **A task is taken by any house, or handed to one by its creator.** Any
  house takes a free task and lets it go. The task's creator, the project's
  creator or a steward may assign a house (agreed in real life first) or clear
  the holder — the assigned house is told (owner's decision 2026-09-05, evening,
  reversing the afternoon's "never assigned": the reminder that will one day
  read `due_at` replaces the assigner nagging in person). No other house
  touches `assigned_to`. The project's creator hears when a task is taken. "3 of 5
  done" is a project's progress; never show a house's count of tasks. The list
  also tallies a project's **pictures and events on the right of its summary
  line** — every event, finished ones too, because the tally says how much the
  job carries and not what is left to do. That is a count of the work, never of
  a house, so it is no exception to "no tallies of favours"; a zero is left off
  the line rather than printed.
- **The campground holds no amounts.** One row = one camper's stay: a house
  noticed it (arrived), a house has the money (held), it reached the box
  (handed). A tick on arrival ("I already have the money") lands the row in
  handed at once — to be revisited. A note is optional. No `amount` column, no sums, no plates
  in the label ("grey camper", "family from NL"). The cash box is the ledger.
  Who the money is handed *to* is `TBD` — the treasurer question in
  `docs/design-more-rooms.md`. Retention of `from_who` (a steward clears labels
  older than 12 months, by a button) is `TBD` and not built; a privacy note
  decides it. The `camp` push carries the house and the camper label, never
  the note. **The Campground is on the agent surface** (owner's decision
  2026-09-09): `list_camp`, `create_camp`, `update_camp` run the same handlers,
  and because the handler refuses no label, the tool description carries the
  rule — one row is one stay, no amount, no total, no plate or nationality.
- **The dues ledger counts a due, never a house** (behind the `dues` switch,
  off by default; `docs/design-more-rooms.md` § Dues ledger). A due is a
  title, an optional amount per house (cents, with the currency as the
  village writes it), an optional date, and one row per house asked — never
  a common place. **A steward writes it and names the house that collects
  it; that house or a steward marks a house paid**, and the row keeps who
  marked it and when. A house never marks itself. Every house sees every
  due, who paid, and **the due's totals, collected against expected** —
  that is the ledger, and the one total besides the roster's. **No sum of a
  house across dues, no balance a house owes**: the query is one `GROUP BY`
  away, so the rule lives here. **The portal records, it never moves
  money**: no payment link, no account-number field; how to pay is the
  notes, in the village's words. **Nothing pushes** — money should not buzz
  phones. A house that is deleted leaves all its rows without a name, paid and
  unpaid (`026_dues.sql`): deleting a house never forgives a debt, and an
  edit of the due's houses does not either, so the collected
  total never drops under money already in the box; a house that has paid
  cannot be dropped from a due until it is marked unpaid. `list_dues` and
  `mark_due` are its agent tools, gone with the switch.
- **The phone book keeps a number, not a file on a person.** Contacts is a
  name, a phone, a note and a type. **Any house writes one down and any house
  corrects it** — the same provisional footing as editing an event — and the
  card **names only the house that last changed it** (owner's decision
  2026-09-10): who wrote a number down is in the row for the delete check and
  nowhere on the card, because the number is the village's and not that
  house's. Removing one
  is the adder's or a steward's, and a contact **outlives the house that added
  it** (`ON DELETE SET NULL`, like a project picture), so an orphaned number is
  a steward's to take away. **The type is one free word and there is no table
  of types**: the picker offers the words the rows already carry, narrowed the
  way fzf narrows — the letters in order, a run of them and the start of a word
  worth more — and a word nobody has used *is* a new type the moment it is
  written. A type therefore exists exactly as long as a row wears it. On the
  card the type is **a pill beside the name**, not a heading over a group
  (owner's decision 2026-09-10). **The book is alphabetical, under its
  letters** (owner's ask 2026-09-28): the page sorts by name with
  `Intl.Collator` in the page's language and draws a large, faint letter on a
  rule over each letter's cards — on the rule, not in a column, so no card
  gives up width on a phone. The letter is the language's: Á files under A in
  Portuguese, Č, Š and Ž are letters of their own in Slovenian (ICU's
  collation, no table of ours), and a name that starts with a digit or a
  symbol files under "#", last, as a phone's own contacts do. The sort is one
  function (`sortContacts`), so ordering by date later is one more comparator.
  **One search box narrows it over name and type at once** the same fzf way,
  because a villager remembers the trade or the name and rarely both, and the
  letters are drawn over what survives. Each card carries **one faint date
  line**: "Added" and the day it was written down, or "Updated", the day and
  the house that last corrected it. The
  backend folds a spelling that differs only in case into the one already in
  use, because two spellings of one word split the group with nobody deciding
  it. **Nothing in the room pushes** (owner's decision 2026-09-10): a number is
  looked up when the pipe bursts, not announced, and a tenth kind in every
  house's list buys nobody anything. Its thread is silent for the same reason —
  borrowing another room's kind would make somebody's opt-out mean what they
  did not choose. The cost, paid knowingly: "this number is dead" waits in the
  room until somebody opens it. **The thread is for the number, not for the
  person**: whether it still works, who came last, what to ask for. The
  tradesman is not in the room to answer, so a room that collected "overcharged
  us" would be a ratings board about somebody who never joined — the invariant
  and the tool description say so, and nothing enforces it. The Home tile
  carries **no count**: how many numbers the village keeps is not a question
  anybody has. Comments, Edit and Delete **stay on one row at every width a
  phone has**: three actions that wrap read as two groups, and the third looks
  like it belongs to something else. What that costs in padding and face, and
  the longest row it was measured against, is in `styles.css`.
- **The duty roster counts turns, and only because a village switched it on**
  (`roster`, off by default; `roster.go`, `rooms/Roster.tsx`, design and the
  options rejected in `docs/design-more-rooms.md` § Duty roster). A turn is
  one house, one duty, one date, planned or done; **only done counts**. The
  count is done turns per house per duty over **the last twelve months,
  rolling** — one whole cycle of every seasonal chore, and nobody banks credit
  or carries a debt forever. **Houses in the village's order, never by count;
  no sum across duties** — a pump check is not a mowing. **A steward writes
  the duties down** (the list of what is counted is the village's agreement,
  not one house's), retires one, and deletes one only while nobody has done
  it. **A house writes only its own turns, and nobody else touches them, a
  steward included** — a count a steward could lower is a count nobody trusts.
  A duty may carry a cadence (`every_days`) and shows "next around" off its
  last done turn, or off the day it was written down before the first; **nothing assigns a house to a date and nothing reminds**.
  **A turn that did not happen is not "missed"**: it is not counted, and after
  its day only its own house sees it. A house leaving keeps its done turns in
  the log without its name (SET NULL), so next due does not jump; the table
  counts houses that exist. **Nothing pushes.** Kept apart from work parties,
  whose `yes` stays a headcount for one day in every village.
- **Done is a state, never a deletion.** Finished projects and closed tasks
  stay readable with their closing notes. Nothing archives itself.
- **Exit is designed.** `GET /api/export` (steward, the 📦 button on the
  settings page) dumps everything as JSON, a switched-off room's rows too;
  `VACUUM INTO` backups land nightly in `${DATA_DIR}/backups/`. Keep both working.
- **No secrets in this repo, and none needed.** The first steward code is
  generated on first boot and printed to the container log. The VAPID key pair
  for web push is generated on first use and kept in the `settings` table.
  The one secret a village may add, `RESEND_API_KEY`, is optional and lives in
  the deploy's environment: without it feedback goes to the log.
- **Push is opt-in per phone, filtered per house, mutable village-wide.** A
  phone may register only an `https://` endpoint on a dotted hostname —
  the backend POSTs to whatever it is given, and what actually keeps it off a
  compose neighbour is TLS verification, which the hostname check just
  fails early. A
  phone subscribes after the villager taps Allow and records the language it
  subscribed in; the house switches kinds off in `notify_off` (empty = all
  offered kinds on — a switched-off feature's kind is not offered); a steward mutes a kind for everyone in `notify_off_global`
  from the settings page, which records who and when, shown to every house in
  its own Notifications panel. Both lists are checked on
  every send. **Nothing is exempt**: the alarm kind was removed on 2026-09-06
  because a real emergency is a phone call, and a notification nobody is holding
  is worse than none. An event is `event` or `work`, and that is all. The author's
  house never receives its own event. **A run's riders hear when it moves and
  when it is called off** (owner's decision 2026-09-08): editing a run's place
  or time, or deleting the run, pushes to the houses whose open needs sit on
  it, and to nobody else — not the driver, not the editor; a notes edit rings
  nobody. Both banners name the driver, but a **cancellation names it as a
  label, not as an actor** — a steward may be the one who deleted the row, and
  "Zeleni Volk calls it off" would blame a house for something it did not do; moving
  a run is neutral enough to keep the active form. Deleting needs its own
  handler (`deleteRun`, not `deleteRow`) because the riders have to be read
  before the row goes: `needs.run_id` is `ON DELETE SET NULL`, so the need
  outlives the car, and the banner says so rather than letting a house think
  its need went too. Push carries a title, a one-line snippet and a route, in
  human words — `banner_test.go` pins every creation banner and
  `TestMarketEdits` the run-change and run-cancelled ones, so read them before
  changing copy. **A run is a trip or an order** (`runs.kind`, migration
  `028`): one house, a cut-off, the needs that join it. A trip 🚗 drives
  somewhere and leaves; an order 📦 is placed with a shop, a catalogue or a
  supplier and closes. Only the words differ — card, form and all three
  banners (`runWords` in `server.go`, pinned in `TestRunKinds`) — and an
  order never "drives to". **The app records the order, never the money**: no
  amount column, no shares, no who-owes-whom. Both ring under the one `runs`
  kind, so its opt-out reads "Runs & orders". Flipping the kind rings the
  riders like a move does, in the new kind's words: milk on a car that is
  back tonight is not milk in a delivery that comes next week.
- **What a lock screen may say about an empty house.** Anyone holding a phone
  can read a notification. The line: a **multi-day absence is anonymous** — an
  away push names no house, no dates, no notes, only "new notice, open the
  Watchtower", and a comment on that notice rings the same way, with no author
  and no snippet. A **scheduled short trip is named** — a shop run says who drives
  and when, because the house is coming back the same day and the whole point
  is to answer it; an order says who orders and until when, and nobody
  leaves home for it. Classify a new kind against that line; do not guess.
  The one other exception is the optional author name on a tavern post, which
  the poster typed themselves.
- **Quiet hours: 21:00–07:00 nothing rings, unless a phone asked.** Everything
  waits in the app rather than buzzing a neighbour at night. `s.now()` decides,
  so tests can move the clock. The one way through is `devices.quiet_ok`, set by
  that phone alone (`PUT /api/me/device`): the flag is on the **device**, not the
  house, because the phone on a bedside table belongs to a person and the other
  phones of that house did not choose. `nightFilter()` narrows the recipients
  instead of dropping the send, and a subscription with no device row behind it
  stays silent — the default is quiet. **Tool return reminders are the
  exception to the exception**: `remind.go` still skips the whole night cycle,
  so the nudge lands in the morning. A nudge is not news.
- **The way back in.** When nobody holds a session — a steward's phone wiped,
  the origin changed — `docker exec <container> /server code "<house>"` on the
  VM rotates that house's invite and prints the link (`cli.go`).
  SSH to the machine is the credential; there is no in-app password to reset
  and no email to send. Keep it that way.
- **A door for agents: `POST /api/mcp`, and a key that is a device.** The
  portal's API as MCP tools (`mcp.go`), so a house tells its assistant and it
  is in the portal — the owner's agent and a villager's alike. JSON-RPC by
  hand, no dependency, same origin, tools only (no OAuth, so claude.ai's web
  connectors cannot connect; Claude Code and any client that takes a header
  can). **A key is a device row with `agent = 1`**, minted by
  `POST /api/devices/agent` from the Houses room (`+ MCP key`), `potok_` + 64
  hex, hashed like a phone's token, listed in Your devices with 🤖, revoked
  with the same Remove button, gone with the house. **Shown once**: the portal
  keeps the hash and cannot show it again. **A key opens `/api/mcp` and
  nothing else**: `requireHouse` answers 403 to an agent device on every other
  route unless the request carries the `viaMCP` context mark the dispatcher
  sets — so the tool table in `mcp.go` is the whole surface (51 tools, pinned
  by `TestMCPSurface`: reads and writes in every room, a project picture in,
  **no deletes**, no devices, pairing, push, house row or steward route), and
  the raw REST API is closed to a key on purpose. **And the door takes a key
  and nothing else**: a phone's own session token is answered 403 at
  `/api/mcp`, which is the other half of the same fence — without it a steward
  could paste what its browser holds into an agent and edit every house's rows
  through `ownerOrSteward`, and "a key carries no stewardship" would have a
  side door nobody wrote down. **A tool argument that lands in a path may not
  be `.`, `..` or carry `/?#%`**, and a status of 300 or more is a tool error:
  Go's mux answers an unclean path with a bodiless 301, which would otherwise
  read back to the model as a write that landed. **A key carries no
  stewardship for now** — `IsSteward` is false for an agent, whatever its
  house; the owner said a key might carry it later, so that is an open
  possibility, not a closed question. **A tool call is the existing handler
  run in-process**: every check and every push as for a phone, no second
  write path, and the handler's own error text is the tool's. Nothing marks a
  row as written by an agent — the house answers for its key as for its
  phones. `author` stays empty unless the person dictated the words; the tool
  descriptions and the `initialize` instructions say so. The MCP body cap is
  3 MiB, because a picture rides in as base64, and `readPhoto` still caps the
  decoded bytes at 2 MB. No rate limit: an agent in a loop is a house in a
  loop, and Remove is the brake. No expiry. Tool descriptions are English —
  a model reads them, not a villager — the one place `t()` does not apply.
  Rationale and the options rejected: `docs/design-mcp.md`.
- **Codes.** A steward's invite is 10 characters and multi-use for 14 days — it
  travels through WhatsApp and lets a whole house in. A pairing code is 6
  digits, single use, 15 minutes, and only ever adds one more phone to a house
  that is already inside. The small keyspace is paid for by a per-IP cap on
  `/api/join` and by burning every live pairing code after five wrong guesses.
  Do not widen either without redoing that arithmetic.
- **One date control.** Every date or date-time in the app is `DatePicker.tsx`
  (parchment button + month grid in the calendar's own classes). Never a native
  `type=date` / `datetime-local` again — they look foreign and cannot prefill a
  month without a day. Values stay plain strings: `YYYY-MM-DD` or
  `YYYY-MM-DDTHH:MM`. The far end of a range takes `min` (the near end's value,
  same shape): days and hours before it are **dead cells, never a silent
  clamp** — a picker that moves the day you pressed is worse than one that shows
  the door is shut. Moving the near end past the far end drags the far end
  along, because only the field you touched may change under you. `badRange()`
  in the backend refuses a backwards pair whatever the client sends. The popup
  cancels its clicks' default action: every room puts it inside a `<label>`, and
  a label forwards a click to its first control — here the field button, which
  reopens the popup you just closed. It also anchors to the field's right edge
  when the left would hang it off a narrow screen; a popup past the edge makes
  the page pan sideways and taps get eaten as scrolls.
- **Two clocks live in the database, and they never compare directly.** A time a
  villager typed is local wall clock, `YYYY-MM-DDTHH:MM`. Anything
  `datetime('now')` wrote is UTC with a space and seconds — SQLite ignores the
  container's `TZ`, which only reaches Go's `time.Now()` (so quiet hours are
  local). `When` in `rooms/shared.tsx` tells the two apart by that space and
  appends the `Z`; a bare stamp reads as local and shows the village two hours
  early in summer. SQL comparing a wall-clock column to now needs `strftime` in
  that column's own shape — `starts_at >= datetime('now')` compares `'T'` with
  `' '` and quietly answers wrong.
- **The button frame is the house style, not only the button.** A weather
  forecast day wears `.lesser`'s border, fill and thin shadow (`.w-day`) and is
  not clickable: no hover, no press. Reuse the frame where a small box needs to
  belong; never reuse the affordance.
- **Buttons come in three weights.** `primary` for the one action a card or
  form is for, plain for the rest, `lesser` for inner additions (add a task, add
  a note) — a real button one size down, never a ghost link. `ghost` is for
  dismiss and delete only.
- **The village's language first.** `LANGUAGES` lists a village's languages
  in its order, the first is its default (`sl,en` here); the top bar's buttons
  and a phone's push language come from that list. Every user-facing string
  goes through `t()` in `frontend/src/i18n.tsx`, and every push text — and the
  manifest's one line — through `tr(lang, …)` in `i18n.go` — **both read the
  same dictionary**,
  `backend/internal/httpapi/i18n/<lang>.json`, keyed by the English phrase, so
  a phrase is translated once. English is the key and has no file. A new
  language is one new file and nothing else; a phrase with no entry reads as
  English. The dictionary lives with the backend because Go can embed only
  below its own module; Vite reaches it through the `@i18n` alias.
- **The bottom bar fits five.** Village plus four rooms — Hall, Projects,
  Market, Shed (owner's pick 2026-09-05). A new room either replaces one, merges
  into one, or lives off the bar like Watchtower, Campground, Houses and
  Contacts do — tiles on the Home map, two taps from anywhere. The Codex and the Changelog
  are not rooms: they are chips on Home, at its head and at its foot. Projects has a second door: the
  📋 chip on a calendar event. Rooms carry a `short` label for
  the bar because a phone gives each item about 60 px. Rationale and the
  options rejected: `docs/design-more-rooms.md`.
- **A tavern peek row spends its width on the title — provisional.** A phone
  gives that row about 20 characters before `HallPeek` cuts the title to dots,
  so what shares the line has to answer a villager's question: **a work bee
  keeps the crest of the house that called it** — who is asking for hands is
  half the line — **and an ordinary event drops it** (owner's decision
  2026-09-09), because the room, not the caller, is why anyone reads that row.
  The crest may leave the work bee too; the owner kept it for now. The time and
  the answer tally shrink a step under 560 px and sit closer to the title, for
  the same reason. How the cut itself is measured: the comment in `HallPeek`.
- **Old notification links must keep working.** Payload URLs live in the
  database of no one — they are already on people's phones. `/bell` survives as
  a route alias after the merge; a post links to `#/tavern?at=board`.
- **The Codex is the village's text, so it lives in the database, never in
  the repo** (owner's decision 2026-09-09). `codex_sections` holds the founding
  document — values and the agreements the council adopted — as ordered
  sections whose words live in `codex_texts`, one row per language of the
  village (`LANGUAGES`, so a third language is a row and not a migration);
  the page shows the reader's language and borrows the first of the village's
  that has text, saying which. The API and the seed carry `texts` — `{"sl":
  {"title", "body"}, …}` — and refuse a language the village does not speak. **Any
  house edits a section or adds one**, on the same provisional footing as
  editing an event, and every section names the house that last wrote it and
  the day — a text anyone may change must show who did. `updated_by IS NULL`
  reads as *"adopted by the village council"* with the section's date. **A
  steward removes a section**, and the nightly backup is its undo. A section
  outlives the house that edited it (`ON DELETE SET NULL`), like a project
  picture. **Nothing pushes**: a changed codex is announced in the Tavern by
  the house that changed it, in its own words. The adopted text enters once,
  by `/server codex-import < codex.json` on the VM (`codex.go` has the shape),
  stamped with the adoption date and no house;
  it refuses a codex that already has sections. The repo is public and the
  document is the collective's, so the seed file stays off the repo. The
  export carries the table. Bodies are plain text with two habits and no
  markdown library (`Prose` in `Codex.tsx`): a blank line splits paragraphs, a
  block of `- ` lines is a list, and a bullet's words before its first colon
  are its lead, set in bold. **Two houses on one section**: the form sends the
  `rev` it opened on and the update is one compare-and-set statement, so a PUT
  with an older rev is a **409** and lands nothing; the page then shows their
  text above the form and keeps this one, so the next save replaces theirs
  knowingly. A counter, not the stamp: a stamp has seconds. A PUT without a
  rev is taken as it is.
  **Where it lives**: the fourth chip of the Home row — Map, Weather, Tavern,
  Codex — so a device that keeps it on top opens the portal on the village's
  values; `#/codex` is its address for pasting into the Tavern, not a door.
- **The demo village is invented ground, seeded once, dated off the day it
  runs.** `/server demo-seed` (`demo.go`) writes `demo/seed.json` — an
  invented Alentejo collective, *Vale do Sobreiro Torto*, ten houses and a
  square — into an **empty** database only, and refuses one with houses. It
  shows every feature a visitor can open: pinned posts and replies, both
  kinds of event with yes, no and maybe, an event whose time moved with its
  struck answers, events on a project's task, a trip and an order with needs
  on them, needs in every state and give-aways in every state and tag the
  pickers still offer, absences now, soon and just over
  (one watched and talked about, one with nobody watching), tools in the
  shed and one out over a week, wishes with findings and threads, projects
  planned, in progress and finished with tasks free, held by another house
  and done, camp rows in all three states, a phone book written down over a
  year with corrections and threads, a codex with one section a house
  rewrote, dues (one partly paid with a payment a steward marked, one all
  paid last quarter, one with no amount) and a roster (a duty due, one not
  yet, one done when needed, a year of turns with the newcomer's few, plans
  ahead). A due's title may say `{trimestre}`, `{trimestre passado}` or
  `{ano}`, filled from the day it seeds. A thread on every subject `threads.go` takes, arguments in some.
  **Dates**: a plain offset for what could happen in any month; a calendar
  anchor (`on`, most recent or `ahead`, with `keep` days of grace) for what
  the land decides; `at` a keyed event or project for what hangs off one; a
  row inside another counts from its parent. **Words are never written in the
  future** — a line whose moment has not come is not seeded — so the demo
  never reads as abandoned, never picks grapes in March, and a harvest's
  thread appears after the harvest. `TestDemoAnyDay` seeds it twice a month
  through a year and proves every room shows its features each time. The
  seam: an anchored absence does not silence its house's other rows, so
  around the year end a house abroad is also at the film night. The
  cadastre and the water are
  synthetic (`demo/parcels.geojson`, `demo/water.json`, generated by a
  jittered grid): **nobody's land, never a real village's parcels**, and every
  house, name and phone number is invented (`000` patterns). Real places
  around it may be named — a town, its fire brigade, its health centre, its
  post office — because a village in the Alentejo has them and an evaluator
  should see a real phone book; a private business or a person never is,
  not even a first name with a town after it. The words are
  the writing job: **a bilingual village, the way one in the Alentejo is** —
  six Portuguese houses writing pt-PT (one never in English), a Scottish, a
  German and a French house writing English, and a mixed Dutch-Portuguese
  house writing both, the author field telling who; someone translates the
  key line for the others. Every room carries enough English for an
  evaluator and enough Portuguese to be plainly Portugal. Each house keeps
  its voice as in a WhatsApp group; a line that reads like software is a
  bug. The demo's languages are `pt,en` (`pt.json` is the
  dictionary that came with it). **The door with no code exists only in the
  demo build** (`demodoor.go`, `//go:build demo`): `GET /api/demo/houses` and
  `POST /api/demo/enter` hand a visitor a device token for any house, the
  page's `{{DEMO}}` meta tag turns the house link into a switcher, and the
  village is wiped and seeded again on the hour, and the demo build seeds
  itself on boot when the ground is empty (its data dir is a tmpfs). In every village's binary
  that file is not compiled — `demodoor_test.go` pins the 404 — and CI ships
  the demo as its own image, `ghcr.io/dennislapchenko/porta-pagi-demo`, so no
  village can pull the door by mistake. Not seeded: project pictures and tool
  photos, because a picture of an invented village would have to be
  fabricated; the strips stay empty.
- **The changelog is the git log, filtered, written at build time.**
  `frontend/scripts/changelog.mjs` writes `public/changelog.json` before every
  build and dev start — `built` is HEAD's date, `entries` are the last **90
  days** of commit subjects minus deploy rolls, docs, tooling, reverts and
  merges — and the panel groups them by day. A window, not a history: the
  panel sits at the foot of Home and a list with no end is nobody's answer.
  CI writes the file with the whole history (`fetch-depth: 0`) before the
  Docker build, because the web stage copies `frontend/` without `.git`; a
  build with no git keeps the file it finds, or writes an empty one so the
  panel still loads. The file is gitignored and served `no-cache`. Commit
  subjects are what a villager reads there, in English — write them for that
  reader. **Where it lives**: a chip row mirrored at the foot of Home, under
  the buildings, seen only by whoever scrolls down; its chips — 🔨 Changelog
  and ✉️ Feedback — open one panel at a time under the row, per device, off
  by default.
- **Feedback goes to the builder, never into the database.** `POST
  /api/feedback` (`feedback.go`), any house: a section (the rooms Home shows,
  Map, Codex, Notifications, Something else — the English name travels), the
  text (1–4000 characters) and an optional reply address, which becomes the
  email's `reply_to` and is used for nothing else. With `RESEND_API_KEY` and
  `FEEDBACK_TO` set it is emailed through Resend (`internal/mailer`, stdlib,
  the shape of truetothebook's client, no retry); without them the whole
  message is written to the server log under `FEEDBACK` and the house is told
  it went, so a village or the demo loses nothing before email is wired. A
  failed send is a 502 the page shows as "try again later", and the message
  is logged with the cause. The demo hands anyone a house, so the door is
  public there: 5 per IP per hour, and past 50 emailed a day village-wide
  a message is logged, not emailed, and the house is told to try tomorrow —
  never told it went. Both caps are in memory: per container, reset by a
  restart. The form says the house's name travels with the message. The reply address
  and the text leave for Resend, a processor — the privacy note owes a line
  (`docs/later.md`).

## Layout

```
backend/            Go: main.go, internal/{config,store,httpapi}; migrations embedded
frontend/           Vite + React; src/rooms/* one file per room (Projects.tsx holds list + page; Contacts.tsx holds the room, the
                    fzf-style match, the letter shelves and the type picker that is its only user). Hall.tsx = Calendar.tsx + Board.tsx
                    stacked, because the tavern is one door; src/map/VillageMap.tsx; rooms/Codex.tsx = the founding text, from the DB;
                    rooms/Settings.tsx = the steward's village settings (switches, village-wide mutes, export), a page, not a room;
                    src/Changelog.tsx reads public/changelog.json, which scripts/changelog.mjs writes from the git log at build
frontend/public/    manifest.webmanifest, sw.js (push only, no caching), icons, backdrop.jpg (aerial photo behind the gate), fonts/ (self-hosted, OFL)
                    icon.svg is hand-drawn paths, full-bleed, content inside the central 80 % safe circle
                    (an emoji glyph renders off-centre and monochrome — do not go back to one)
                    src/push.ts, src/Install.tsx, src/AddPhone.tsx, src/photo.ts (auth'd photo fetch + browser-side shrink)
backend/internal/httpapi/codex.go  the codex: four routes and the one-time stdin import
backend/internal/httpapi/feedback.go  the feedback door; internal/mailer = one Resend POST, stdlib; frontend src/Feedback.tsx is its form
backend/internal/httpapi/i18n.go   tr() for push texts and the manifest's line, off i18n/<lang>.json — the dictionaries the page imports too (@i18n alias);
                    i18n_test.go refuses a translation whose verbs, end spaces or leading icon differ from its key
backend/internal/httpapi/contacts.go  the phone book: four routes, and the one place a type's spelling is settled
backend/internal/httpapi/roster.go    the duty roster behind its switch: duties (steward), turns (a house's own), the twelve-month count
backend/internal/httpapi/dues.go   the dues ledger, behind its switch: five routes, the amount parser, the one total besides the roster's
backend/internal/httpapi/mcp.go    the door for agents: the tool table (the whole surface), the JSON-RPC endpoint, the in-process
                    dispatcher and the key route; the fence that keeps a key on /api/mcp is in auth.go
backend/internal/httpapi/shed.go   tool photo routes, wishlist; photos.go = how a photo is read and served (≤2 MB, auth) + project pictures; remind.go = return nudges
                    threads.go = comments on any subject + wish options; weather.go = ARSO, server-side; static.go = the embedded frontend,
                    which fills the village tokens ({{VILLAGE_NAME}}, {{LANGUAGES}}, {{LANG}}, {{DESCRIPTION}}) into index.html and the
                    manifest at serve time, and the village's own backdrop over the built-in one — the build knows no village
                    (config.go: VILLAGE_NAME, LANGUAGES, PUBLIC_URL, PUSH_SUBJECT have no default, the binary refuses to start without them)
docs/               design docs the owner and the assistant decide on together (navigation growth, Projects, Campground,
                    `design-membership.md` — accounts for people who live here without land, options only, nothing built;
                    and `design-mcp.md` — the API as MCP tools for agents: built, with the options rejected and their costs);
                    `later.md` = the one home for what is brainstormed, designed-and-set-aside, or still undecided
docs/diagrams/      hand-drawn SVG sketches belonging to those docs
.github/workflows/  build-backend: the image to GHCR (ghcr.io/dennislapchenko/porta-pagi), sha-<commit> and latest; nothing deploys from here
backend/internal/httpapi/maps.go   serves and imports the map rows; a village's seed files live in its deploy repo
backend/internal/httpapi/demo.go   the demo village: demo/seed.json (the words), demo/parcels.geojson + water.json (nobody's land),
                    seeded by /server demo-seed into an empty database
```

## Docs stay current, in the same change

- **One fact, one home.** Architecture and invariants: this file. Stack and
  commands: `README.md`. A village's manual VM steps and its context live in
  that village's deploy repo, not here.
- A change to the API shape, a table, the deploy path or a workflow updates the
  doc that owns it **in the same commit**. Docs describe current state — when
  something becomes history, delete it rather than framing it as a change.
- A new room or a new table has to answer which villager job it serves, in one
  line, in the design doc that proposes it. If that line cannot be written,
  the feature is not needed.
- `.claude/launch.json` starts the dev frontend or the backend; `task be:run`
  is the same backend from a terminal. Checking the CSP against a real page:
  `README.md`.

## Memory

@.claude/memory/MEMORY.md

In-repo memory lives in `.claude/memory/`: one file per fact, `MEMORY.md` the
index, same frontmatter shape as the assistant's global memory. It holds how
the owner likes to work, project constraints not derivable from code or git,
and pointers. It never holds what a doc already records, and never anything
private: the repo is public (first working rule below). Update a file rather
than adding a duplicate, delete one that turns out wrong.

## The reviewer

`.claude/agents/reviewer.md` is a Fable subagent that answers one question
about a finished turn: what should have been done better, as a
psychotherapist, an engineer, a UX/UI specialist and a homesteading
practitioner at once. **Every turn that changed files invokes it before
ending**, with what was asked and what was done, and then acts on the
feedback or surfaces it to the owner. `.claude/hooks/review-gate.py`, wired as
a Stop hook in `.claude/settings.json`, blocks the turn once if that was
skipped. The reviewer edits nothing.

## Working rules

- **This repo is public, or will be.** The name of the village that built it
  and the public GURS and ARSO data are the only specifics in it. Nothing that identifies a person or
  describes a household comes in here: no names of people, no phone numbers,
  no house-by-house facts, no quotes from the owner's private homestead notes
  about people or tenure. Pointing at a file in that repo by path is fine, its
  content about people is not. Abstract it or leave it out. This applies to
  docs, commits, memory and agent files alike. Test fixtures use invented
  house names. History is not rewritten for a name that slipped in (owner's
  decision 2026-09-09): fix HEAD, no force-push.
- Work on `main`. One person owns this repo; a side branch and a pull request
  only put a gate between a verified change and the villages. Commit to `main`
  and push (owner's rule 2026-09-07).
- Run `task check` before every commit. CI runs the same.
- Migrations are append-only files in `backend/internal/store/migrations/`;
  never edit an applied one.
- Prefer a few lines of code over a dependency. Backend has two (sqlite,
  webpush-go for VAPID + payload encryption); frontend has three runtime deps.
  Adding one is a decision to write down here.
- Commits: lowercase, succinct, say what changed and why. No AI trailers, no
  backticks in subjects (they break the Telegram deploy ping).
- Push to `main` builds the image, frontend and backend together
  (`backend/Dockerfile` builds the frontend), and CI pushes it to GHCR as
  `sha-<commit>`. **Nothing deploys from here**: a village rolls the tag in
  its own deploy repo, and verifies there. Only `backend/**`, `frontend/**`
  and the workflow itself trigger a build — a docs-only commit builds nothing.
- A change that a village's deploy needs — an env variable, a migration a
  village must rehearse, a CLI a village must run — is announced in the commit
  subject: that line is what the village's changelog panel prints.
