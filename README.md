# Porta Pagi

A village on one page. Private, quiet, and yours to keep.

A map of your hamlet where the things that must stay findable live: who is
away, who is going to the shop, when the work party is, who has the ladder.
Not a group chat. Not a social network. No ads, no trackers, nothing sold
inside.

Closed web portal for the houses of one small community: a cadastral map as
the home screen, and seven rooms — Tavern (pinned notices, the month calendar
with work-bee sign-ups, and the message board), Market (needs, give-aways,
shop runs), Watchtower (who is away), Tool shed (what the village lends),
Projects (long jobs split into takable tasks), Campground (who collected from
which camper) and Contacts (who to call) — plus the Codex, the village's
founding text, editable by every house. One shared login per house, handed
out as an invite link; a house adds its own further phones with a six-digit
pairing code. Installs to the home screen and sends notifications.

Built by and for one collective in the Slovenian karst; the reasons behind
every room are in `CLAUDE.md`, and the village that built it runs it from its
own deploy repo.

- **Frontend:** Vite + React 18 + TypeScript, hash-routed SPA, one dictionary
  per language, installable (web manifest + service worker for push). The
  changelog at the foot of Home is the last 90 days of the git log, filtered
  and written to `public/changelog.json` by `scripts/changelog.mjs` before
  every build. Built **into the backend image** (`backend/Dockerfile`, build
  context is the repo root) and served by the Go binary, so the portal is one
  origin on its own domain.
- **Backend:** Go, stdlib HTTP, SQLite (pure Go), VAPID web push, one binary,
  nightly backups in-process. Image built by CI to
  `ghcr.io/dennislapchenko/porta-pagi`, tagged `sha-<commit>` and `latest`.
- **Map data:** two rows in the database (`map_files`), entered once from a
  file — `/server map-import parcels <source> <YYYY-MM-DD>` and `/server
  map-import water` — so the SQLite backup is the whole village and one image
  serves any village. Parcels must be a GeoJSON FeatureCollection in planar
  metres, under 2 MB; a village without the row sees what it needs instead of
  a map. House ↔ parcel assignment is app data, never in git.

```sh
task check      # vet + test backend, typecheck + build frontend
task be:run     # backend on 127.0.0.1:8788 (bootstrap code printed in the log); BIND unset = all interfaces, as in the container
# CSP check against a real page: build, copy dist into the embed dir, run the
# backend, open :8788 — then `git checkout backend/internal/httpapi/web/` to get the placeholders back.
task fe:build && cp -R frontend/dist/. backend/internal/httpapi/web/
task fe:dev     # frontend on :5173 against the local backend
```

## Running a village

One container, one volume. Pull `ghcr.io/dennislapchenko/porta-pagi:<tag>`,
mount a data directory at `/data`, put a TLS proxy in front, and set the
environment below. The first steward's bootstrap code is printed to the
container log on first boot; `docker exec <container> /server code "<house>"`
prints a fresh invite link for a house when nobody is logged in.

The binary carries no village. Four variables name one, have no default, and
the process refuses to start without them: `VILLAGE_NAME` (the tab title, the
home-screen label, the heading), `LANGUAGES` (the village's languages in its
order, first is the default — `sl,en`; each needs a dictionary in
`backend/internal/httpapi/i18n/`, English needs none), `PUBLIC_URL` (invite
links are printed under it; it is the contact in the User-Agent sent to the
weather provider), `PUSH_SUBJECT` (the VAPID subject push services see).
Optional: the weather — `WEATHER_PROVIDER` is `open-meteo` (the default:
anywhere, from `WEATHER_COORDS=lat,lon` — the pair any map gives,
OpenStreetMap's URL carries it — with `WEATHER_LOCATION` as the place name the
panel prints) or `arso` (Slovenia, `WEATHER_LOCATION` is the ARSO place name),
and the panel is off while either is unset — `PAGI_BOOTSTRAP_CODE` (a fixed
first-steward code; unset means generated and logged), `DATA_DIR`, `PORT`,
`BIND`, `TZ`. The picture behind the gate is the built-in `backdrop.jpg` until
a village imports its own (`/server site-import backdrop < picture.jpg`, a row
in `site_files`). The founding text enters once with `/server codex-import <
codex.json` (shape in `backend/internal/httpapi/codex.go`).

## The demo village

`/server demo-seed` fills an **empty** database with an invented collective —
*Vale do Sobreiro Torto*, ten houses, a year of their life in Portuguese and
English — dated off the day it runs. Run it with `LANGUAGES=pt,en` and
`VILLAGE_NAME="Sobreiro Torto"`; then `/server code "Casa da Eira"` prints
the steward house's invite link. Nothing in it is real: the land is a
synthetic mesh, the numbers are `000` patterns.

## For agents (MCP)

The portal's API as MCP tools: an agent acts **as your house** — same rows,
same pushes, same rules as a phone. For any villager who runs one.

Get a key: Houses room → Your house → `+ MCP key`. It is shown once; the
portal keeps only its fingerprint. Then:

```sh
claude mcp add --transport http potok https://<the portal's domain>/api/mcp \
  --header "Authorization: Bearer potok_…"
```

Other clients, `.mcp.json` shape:

```json
{"mcpServers":{"potok":{"type":"http","url":"https://<the portal's domain>/api/mcp",
  "headers":{"Authorization":"Bearer potok_…"}}}}
```

What the agent can do: read and write in every room — board, calendar and
answers, threads, shop runs and needs, give-aways, the Watchtower, the shed
and its wishlist, projects, tasks and a picture on a project, the campground,
the codex, the weather. What it cannot: delete anything, mint keys or touch
devices, edit the house itself, or do a steward's work. **The key opens
`/api/mcp` and nothing else** — `curl` with it gets a 403 — so the tool list
is the whole surface.

Two clocks: send local wall clock `YYYY-MM-DDTHH:MM` (a date alone is
`YYYY-MM-DD`); what you read as `created_at` is UTC with a space. Leave
`author` empty unless the person dictated the words — a post with no name
reads as the house, which is true. claude.ai's web connectors need OAuth and
will not connect. Claude Code takes the header as above; another client whose
config takes a header should work the same, but none was tried.

Revoke: Your devices → Remove on the 🤖 row. The key is your house's login in
plain text in a config file on a laptop — treat it like the invite link.
Dev: `http://127.0.0.1:8788/api/mcp` against `task be:run`.

## Licence

Business Source License 1.1 — see `LICENSE`. Free to self-host for one
community of up to ten houses. Above that, or to host it for others for money
in any form, a commercial licence. Each version becomes AGPL four years after
release. Read `CLAUDE.md` before changing anything; pull requests are closed
for now (`CONTRIBUTING.md`).
