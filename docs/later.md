# Later: what is written down and not built

Recorded 2026-09-07, four days after the first commit. This file is the one
home for everything the portal does **not** do yet: rooms that were
brainstormed, variants that were designed and set aside, decisions still open,
and ideas pondered for a tight-living community. What *is* built lives in
`README.md` (rooms and stack), the village's context file in its deploy repo (the villager job
each room serves) and `CLAUDE.md` (the invariants). Nothing here is a promise.

**The rule that governs this whole file**: no room ships before the houses ask
for it, after about four weeks of using what exists. Every row below is a
hypothesis until a house says otherwise, in the Tavern itself.

## Designed and set aside

Each of these has a written argument. Build from the argument, never from
memory.

| Item | Where the argument is | What it would cost |
| --- | --- | --- |
| `houses.left_at`, the soft ending of a stay: devices and invite revoked, the row kept and flagged, faded to the bottom of the Houses room, reversible by a steward | `design-membership.md` § Off-season, and ending a stay | One column, a handful of filters. Replaces deletion as the exit only if a deletion ever hurts |
| A "Commons" super-room holding Projects, Campground and a future Archive behind one tile | `design-more-rooms.md` § Navigation, option F | One more tap to every room in it. Earns its place only when Home tiles pass about eight |
| Task due reminders, read off `due_at` and one `reminded_at`, like tool returns | `design-more-rooms.md` § Push | A second nudge loop in `remind.go`. No counter, ever |
| Steward button that clears camp labels older than 12 months | `design-more-rooms.md` § Privacy | One endpoint, one button. Waits for the privacy note |
| Narrowing "any house may edit an event" to the creator and stewards | `CLAUDE.md`, provisional invariant | Waits for a reduced viewer role, which waits for volunteers who are not villagers |
| Sorting contacts by the date a number was added or last updated, beside the alphabetical order. The owner said "maybe later" (2026-09-28) | `CLAUDE.md` § the phone book; `sortContacts` in `Contacts.tsx` | One comparator and a toggle. Under a date order the letters give way — a date has none to thumb |
| Slovenian case in the run banners. A destination is free text a house typed, so "v " + text prints "v Ribnica" where "v Ribnico" is correct. All three banners have it — the run created, the run moved, the run called off | `server.go`, the run notifications | The case-free form is "smer Ribnica" ("vožnja smer Ribnica"), applied to all three at once so they still read alike. Three strings and their pins in `banner_test.go` and `TestMarketEdits`. The order banners (`runWords` in `server.go`) do not repeat it: the shop a house typed sits after a colon, "zbira naročila: Amazon", "Odpade naročilo hiše Zeleni Volk: Amazon", and the pins in `banner_test.go` and `TestRunKinds` hold them to it |

## Brainstormed rooms, ranked by how often a village would reach for them

Carried over from the owner's planning notes, abstracted. Version labels are
the original guess, not a schedule.

| Idea | Building | Why | Version | Risk |
| --- | --- | --- | --- | --- |
| Seed and seedling exchange, harvest surplus | Market, a seasonal shelf | Spring and August peaks. Same shelf as give-aways with a "seeds" tag | v1 | Low |
| Bulk orders: firewood, feed, gravel, insulation — **built 2026-09-28 as an order run** (`runs.kind = order`): one house places it, the others add their needs before it closes, one delivery | Market, Runs & orders | One delivery, shared freight. **The app records the order, never the money**: no amount, no shares, no who-owes-whom | built | Money among neighbours. A list, not a ledger |
| Visitors board: who hosts guests this weekend | Watchtower | Dogs, gates, strangers on the path | v1 | Low |
| Photo wall, a chronicle | Tavern | The village sees itself grow. Photos already live in the database as BLOBs | v1 | Storage grows with the years |
| Seasonal map skin | Map | The game feel. The map changes with the real season | v1 | Low |
| Fields: growing plans per house | New room | What grows where, what is planted when | v1 | Only if houses garden together |
| Almanac: per-house frost log | New room | Frost pooling is this site's binding constraint. A shared record of first and last frost per place is worth more than any forecast | v1 | Needs a habit, or it stays empty |
| Archive: village knowledge base. Valves, pipes, who to call, how the water system works | New room | Infrastructure knowledge lives in a few heads | v2 | Needs a steward or it rots |
| Skills board: who welds, who has a chainsaw licence | Archive, opt-in cards | Useful the day something breaks | v2 | Privacy. Opt-in only, each house writes its own |
| Duty roster: mowing common ground, snow, water system | **Built** behind the `roster` switch, off by default (`docs/design-more-rooms.md` § Duty roster). The village that built the portal leaves it off | Only if the collective has duties | — | Builds resentment if the roster is not the collective's own idea: a steward writes the duties the village agreed, nothing assigns a house |
| WhatsApp one-time-code login | Gate | Replaces the invite link | v2 or never | Meta's Business API, a per-verification fee |
| Moving into the collective's own website | Deploy | One address for everything | v2 or never | The collective asks for it |
| Renown points, quests, leaderboards | — | **Never, unless the houses ask.** Counted reciprocity turns into debt | — | High |

