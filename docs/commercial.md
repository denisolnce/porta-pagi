# Selling the portal: what it would take

Written 2026-09-13, at the owner's request. It is the argument for and against
turning the village portal into something other villages pay for, with the
work, the money and the costs named. **Nothing here is built.** What is decided
is in the block below; every other line is an argument, not a promise.

## Decided (owner, 2026-09-23)

- **Path B.** Open the source, sell hosting: "your own server, set up and run
  by us, and you can get at it too." One person, small on purpose.
- **A free village is a small village.** Self-hosting is free up to **ten
  houses**; above that a licence. Houses, not people — the portal has no
  per-person accounts and cannot count people, and a cap it cannot count is a
  cap it cannot enforce. The owner knows a cap in published source is a
  suggestion to anyone who can edit a constant; the fence is the licence text,
  the code only makes the honest path the easy one. See §13.
- **The product is Porta Pagi** — Latin, "the village's gate" — and the repo
  is `porta-pagi` (owner, 2026-09-23).
- **The product moved to its own repo on 2026-09-23** — this one, `porta-pagi`
  (`f6b9c24` merged the village repo's history, `1fbbe4d` renamed the module
  and took the deploy files out) — and is the source of truth. The village's
  repo, `mokri-potok-portal`, keeps the village: its deployment, its config,
  and nothing the product owns. The shape, and what was pulled out of the
  code first: §13.
- **The demo village speaks English and Portuguese.** Its year of life is
  drafted by the assistant, read against real community writing first, and
  argued over by a reviewer whose one job is to catch the sentence no villager
  would write. Portuguese is a new language for the portal, not only for the
  demo — §13.
- **Support: everything answered within 48 hours.** The owner can bring in
  other people when needed. Who, and for what, is not decided.
- **The collective's portal goes nowhere if the business stops.** It is
  self-hosted already, pinned to an image by tag (`sha-<commit>`, which the
  registry could still lose with the package), and the split made that
  nearly literal: the village's repo keeps enough to run it with no product
  repo at all — except the image itself, which is still only on GHCR
  (`later.md`).
- **Cadastre is the village's choice.** No second country is planned. A
  village brings its own map data, or pays for the map service (§5), and until
  then the map button says what is needed and how to get help with it.
- **The website is the control plane.** The landing page and the hosted
  villages' management live in one private repo (§13); the demo is a
  deployment of the product image.
- **The licence is BSL 1.1** with the ten-house grant (§13). Written and
  pushed on 2026-09-23 as `LICENSE` in `dennislapchenko/porta-pagi`, private
  and the code landed there on 2026-09-23; the package
  `ghcr.io/dennislapchenko/porta-pagi` is public since that day so a village
  pulls it, the source repo stays private until the two licence parameters
  are settled. Two parameters in it are the assistant's picks,
  changeable until the repo goes public. The **Change License is AGPL 3.0 or
  later**: the BSL covenant takes any licence compatible with GPL 2.0 *or a
  later version*, and AGPL 3 is compatible with GPL 3 by its own section 13 —
  a defensible reading, not one MariaDB's FAQ confirms either way. On that
  reading the covenant does not choose; the business does. Apache would let a
  competitor host the four-year-old version closed; AGPL would not, and
  hosting is the business. And the grant **forbids offering the work to
  third parties as a hosted service for a fee or other consideration,
  directly or indirectly** — the standard BSL clause that keeps paid hosting
  as the business. A volunteer running three hamlets' instances for nothing
  is inside the grant; one passing on the VPS bill is arguably not, and
  whether cost recovery should be inside is the owner's to say (§12). A
  house account is defined in the grant itself, in plain words: one shared
  account for one household, however many phones use it; accounts that stand
  for shared land or places are not house accounts.
- **`porta-pagi` lives under the owner's own GitHub account** for now.
- **The two licence parameters stand** (owner, 2026-09-24): AGPL 3.0 or
  later as the Change License, and the grant's clause against hosting for
  third parties for a fee. §12 q1 is closed. The source repo goes public when
  the landing does, because the landing says "the source under BSL 1.1".
- **The cloud repo exists** (owner's yes, 2026-09-24): `porta-pagi-cloud`,
  private, under the owner's account. It holds the landing page (§3) and
  ships it as an image, `ghcr.io/dennislapchenko/porta-pagi-site`, from its
  own CI. The control plane grows there later (§13).
- **The landing and the demo run on the gaias-choice VM** (owner,
  2026-09-24), bundled with that stack for now:
  `porta-pagi.gardenofatlantis.com` is the landing,
  `pp-demo.gardenofatlantis.com` the demo; the owner makes the two A records.
  Both are services in that stack's own compose, not a third poll entry: that
  stack owns Caddy, its poll entry exists, and a private repo's entry would
  need a token in the VM's secrets and a hand step in `/opt/doco-cd/poll.yaml`.
  The cloud repo ships images, the VM runs them. **This puts the demo beside
  a village, which §4 says never to do.** The owner chose it knowingly, for
  now; the demo's `/data` is a size-capped tmpfs, the container has memory,
  CPU and process caps, and it sits on a compose network of its own where
  the only neighbour is Caddy, so the worst a visitor can do is kill the
  demo and take half a core with it, not fill the village's disk, starve
  it, or reach its container. It
  moves to its own host before the first paying village.
- **The demo's Portuguese is Portugal's** (pt-PT).
- **Outside patches: none for now, a CLA when the first real one arrives.**
  `CONTRIBUTING.md` in `porta-pagi` says so.
- **A village that wants a dues ledger or a duty roster with counts gets
  one** (owner, 2026-09-23): the portal is hosted by them or for them, so what
  it counts is theirs to decide. The owner's own village keeps its invariants.
  How the two live in one binary (owner, same day): **a list of switches —
  features a village turns on or off, each with its own default**, most of
  them off. Never a fork. The "do not add a total anywhere" rule then belongs
  to the defaults, and is resolved in `porta-pagi`'s own `CLAUDE.md` after
  the split (§13), not here. The cost, paid knowingly: "we count nothing"
  stops being the product's line and becomes "counting is yours to switch
  on"; §3 says so.

## 0. The tension, first

`CLAUDE.md` says, in its second paragraph:

> It is a **social tool for a handful of houses**, not a product: every feature
> must beat "scroll up in the WhatsApp group" or it does not ship.

And `village.md` lists under *Things that are deliberately not here*: renown
points, leaderboards, **money of any kind**, public pages.

Selling this does not violate those rules — a village that pays for hosting
still gets a portal with no points and no ledger — but it puts a second party
in the room. Today one collective decides what the portal is, in the Tavern.
The day a paying village asks for dues tracking, a duty roster with counts, or
a public page for their guesthouse, the invariants stop being the owner's
convictions and become a support conversation.

**That is the actual decision.** Not the code, not the hosting, not the price.
The engineering below is three to five months of ordinary work. The question is
whether the owner wants the collective's tool to have customers, because a
customer's "no" costs something a villager's "no" does not.

Two guardrails make it survivable (the first in its 2026-09-23 form; the
2026-09-13 draft had it refusing every counted thing outright):

1. **A paying village may have what it asks for, behind its own switch.** The
   invariants stay the defaults, and the owner's village never flips them. The
   refusals that stay refusals — no ads, no trackers, no third party in the
   page, nothing sold inside the rooms — are the manifesto.
2. **The owner's collective stays the design authority for the defaults.** A
   default changes because the village asked in the Tavern, never because a
   customer asked in an email. A customer gets a switch, not a vote.

**And two costs the village pays, which the guardrails do not cover:**

- **The roadmap keeps its direction but loses its speed.** Today a thing asked
  for in the Tavern on Saturday can ship on Sunday. With a hundred villages it
  means a migration, six translations, a rollout across a hundred databases and
  a support queue — so the same request takes a fortnight. Guardrail 2 protects
  *what* gets built; nothing protects *how fast*, and the speed is the part the
  villagers actually feel.
- **The owner's evenings do not become free, they change shape.** The money
  buys off the hosting bill, not the hours: what replaces building is support,
  in languages the owner does not speak, for people who cannot read the logs.
  Anyone doing this to get their evenings back has the arithmetic backwards.

If either guardrail feels like a cage, or either cost feels too high, stop
reading — the honest answer is to open-source it and host nothing.

## 1. What actually exists, as an asset

Worth stating plainly, because the sales story and the cost estimates both
follow from it.

| Asset | State | Commercial weight |
| --- | --- | --- |
| Go backend, stdlib HTTP, SQLite, ~4.4k lines plus ~2.4k of tests, 92 routes, **two** dependencies | Working, tested, deployed | Very high. One static binary + one file is the cheapest per-tenant unit in the industry |
| React frontend, ~3.4k lines, **three** runtime deps, PWA with push | Working | High. No app store, no build farm |
| One container serves API + frontend, one origin, CSP, no CORS | Working | High. Provisioning a village is `docker run` |
| Nightly `VACUUM INTO` backups, `GET /api/export` full JSON dump | Working | High. "Your data, out, in one click" is a sales line most competitors cannot say |
| Nine rooms with a real opinion behind each | Working | The product. Not replicable by a generic tool |
| Cadastral map as the home screen, terrain-modelled watercourses | Working, **Slovenia-specific** | The differentiator **and** the biggest per-country cost. See §11 |
| MCP door — 51 tools, the API as an agent surface | Working | Genuinely novel. No competitor has it. Small audience today, good press |
| Slovenian + English, `t()` everywhere, 379-line flat dictionary | Working | Cheap to extend, but "Slovenian first" is an invariant that must become "the village's language first" |
| Web push, quiet hours, per-phone consent, per-house kind filters | Working | High. Notification manners are a real feature nobody markets |
| **No LICENSE file** | **Missing** | **Blocking.** A public repo with no licence is all-rights-reserved: nobody may legally run it. See §11 |

Three `TBD`s in the repo become shipping blockers the moment money changes
hands: `backdrop.jpg` provenance (a screenshot of a map service, licence not
recorded), the GURS attribution string, and camp-label retention. A hobby repo
can carry those. A product cannot.

## 2. Who buys this

The instinct is "villages". The market is narrower and stranger than that.

**Real buyers, ranked by fit:**

1. **Intentional communities and ecovillages** — 10–60 households, ideologically
   allergic to surveillance SaaS, already self-organising, already frustrated by
   their chat group. The Foundation for Intentional Community directory lists
   roughly a thousand; GEN Europe has hundreds more. **Best fit by a wide
   margin**: the refusals are why they buy.
2. **Cohousing projects** — Netherlands, Denmark, Belgium, Germany, UK. More
   organised, more likely to have a small budget, more likely to also want the
   thing the portal keeps off by default (a shared ledger). Good fit: it is
   theirs to switch on.
3. **Hamlets and dispersed rural settlements** — the owner's own case. Hardest
   to reach (no directory, no conference, no mailing list) and lowest ability to
   pay. Reachable only through municipalities or word of mouth.
4. **Alpine hut collectives, private road associations, allotment
   associations, small marinas** — narrow, real, and the map matters to all of
   them. Underserved and unglamorous.
5. **Municipalities running it for several hamlets** — biggest contracts,
   worst fit. Procurement, accessibility law, tenders, and they will want
   public pages. Take the money knowingly or not at all.

**Who does not buy this:** suburban HOAs (they want dues, ledgers and
violation tracking — that is a different product, well served by Buildium,
FrontSteps, Condo Control), apartment buildings (no map, no land), and anyone
whose problem is "we need a website".

**The competitor is not software.** It is the WhatsApp group: free, installed,
and already carrying every conversation. Every sale is an argument that a
second app earns its icon. That argument only lands on things chat is
genuinely bad at — the Watchtower, the shed's "who has the ladder", the
calendar with answers, the map — which is exactly what got built. The rooms
that lose to chat were correctly never built.

**Honest market size.** Take the addressable set as ~5,000 organised
communities in Europe and North America that could plausibly want this. A very
good outcome converts 2–4 %. That is 100–200 villages. At the pricing in §3
that is €18–36k a year in subscriptions, before churn. **This is a lifestyle business, not a venture
business, and no amount of execution changes that** — the ceiling is set by
how many small collectives exist, not by how well it is sold.

## 3. The offer

One sentence, on the landing page, above everything:

> **Your village on one page — private, quiet, and yours to keep.**
> A map of your hamlet where the things that must stay findable live: who is
> away, who is going to the shop, when the work party is, who has the ladder.
> Not a group chat. Not a social network. No ads, no trackers, nothing sold
> inside. What it counts is up to your village, and the default is nothing.

And under it, the commercial line:

> **€15 a month for the whole village. Every house included. Your own server,
> your own domain, your data exportable forever. Walk through a real one — no
> signup, no email.**

Why this shape:

- **Per village, not per house.** Per-seat pricing punishes the village for
  including the renter with no land — the exact house the membership design
  went out of its way to welcome. It would contradict the product in the
  price list.
- **"Your own server" as the default, not a tier.** See §6. It is true, it is
  cheap, and it is the sentence this audience is listening for.
- **"No signup" on the demo.** The product is felt, not specified. Every
  gate before the feeling costs a customer.
- **The refusals are in the offer.** "No ads, no trackers, nothing sold
  inside" is not a footnote — for this buyer it is the reason. What a village
  may switch on is said in the same breath, so nobody feels sold a different
  thing later.

### The landing page, section by section

One page, no navigation, self-hosted fonts, no third-party anything — the same
rules the portal itself follows, because the buyer will check.

| # | Section | Content |
| --- | --- | --- |
| 1 | Hero | The **demo** village's map, crests on parcels. The sentence above. Two buttons: **Walk through the demo** (primary), **Read our promises** (plain; it was "Read what it refuses" until the owner found a second button that says no a poor second door, 2026-09-28). Never the real village — see the rule under this table |
| 2 | The problem | Three lines of a fake WhatsApp group scrolling past: "Is anyone going to town?" … 40 messages … "sorry what was the answer". Then: "It was said once. Then it was gone." |
| 3 | The nine rooms | One scroll, one screenshot each, one sentence each — lifted from `village.md`'s job table, which is already written in exactly the right voice |
| 4 | **What this will never do, and what is yours to switch on** | Two lists. Never: ads, trackers, third-party scripts, public pages, anything sold inside the portal — the price is for the hosting, and it is on this page, not in the rooms. Off unless your village turns it on: a dues ledger, a duty roster with counts, the campground. The default is the village that built it. **This section is the differentiator and should be the longest on the page** |
| 5 | The map | "Your village, drawn from your country's public cadastre — bring the file, or we draw it for you." One country today (§5, the Decided block: no second country is planned). No list of countries, an honest "ask us" |
| 6 | Your data | Nightly backups, one-click full JSON export, the source under BSL 1.1 (§13), "leave whenever and take everything" |
| 7 | Price | Two columns: **Self-host, free forever** (docker run, the docs, no support) and **Hosted, €15/month** (your subdomain, your map drawn for you, backups, updates, a person to email). One-time map setup fee stated plainly |
| 8 | The village that made it | Why each room exists, told by the builder, illustrated from the **demo** village. The reason is the asset; the neighbours' parcels are not |
| 9 | FAQ | iPhone push, languages, what happens if you stop paying (the export runs and the container stops — data returned, not held hostage), GDPR, who runs it |
| 10 | Footer | Email. No newsletter, no chat widget, no cookie banner because there are no cookies |

**The rule this page must not break.** A screenshot of the real village with
crests on parcels *is* a published who-lives-where map — the house-by-house
fact `CLAUDE.md` § Working rules keeps out of even the village's repo, and the thing the
anonymous away-push exists to protect. §4 already forbids real parcels in the
demo for that reason; the hero is the same picture with more traffic. So:
**every screenshot on the page comes from the demo village.** If the real one is
ever shown, it takes written per-household consent, and no crest ever sits on
a real parcel. Marketing does not get an exemption the away-notice does not.

And consent between neighbours is not the clean instrument it looks like. The
person asking is the neighbour who built the portal, so "no" is expensive to
say. And consent here is not revocable in any way worth promising: withdrawing
it takes the picture off the page, which is all it can do — the indexed copies
and the screenshots stay. Say that when asking, rather than the word
"revocable", which a neighbour hears as "I can change my mind". So: **the
default is no**, the question is asked once and never a second time, and a
household that declines is never named as having declined — not to the others,
not in the file where the answers are kept.

The page is a static file. It must not be a funnel.

## 4. The demo village

**The highest-leverage thing to build, and it is small.** Ten minutes inside a
lived-in village sells better than any page of copy, because the product's
whole argument is a feeling — that the village is *there*, on a map, with a
year of its life in it.

Requirements:

- **A fictional collective.** Invented house names (the test fixtures already
  do this by rule), invented crests, invented people. The demo is public; the
  public-repo rule applies to it with double force.
- **A synthetic cadastre.** Never a real village's parcels — that would put
  real land holdings behind a public demo login. Either hand-drawn polygons or
  a free open-cadastre extract of an uninhabited area, relabelled.
- **A year of life, seeded relative to now.** `/server demo-seed` writing
  posts, a calendar with past and future events and their answers, threads with
  arguments in them, three projects at all three states with picture strips, a
  shed with tools on loan and overdue nudges, an away notice with a watcher and
  a "I fed them, key is under the step" comment, a codex with real-sounding
  sections, camp rows, a phone book. **Dated relative to `now()`** so it never
  reads as abandoned. This is the writing job, not the coding job — budget more
  time for the words than the SQL.
- **Reset on the hour.** A golden snapshot restored by cron. Visitors may click
  everything, including Delete.
- **A way in with no code** — a route that hands out a device token for a named
  house, push disabled, the export still working (it is a selling point), and a
  mintable MCP key so the agent story is demoable too.
  **This is an unauthenticated write surface, which `CLAUDE.md` § Nothing is
  public forbids outright** — it is the same door refused to park4night on
  2026-09-06, and the demo is not a good enough reason to soften the rule. So it
  is fenced by compilation, not by configuration: the route lives behind a
  **build tag** and is absent from the binary every village runs, a test pins
  that absence, and the demo ships as its own image tag. An env-var typo on a
  real village must not be able to open the gate. If that fence cannot be built
  cleanly, the demo is a read-only recording instead, and the feature is
  dropped.
- **A house switcher** in the corner: "You are Hiša Lipa — become the steward"
  so a visitor can see both sides of every permission.

Cost: **4–6 days**, most of it writing the village's year.

**Built 2026-09-23** (`porta-pagi`): the seed (`demo/seed.json`, ten houses,
a year, pt-PT and English, argued over by a prose reviewer and rewritten
once — seasons, codex voice, replies before their posts, lines that toured
features), the synthetic cadastre, `pt.json`, the build-tagged door, the
hourly reset, the house switcher. **Rewritten 2026-09-28**: every room
bilingual the way an Alentejo village is (Portuguese houses, English-writing
foreign houses, a mixed one, translations of the key line), every feature
shown at least once, the contacts with threads and a year of dates, and
seasons that hold on any day it reseeds — a test seeds it twice a month
through a year and checks each room. Not built: pictures on the projects
and the tools, which would have to be fabricated. The
demo site runs on the owner's VM since 2026-09-24 (§11 row 2). **The rule: it runs on its own host,
never beside a village**, because a visitor holds a steward's powers for up
to an hour — posts, an export, a backdrop upload, push endpoints the server
will POST to — and the reset is the only thing the product itself puts
around that. The owner overrode the rule for now, knowingly, with caps and
a network of its own in place of the distance: the Decided block says how.

## 5. How it is offered

Three doors, one binary behind all of them.

**Self-host, free.** The image, the docs, a one-page quickstart. No support, no
promises. This is not charity: self-hosters are the credibility layer for this
audience, they file the good bug reports, and they tell their federation about
it. Expect ~0 % conversion to paid and value them anyway.

**Hosted, €15/month per village.** We provision a container, a subdomain, TLS,
backups and updates. They get an email address that answers. This is the
product.

**Hosted with your map, one-time €250–450.** The onboarding service: import
the country's cadastre for their bbox, run the terrain model for their
watercourses if the data exists, import their codex, design their crests, a
one-hour call with the founding steward. **This is where the real money is in
year one**, and it should be sold as care, not as a setup fee — for this
buyer, "we drew your village by hand" is worth more than the price of it.

**Sovereign, €45/month.** Their own VM in a country they pick, off-site
backups they hold, a signed DPA, and a support commitment. For federations,
larger ecovillages, and anyone whose constitution demands it.

## 6. Dedicated server per village — yes, and it is not a premium tier

**Recommendation: one container per village is the default architecture, at
every price.**

The architecture already assumes it. SQLite, one file, no shared state,
`VACUUM INTO` backups, house IDs unique only within a database, the export
dumping "everything". Making this multi-tenant means a tenant column on every
table, a tenant predicate on every one of ~92 routes, per-tenant map data
anyway, a rewritten export, and a new class of bug where one village sees
another's away notices. **That is a rewrite of the backend to make it worse.**

Per-village containers instead:

- Cost nothing to build — zero code change.
- Are the privacy story, not a compromise on it. "Your village is its own
  server, its own database file, its own domain" is literally true.
- Make export, backup, migration and deletion trivial: it is one file.
- Let a village leave by being handed a tarball, which is the promise on the
  landing page.
- Run cheap. A €6/month VPS holds 20–40 of these comfortably; the binary idles
  at a few MB and the databases are small until photos accumulate.

**The one thing this needs is a control plane**, and it is the largest new
piece of engineering in the whole plan:

- Provision: write a compose file from a template, a subdomain, Caddy
  on-demand TLS, start it, print the bootstrap code into the founder's browser
  instead of into `docker logs`.
- Roll: update `BE_TAG` across N villages, health-check, roll back one.
- Backup: pull the nightly `VACUUM INTO` off-box, encrypted, per village.
- Bill: see §11.
- Rescue: rotate a steward invite without SSH. This is the login fix in §7.

Build it boring: one Go service, the same stack the owner already knows,
Docker + Caddy on a VPS. **Not Kubernetes**, not a per-village Fly app, not a
serverless anything. ~3–5 weeks. Reuse `doco-cd` thinking where it fits.

Photos in the database is the one thing to watch: a village that puts fifty
project pictures up a year grows a database that a nightly `VACUUM INTO`
copies whole. Fine at village scale, worth a quota line in the terms.

## 7. Revamping the logins

The current model is the best part of the product and **it should not change
inside the village.** A house is the account; an invite link travels through
WhatsApp; a pairing code adds a phone across a kitchen table; there is no
password to forget and no email to send. That is not a limitation to be fixed —
it is why a house with an old phone and no patience for accounts gets in.

What is missing is not in the village. It is above it:

| Gap today | Why it is fatal when money is involved |
| --- | --- |
| No self-serve signup | The first steward's code is printed to `docker logs`. A customer cannot read that |
| No recovery but SSH | `docker exec /server code "<house>"` is the only way back in. Not offerable |
| Nobody to invoice | There is no email address anywhere in the system, by design |
| No owner distinct from a steward | The person who pays and the person who pins posts are the same row |

**The fix: a control-plane account, and the village untouched.**

- **One village account** — an email and a magic link (or a passkey), living in
  the *control plane database*, never in the village's SQLite. It can: create
  the village, choose the subdomain, pay, download the backup, and **mint a
  fresh steward invite link**. That last one is `task vm:code` behind a login,
  and it retires SSH as the recovery path.
- **Inside the village: nothing changes.** Invite link, pairing code, house =
  account, device rows, MCP keys. Every invariant holds. The village database
  gains no email column, no password, no person.
- **The founding flow becomes**: pay → we provision → the browser shows the
  bootstrap code once → the founder names their house and is the first steward
  → they WhatsApp the invite links. Same as today, with the log replaced by a
  page.

Explicitly rejected, with reasons:

- **Per-person accounts.** Would break the load-bearing invariant and the
  whole membership design. Never.
- **Google / Apple sign-in.** Puts a third party on a logged-in page and
  contradicts the manifesto that sells the product.
- **WhatsApp Business API one-time codes.** Already rejected in `later.md`:
  a Meta dependency and a per-verification fee. The invite link *is* the
  WhatsApp login and it costs nothing.
- **Email for every house.** Half this audience does not want one. Optional
  per-house recovery email is a reasonable *opt-in* add-on and should default
  off.
- **Passwords.** Nothing in the system has one. Keep it that way.

One genuine new requirement, and it needs stating precisely, because the
comfortable version of it is a lie. **The control plane never queries a
village's database** — it provisions, bills, rolls and rescues, and it has no
read path into the rooms. But it *does* pull the nightly `VACUUM INTO` off the
box (§6), and a backup is the whole village in one file. So on the standard
tier **the operator can read a village's data if they choose to**, exactly as
any host with a backup can, and the landing page must not claim otherwise.

What can honestly be promised, tier by tier:

- **Hosted**: no query path, backups encrypted at rest with an operator-held
  key, access logged, and a written commitment not to look. That is a promise
  about conduct, not about cryptography, and it should be worded as one.
- **Sovereign**: a jurisdiction the village picks, a contract, and an off-site
  backup copy encrypted to a key the village holds, which the operator cannot
  open. What it does **not** buy is cryptographic inability: §5 has the
  operator provisioning that VM and §6 has the control plane rolling and
  rescuing it, so the operator has root, the live SQLite file sits unencrypted
  on that disk, and any key present on the box is reachable by the same root.
  Sovereign buys jurisdiction, contract and recourse — say that, not more.
- **Self-host** is the only tier where "we cannot read your village" is
  literally true, and it is free. That is not a hole in the pricing; it is the
  honest shape of the thing, and saying so out loud is worth more trust than
  the sentence it gives up.

Also worth getting right before a lawyer does: for the village's own data you
are a **processor**, but for the control-plane account, the billing records and
the support mail you are a **controller** — different duties, different
paperwork. And the counterparty signing the DPA is often an unincorporated
collective with no legal personality, which is a weak signature; expect to
contract with a named steward or the collective's association where one exists.

## 8. Native app, or the PWA

**Recommendation: stay a PWA. Do not build native apps.**

The portal already installs to the home screen, has an icon, and sends push.
Native would cost two store accounts (€99/yr Apple, one-time Google), two
build pipelines, review cycles that gate every deploy, and it would break the
thing that makes deploys work today — one image carrying frontend and backend
together, rolled in two minutes.

The one real argument for native is iOS push, which requires the site be added
to the home screen first and silently does nothing otherwise. That is a genuine
tax and it will be the top support question. **Fix it with a first-run install
coach, not with an app**: detect iOS Safari, show the Share → Add to Home
Screen walkthrough with pictures, and do not offer the notification toggle
until installed. `Install.tsx` already exists to build on.

Revisit only if churn interviews name push reliability as the reason people
left. Then consider a thin native shell (Capacitor) that is the same web app
with native push — not a rewrite.

## 9. The business plan, with the numbers said out loud

**Costs, yearly, at 100 villages:**

| Item | Cost |
| --- | --- |
| VPS ×3–4 (villages + control plane + demo) | €300–500 |
| Domain, DNS, transactional email | €100 |
| Backup storage (object storage, encrypted) | €150 |
| Paddle / Stripe fees (~5 % incl. EU VAT handling) | €1,500 |
| Accountant, business registration | €600–1,500 |
| Legal: DPA, privacy policy, terms, one review | €1,000–3,000 (year one) |
| **Total** | **~€4–6k year one, ~€2.5–3k after** |

Marginal cost per village is roughly **€3–5 a year**. Gross margin at €15/month
is above 95 %. The economics are not the problem; the ceiling is.

**Revenue, three scenarios, per year:**

| | Year 1 | Year 2 | Year 3 |
| --- | --- | --- | --- |
| **Pessimistic** — the audience is smaller than it looks | 6 villages, €1.1k + €1.5k setup | 15, €2.7k | 25, €4.5k |
| **Base** — one good case study, steady federation word of mouth | 15, €2.7k + €4k setup | 45, €8.1k + €8k | 90, €16k + €10k |
| **Optimistic** — a federation adopts it, or one post lands | 30, €5.4k + €9k | 90, €16k + €15k | 200, €36k + €20k |

**These are gross adds with no churn in them, which is the most flattering
assumption in this document.** The product's own success metric is whether a
village is still writing ninety days later (§10), so churn is the number that
decides this business, and a first year of 30 % is realistic for community
software — a village tries it, three houses never install it, and they drift
back to the chat group.

The table also bills every village twelve months of the year it arrived, which
no cohort does: villages arrive through the year, so on average each pays
about half of its first year. That one is arithmetic, not an estimate.

Applying both at once, so neither is counted twice — 30 % of the start-of-year
base lost across the year, each arriving cohort paying half a year and seeing
half a year of churn — the base case is roughly **13 / 34 / 62 villages**, and
year three's subscription line is about **€9k**, not €16k. The working is two
lines: the base carried into a year is last year's × 0.7 plus its arrivals
× 0.85; that year's subscriptions are the base × 0.85 × €180 plus its
arrivals × €90.

**Read the base case honestly: year three is around €26k gross as the table
has it and about €19k under that model, call it €16k after the running costs
above, for work that never fully stops.** That is a real second income and a
below-minimum first one. Anyone doing this for the money should not.

The reasons to do it anyway, which are better reasons:

- The collective's tool gets its hosting, its map data and its legal cover
  paid for by strangers rather than by the owner. Not the hours — §0 says
  where those go.
- A hundred villages running a portal with no points and no ledger is a small
  argument, made at scale, about how software could treat people.
- The onboarding service is genuinely enjoyable work: drawing a stranger's
  village from their cadastre and handing it back to them.
- It compounds slowly and owes nobody: no investor, no loan, no payroll. The
  one thing it burns is the owner's evenings, and that is a rate too.

**Three paths, and a recommendation:**

- **Path A — open source, host nothing.** AGPL it, write the docs, let people
  self-host. Cost: two weeks. Revenue: zero. Support load: a GitHub issues tab.
  Keeps the invariants absolutely safe. *Choose this if §0's guardrails feel
  like a cage.*
- **Path B — open source + hosted, deliberately small.** AGPL, sell hosting and
  the map service, cap ambition at ~200 villages, stay one person. Cost:
  3–5 months of build, then ~4h/week. *Recommended.*
- **Path C — a real company.** Multi-tenant, funded, sales, municipalities,
  every feature a customer asks for. Requires abandoning most of §0 and
  rewriting §6. *Do not.*

**Recommend Path B**, with a hard gate: build the demo village and the landing
page **first**, put them in front of thirty communities from the directories in
§10, and only build the control plane if ten of them ask what it costs. The
demo and the page are two weeks. The control plane is a month. Do not build the
month before the two weeks have answered the question.

## 10. Marketing

The audience is small, findable, allergic to being marketed at, and talks to
itself constantly. That combination rules out most channels and makes two of
them unusually strong.

**Do:**

- **The directories, one email at a time.** GEN and GEN Europe, the Foundation
  for Intentional Community directory, Eurotopia, Diggers & Dreamers, the
  national cohousing federations (NL, DK, BE, DE, UK). These are lists of
  exactly the right buyer, published, with contact addresses. Thirty personal
  emails with a demo link beats any campaign.
- **The reference village, told and not photographed.** One case study: the
  real reason each room exists and the builder's own account of what the
  WhatsApp group kept losing. That story is the whole marketing budget's worth
  of asset and it already exists as lived experience. **The story is the asset,
  the neighbours are not** — no parcel screenshots, no house names, no faces,
  nothing a villager did not write for this purpose. The rule under §3's table
  applies here, at a conference table, and in every email.
- **Conferences where this audience physically gathers.** GEN Europe's annual
  gathering, the European cohousing events, regional permaculture convergences.
  A laptop with the demo on a table does more than a booth.
- **Show HN / Lobsters, once, on the technical story.** A single Go binary with
  two dependencies, SQLite, self-hosted fonts, a CSP, no trackers, watercourses
  derived from a 1 m terrain model, and an MCP surface — that is a genuinely
  good post. It reaches self-hosters, who become the free tier, who mention it
  in their federation. Post it about the engineering, never about the price.
- **The MCP door as press.** "Tell your assistant, and it is in the village
  portal" is a story no competitor can tell, and it is already built and
  documented.
- **Slovenia and the Balkans first.** Language, cadastre, and a reference
  customer are all already there. Prove it locally before translating.
- **EU rural development money.** LEADER/CLLD programmes fund exactly this kind
  of thing for exactly these villages. A one-page "how to get your municipality
  to pay for it" guide would convert better than any discount.

**Do not:**

- Paid ads. The audience is 5,000 people; targeting cannot find them and they
  distrust the format.
- Cold outreach at volume, SEO content mills, a newsletter, a chat widget,
  Product Hunt. Every one of them contradicts the manifesto on the landing
  page, and this buyer reads that page carefully.
- Claim growth numbers, user counts or testimonials that are not real. A
  product whose default is "we count nothing" cannot open with a counter.

**The metric to watch is not signups.** It is: how many villages that started
are still writing in the Tavern ninety days later. If that number is low the
product is wrong, and no marketing fixes it.

## 11. What would be required — the ordered backlog

Sizes are working days for one experienced person who knows this codebase.

**Phase 1 — prove there is a market (≈2.5 weeks). Do this first, before
anything else.**

| # | Work | Days |
| --- | --- | --- |
| 1 | ~~LICENSE~~ Written 2026-09-23, BSL 1.1 with the ten-house grant (the Decided block, §13). The two parameters in it stay the assistant's picks until the repo goes public (§12), and the landing's rows 6 and 7 ("the source under BSL 1.1", "self-host, free forever") are only true once it has | — |
| 2 | ~~Demo village~~ Built 2026-09-23 except the pictures (§4). **Live 2026-09-24 at `pp-demo.gardenofatlantis.com`**: a service in the gaias-choice stack (`cb88cbf`, tags rolled in `58ed96f`) on a size-capped tmpfs with memory, CPU and process caps, beside the landing; the demo build seeds itself on an empty ground (`9136318`). Verified on the VM: seeded on boot, healthy, the village's container untouched | — |
| 3 | ~~Landing page~~ Built 2026-09-24 in `porta-pagi-cloud` (`98127b5`): one HTML file, one stylesheet, the portal's fonts, no script tag; nine phone-width shots of the demo; the §3 sections in order, §5's prices, the village told and not named (§12 q2). Ships as `ghcr.io/dennislapchenko/porta-pagi-site`, **live 2026-09-24 at `porta-pagi.gardenofatlantis.com`** (a test address; the real domain is not chosen). Three things the owner still owes: the footer's email (a placeholder `hello@porta-pagi.example` is on the page); the FAQ's "written terms are being finished with a lawyer", which is a promise the page makes for row 15; and §12 q2 — the page tells the builder's reasons in the first person and names no anecdote, but it still says which valley, and the collective has not said yes | — |
| 4 | **Thirty emails** to the directories in §10. **Before the first one goes** (2026-09-28), the page a stranger clicks must be true: (a) the landing on its real domain, not the test address; (b) a real reply address in its footer (a plain mailbox — the feedback email is separate and can wait); (c) the `porta-pagi` source repo public, since the page says "source under BSL 1.1" — the licence parameters are settled; (d) the collective's yes to "a karst valley in Slovenia", or the line cut (§12 q2); (e) `noindex` removed; (f) row 5, because every demo visitor sees the backdrop first. Then: the list of thirty named contacts and one email in the owner's voice, drafted by the assistant, sent by the owner | 2 |
| 5 | Resolve `backdrop.jpg` provenance — replace it if the licence cannot be established | 0.5 |

**Gate: ten replies asking the price. If not, stop and take Path A.**

**Phase 2 — make it deployable for someone else (≈4 weeks).**

| # | Work | Days |
| --- | --- | --- |
| 6 | ~~De-village-ify config~~ Done 2026-09-23 (`f747351`, §13): `VILLAGE_NAME`, `PUBLIC_URL`, `PUSH_SUBJECT` required, ARSO one provider among two | — |
| 7 | ~~Weather provider interface + Open-Meteo adapter~~ Done 2026-09-23 (§13) | — |
| 8 | ~~Map data out of the frontend build~~ Done 2026-09-23, in the database (§13) | — |
| 9 | ~~Cadastre import CLI~~ Superseded by the Decided block: cadastre is the village's choice, no second country is planned. A village brings its own geojson, or pays for the map service (§5). Comes back only if a paying village's country asks for it | — |
| 9b | ~~Licence review, once per country~~ Superseded with row 9. What stays: this village's own GURS attribution string is still `TBD` and §1 calls that a blocker. Was: "Free to download" is not "free to redistribute inside a paid hosted product", and the terms differ by country. This repo's own GURS attribution string is still `TBD` and §1 calls that a blocker — this is the same blocker six times over, and it gates shipping each country, not just documenting it | 1–2 each, plus legal time on any that are unclear |
| 10 | **i18n**: ~~the engineering~~ done 2026-09-23 (§13); what remains is translation — `pt.json` for the demo, then DE, FR, ES, NL, IT as villages ask | 1 per language |
| 11 | **iOS install coach** in `Install.tsx`, gating the push toggle until installed | 2 |

**Phase 3 — the business machinery (≈5 weeks).**

| # | Work | Days |
| --- | --- | --- |
| 12 | **Control plane** per §6: provision, subdomain, TLS, roll, backup, rescue | 15 |
| 13 | **Control-plane accounts** per §7: magic link, village ownership, steward-link rotation | 5 |
| 14 | **Billing**. Use **Paddle**, not Stripe — as merchant of record it absorbs EU VAT across 27 countries, which is otherwise a genuine tax-compliance job for a solo operator | 4 |
| 15 | **Legal**: privacy policy, DPA, subprocessor list, terms, retention policy, breach process, one lawyer review. The portal holds names, phone numbers, home locations, photos and away-notices — under GDPR you become a **processor** and the `TBD`s in `CLAUDE.md` stop being `TBD` | 5 + €1–3k |
| 16 | **Steward's handbook** — the support strategy. At 100 villages in six languages, a written handbook and the demo are the only affordable support | 4 |

**Total: ~12 weeks of build across three phases, gated after the first
2.5.** Plus ongoing: 2–4 hours a week at 50 villages, more in the first
fortnight of each new one.

**Deliberately not on this list:** multi-tenancy (§6), native apps (§8),
per-person logins (§7), any feature a customer asks for that the invariants
forbid (§0).

## 12. Open questions for the owner

Answered on 2026-09-23 are in the block at the top. Still open, and none of
them can be answered from the code:

1. **The two licence parameters the assistant picked** — AGPL 3.0 or later
   as the Change License, and the no-paid-hosting-for-others clause in the
   grant — including whether a volunteer who only passes on the server bill
   is inside it. Both stand unless the owner says otherwise before the repo
   goes public.
2. **Does the collective agree** to the story being told — why each room
   exists and what the chat group kept losing, with the village named. Not its
   data, not its map (§3, §10). A Tavern conversation, not a repo decision.
3. **Who else answers the email**, by name, when the 48 hours would otherwise
   run out.

## 13. The split, and what the code must lose to become a product

Decided 2026-09-23 in outline; the argument for the shape, and the inventory.

### Three repos

| Repo | Visibility | Holds | Why its own |
| --- | --- | --- | --- |
| **`porta-pagi`**, the product | Public, under the licence below | Go backend, React frontend, Dockerfile, migrations, self-host compose and quickstart, the demo seed and the build-tagged demo door (§4), `CLAUDE.md` and the design docs that describe the product | It is what the licence licenses. The demo lives here because its door is a build tag on this code and its seed follows these migrations — a demo repo would break on every schema change |
| **`porta-pagi-cloud`** (landing page + control plane; name is the assistant's suggestion) | Private | The static landing page (§3), and the service behind it: accounts and magic links (§7), provisioning, subdomains, TLS, rolling, off-box backups, billing (§6, §11 rows 12–14) | Billing code and customer records are not for publishing. The page is static and could be public, but the owner wants it to *become* the control plane, and a page that grows into a service should start in the service's repo |
| **the village's** (`mokri-potok-portal`) | Public, as it was | `deploy/`, `.doco-cd.yml`, `infra-log.md`, the village's own data files (`deploy/map/`, the codex seed off-repo), who the village is | The village. It pulls the product image by tag and carries nothing the product owns (done 2026-09-23, `4897c64`). "Runs with the product repo gone" is only literal if the image is on the box too: a `docker save` tarball beside the nightly backups — **not built**, in `later.md` |

One repo for all of it was considered and refused: it would make the private
part gate the public part's licence, and put the customers' billing code in
the archive every self-hoster downloads.

**Order, as done on 2026-09-23.** De-village-ified in place on the village's
repo, behind config with that village's values as the deploy's env, each
slice deployed and verified there — then the history pushed to `porta-pagi`
and the village's repo cut down to `deploy/`. Forking first would have meant
two diverging copies of the same code for the whole of Phase 2.

**History.** The product repo started from the village repo's history, not from a
snapshot: blame survives, and the working rule that history is not rewritten
(owner, 2026-09-09) carries over. The village's name is in that history and
stays; the repo was public all along.

### The licence, for a ten-house cap

An OSI open-source licence cannot carry a cap — AGPL, MIT, Apache all grant
use to anyone for anything. A cap needs a **source-available** licence, and
the product then is not "open source" in the strict sense; say "source
available, free for small villages" on the landing page, never "open source",
because this buyer knows the difference. Two that fit:

- **Business Source License 1.1** (MariaDB, HashiCorp, CockroachDB). Built for
  exactly this: an *Additional Use Grant* names what production use is free —
  here, *one community of up to ten houses* — and a *Change Date* at most
  four years after each release, when that version becomes a real
  open-source licence (say Apache-2.0 or AGPL). The cap expires per version,
  which this audience will read as fair. Recommended.
- **Elastic License 2.0.** Shorter. Free to use, forbids offering the software
  as a managed service and forbids circumventing a licence key. No cap
  language of its own — the cap would be a licence-key rule — and no
  change date, so it never becomes open source. Fits worse.

What the owner gives up against AGPL: the "we count nothing" manifesto (§3
row 4) gains an exception the landing page has to explain, and the
self-hosters who are the free tier (§5) will read the licence before the
code. Ten houses is generous enough that a hamlet never meets it.

**The cap in code, honestly.** The eleventh house is refused at invite time
unless a licence key is present: a signed blob (village domain, house cap,
expiry) checked against a public key compiled into the binary, one env var.
About a day. A self-hoster can delete the check; the licence says they may
not, and that is the whole fence. It must never count people or devices —
only rows in `houses` where `kind = 'house'` — and it must never appear in
the UI as a number. What is refused is the steward's *creating* of the
eleventh house, never a newcomer's join — the invite already in a hand keeps
working. And the wall must not be met while welcoming a neighbour: once the
grant is full, the Houses room carries one quiet line, a state and not a
count — "this village has outgrown the free licence" — so the steward reads
it before the day a family arrives, not on it.

### What the product has to lose before it is not this village

From `grep` on 2026-09-23. Each becomes config, a default, or a neutral name:

| Where | What | Becomes |
| --- | --- | --- |
| `config.go` | ~~defaults naming the village's domain and town~~ | **Done 2026-09-23** (`f747351` on `main`): `VILLAGE_NAME`, `PUBLIC_URL`, `PUSH_SUBJECT` required, no defaults; `WEATHER_LOCATION` unset = panel off; `PAGI_BOOTSTRAP_CODE` |
| `go.mod` | ~~the module path carried the village~~ | **Done 2026-09-23** (`1fbbe4d`): `github.com/dennislapchenko/porta-pagi/backend` |
| `weather.go` | ~~ARSO the only provider~~ | **Done 2026-09-23** (`eaad841`, `793d191`): `WEATHER_PROVIDER` is `open-meteo` (any `WEATHER_COORDS`, the default) or `arso` (this village); one panel shape with neutral icon keys, the page knows no provider |
| `mcp.go` | the `potok_` key prefix (`serverInfo.name` is `porta-pagi` since `f747351`) | `pagi_` — only keys minted after change; old ones keep working, the hash is what is checked |
| `store.go` | `potok.db`, `potok-YYYY-MM-DD.db` backups | `pagi.db`, `pagi-YYYY-MM-DD.db`. Renaming this village's live DB file is one step in `deploy/infra-log.md` — or the filename stays configurable and this village keeps its name |
| `server.go` | the export's `potok-export-*.json` filename | Same |
| `Taskfile.yml` | the `vm:*` tasks name the village's host and container | Stayed in the village's repo, which is theirs; the product's Taskfile has none |
| `index.html`, `App.tsx`, `manifest.webmanifest`, `backdrop.jpg` | ~~the village's name, language and picture in the shell~~ | **Done 2026-09-23**: `static.go` fills `{{VILLAGE_NAME}}`, `{{LANGUAGES}}` and `{{LANG}}` into the shell and the manifest at serve time; the manifest's description is neutral. The backdrop is every village's default until a village imports its own (`site_files`, `e0ea25d`). The castle icon stays for now (owner, 2026-09-23) |
| `i18n.tsx`, `when.go`, `push.ts` | ~~`sl` or `en` on both sides, Slovenian first~~ | **Done 2026-09-23** (`474a00e`, `fd32af8`): `LANGUAGES` env lists a village's languages, first is default; one dictionary per language in `backend/internal/httpapi/i18n/`, embedded by Go and imported by Vite, keyed by the English phrase; a test keeps every entry in its key's shape. Portuguese is one file, `pt.json`. `Weather.tsx` still parses ARSO's icon names — row 7 |
| `019_codex.sql`, `003_push_lang.sql` | ~~the codex bilingual by column, push language defaulting to `sl`~~ | **Done 2026-09-23** (`e6d74d8`, `097f018`): `codex_texts`, one row per language, migrated on the village's database; a phone subscribes in a language the village speaks or gets its first |
| `localStorage` keys | `potok.*` | Stay — renaming them logs every phone out |
| `frontend/public/data/` | ~~one village's land in the build~~ | **Done 2026-09-23** (`79e71b4`): two rows in `map_files`, in the database like the photos — the owner refused a file on the server as neither portable nor backed up. Imported once with `/server map-import`, served to a token with an ETag, in the export as text; absent parcels = the empty state, absent water = nothing drawn. This village's files are its seed in `deploy/map/`. The map's fixed centre went with it |
| `contacts.go`, `Contacts.tsx` | Slovenian diacritics folded for type matching | Stays — folding is right in every language |
| `README.md`, `CLAUDE.md`, `village.md`, `deploy/` | The village throughout | `CLAUDE.md` and the design docs go with the product and lose the village's specifics; `village.md`, `deploy/` and `infra-log.md` stay here |