## Pondered 2026-09-07: what a tight-living community could still want

Ordered by how much each would beat "scroll up in the WhatsApp group". Each
respects the invariants: no tallies, no money, nothing public, nothing on the
cadastre about who sleeps where.

1. **Cold pools on the map.** The priority-flood that found the streams also
   finds the sinks, and a sink is where frost settles. One more JSON file from
   the same terrain pipeline, one more component beside `Water.tsx`, and the
   Almanac has a map before it has a table.
2. **One real thermometer.** The forecast panel is a town away and must never
   become a frost source. A cheap sensor at the cold pool posting to the backend
   would be the one measurement the village owns. It turns the Almanac from a
   diary into an instrument. Hardware, so a separate decision.
3. **State of the commons.** Pump on or off, road passable, gate broken, event
   ground mown. Each shared thing has a current state, a last note and who wrote
   it. It is a need that never closes.
4. **Knowledge pinned to a place.** The Archive as map pins: the main valve, the
   hydrant, the septic lid, the meter. A newcomer taps the map instead of asking
   who remembers. Infrastructure on the map is not tenancy on the map, so the
   cadastre rule holds.
5. **Who is here.** The Watchtower says who is away. The inverse, who is in the
   village this weekend and who hosts guests, answers the dog, the gate and the
   stranger on the path. Same privacy line, same logged-in app, and it folds
   the visitors board into a room that exists.
6. **The year's wheel.** Chimney sweep, winterising the water system, the spring
   clean of the event ground. A recurring event with no assignee. The roster
   risk disappears when nobody is named.
7. **A thing for a day.** A shared trailer or the good chainsaw booked for
   Saturday. The tool shed already knows who holds a thing. A date on the loan
   is one column.
8. **Wildlife log.** "Bear seen by the upper road, Tuesday evening." Not an
   alarm, because an emergency is a phone call. A dated list is local knowledge
   that otherwise lives in a few heads.
9. **Consent, not votes.** Wish options are findings and nothing ranks them. A
   proposal room could work the same way: a thread where only a reasoned
   objection is a state, and no objection after a date means agreed. This is a
   governance decision for the village to make in a room, not a feature to
   ship quietly.
10. **The village book.** The export already holds every post, event and photo.
    A yearly PDF from it is a chronicle nobody had to write.

The best next feature this week is none of these. It is a pinned post in the
Tavern asking the houses what they miss, and then four weeks of listening.

## Open decisions that block nothing today

| Decision | Who answers | Recorded where once decided |
| --- | --- | --- |
| The roster prints a faint 0 for every house, a house with no phone in the portal included, which can never write a turn. Keep it; let a steward enter a turn for such a house, the line saying who entered it; or draw no row until a house has written its first turn (review, 2026-09-28) | The owner | `docs/design-more-rooms.md` § Duty roster, `Roster.tsx` |
| `listEvents` returns only the last 60 days, so a past event older than that leaves the calendar, while `CLAUDE.md` says a past event never does. Found by the demo seed agent, 2026-09-28; looks like a bug, not a decision — fix or reword | The owner | `server.go` `listEvents`, `CLAUDE.md` § calendar |
| GURS open-data licence string for `parcels.geojson` | The owner, from the GURS terms | `CLAUDE.md` § Cadastre, the village's context file § Map data |
| Treasurer, and who camp money is handed *to* | The collective | `CLAUDE.md` § Campground |
| The one-page privacy note, with a line about campers, **and a line about agents**: an away-notice can be read by a neighbour's assistant, as it can by a neighbour's phone. The village was never told, and the argument for putting the Watchtower on the agent surface only works if it is. Since 2026-09-10 the same note owes **a line about the phone book**: the contacts room holds the numbers of people *outside* the village — a well-driller, a vet, an office — who never joined the portal and cannot switch anything off, and an agent reads those too. Nobody in the village can answer for them, which is the argument for keeping the room to numbers a tradesman hands out anyway, and against it ever growing a field about a person. Since 2026-10-07 it owes **a line about birthdays**: a birthday holds a person's name, year and maybe hour of birth, every house and every house's agent reads it, the export carries it, and any house may write anyone's down — the form asks for the person's yes, nothing enforces it. Since 2026-09-28 it owes **a line about the feedback door**: what a house writes there, and the reply address if it gives one, leave for Resend (an email processor) and the builder's inbox, and no table keeps them — until email is wired they sit in the container log | The collective's board, a lawyer confirms | A new `docs/privacy.md` |
| A Slovenian-speaking house checks the room names, the Houses subtitle and the MCP key box (`design-mcp.md` § The Houses room), and since 2026-10-07 the almanac's lines (the full moons' months, the eclipse lines — "Polna luna" over "ščip" is the owner's pick, 2026-10-07, until a Slovenian house says otherwise; the pill for *celestial* reads "nebo", the owner's pick, 2026-10-08) and the birthday words | Any house | `i18n/sl.json` |
| Stewardship on an MCP key. None today: an agent has no confirm dialog and the steward routes are the ones a confirm dialog guards. The owner left it open — "later key might carry stewardship". One line in `requireHouse` | The owner, when an agent needs to export or create a house | `CLAUDE.md` § A door for agents, `design-mcp.md` § Decisions |
| OAuth for MCP clients that demand it (claude.ai connectors). A static key in a header is the whole auth story today | The owner, if a house uses claude.ai and not Claude Code | `design-mcp.md` § Decisions |
| Deletes through MCP. None today: a phone gets a confirm first. Would come as one `delete_*` per table with the route's own owner-or-steward check | The owner, if an agent ever needs one | `design-mcp.md` § The tool surface |
| The demo's first language. `CLAUDE.md` § The demo village and `README.md` say `pt,en`; the demo on the VM (gaias-choice `deploy/app/compose.yaml`) and the local `pagi-demo` preview run `en,pt`, so a visitor lands in English. Found 2026-10-07 when the preview moved here | The owner | `CLAUDE.md` and `README.md` § The demo village, or the two configs |
| The all-kinds-on push default once more houses join | Every house, after a week | the village's context file § Notifications |
| A second steward | The village | the village's context file § People and roles |
| What a household without land calls itself | The household, when the first one joins | `design-membership.md` |
| Whether editing the Codex in the app *amends* the adopted text or *proposes* an amendment for the council. Today any house edits and the section names it; the first edit replaces the council's wording with no copy kept but the nightly backup. A proposal state, or a section history, waits for the village to say it wants one | The collective | `CLAUDE.md` § Codex |
| Reordering codex sections in the app. New sections append; the order the import set is the document's | Any house, if a new section ever belongs in the middle | `Codex.tsx`, one `ord` field |
| The two languages of the adopted text do not carry the same number of principles in one section. Which one is the text | The collective | The codex itself, in the app |
| The offers' `joint` tag beside order runs. A give-away tagged `joint` rings "starts a joint order", but an offer is claimed by one house, so it cannot hold who wants what; an order run can. Either `joint` goes (existing rows read as give-aways, the MCP tag list shrinks) or it is kept for a different thing — the share-out after the delivery, "12 sacks came, 3 spare, who wants one" — and renamed to say so. Nothing removed until then | The owner | `CLAUDE.md` § Push, the Market's offer kinds |
| Since 2026-09-28 the Market's pickers no longer offer `joint` for a new give-away (the orders review: two 📦 pickers for one idea); an existing `joint` row keeps it when edited, and `create_offer` still takes it. Whether it goes for good or comes back as the share-out after a delivery is still the owner's | The owner | `Market.tsx`, `mcp.go` |
| How long an order stays on the list. A run leaves it a day after its cut-off, which fits a car; an order's delivery comes days later, and until then the needs on it show no 📦 on their card. A delivery date, or "leaves the list when the driver marks it arrived", would fix it — the same one-column shape either way | The owner, after the first real order | `server.go` `listRuns` |
| The village's promise that its portal runs with the product repo gone is only literal with the image on the box: a `docker save` tarball beside the nightly backups, refreshed by the same cron. Today the image lives only on GHCR under a mutable tag | The owner, before a second village pays | `porta-pagi-cloud/docs/commercial.md` §13 |
