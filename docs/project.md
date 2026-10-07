# The project around this repo

What a session working from `porta-pagi` needs to know about the other repos,
the VM, and how things ship. Pointers, not values: each fact has one owner,
named here. Written 2026-10-07 as the handoff when the owner moved all work on
this project into sessions started from this repo.

This repo will be public. Nothing private goes in here: the business plan,
prices' reasoning and the owner's open commercial decisions live in
`porta-pagi-cloud` (private); people and households of the village live
nowhere in git.

## The four repos

All under `~/dev/dsolnce/`, all on GitHub as `denisolnce/<name>` (the account
was `dennislapchenko` until 2026-10; the old name is gone from config).

| Repo | Visibility | What it holds | Ships as |
| --- | --- | --- | --- |
| `porta-pagi` (this) | private, going public | The product: Go backend, React frontend, migrations, the demo seed, `CLAUDE.md` invariants, design docs, `docs/later.md` (backlog and open product decisions), `LICENSE` (BSL 1.1, ten-house grant) | CI on push to `main` builds `ghcr.io/denisolnce/porta-pagi:sha-<commit>` and `porta-pagi-demo:sha-<commit>` (the demo build tag). Both packages public |
| `porta-pagi-cloud` | private, stays so | The landing page (`site/`, one HTML file, no scripts), `docs/commercial.md` (the business plan: Decided block, §5 price table, §11 ordered backlog, §12 open questions in unblocking order), later the control plane | CI builds `ghcr.io/denisolnce/porta-pagi-site:sha-<commit>` (public package) |
| `mokri-potok-portal` | public | The deploy repo of the village that built the portal: its compose, env, `BE_TAG`, map seed, `infra-log.md`, `.claude/context/village.md` (who uses it, which switches are on) | Nothing; `task roll` writes `BE_TAG` and the VM reconciles |
| `gaias-choice` | public | Another project whose VM hosts everything. Owns Caddy (`deploy/app/Caddyfile`), the doco-cd controller and its secrets (`deploy/controller/`, `task doco:secrets`), and runs the Porta Pagi **demo** and **landing** as services in its stack (`pagi-demo`, `pagi-site`, on their own compose network `pagi`) | Tags in its `.doco-cd.yml`: `PAGI_DEMO_TAG`, `PAGI_SITE_TAG` |

## What runs where

One VM (Hetzner, Helsinki): `ssh -p 13337 -i ~/.ssh/gaia
root@gaias-choice.gardenofatlantis.com`. doco-cd polls both deploy repos and
reconciles within about two minutes of a push.

- **The village** — `https://vas.mokri-potok.si`, container
  `mokri-potok-potok-api-1`, data `/srv/mokri-potok/data`. Tag:
  `mokri-potok-portal/.doco-cd.yml` `BE_TAG`.
- **The demo** — `https://pp-demo.gardenofatlantis.com`, container
  `gaias-choice-pagi-demo-1`: the demo build on a 256 MB tmpfs, seeds itself on
  boot, wipes and reseeds every hour, memory/CPU/pid caps, refuses switch flips
  and picture uploads (`demoLocked`).
- **The landing** — `https://porta-pagi.gardenofatlantis.com` (a test
  address; `noindex` until the real domain and a public source repo),
  container `gaias-choice-pagi-site-1`.

## How to ship

1. **Change the product** here; run `task check` and judge it **by exit code**
   (`task check; echo $?`) — a failed `go vet` prints no FAIL line, and a grep
   once let a red main through. Push to `main`; wait for CI
   (`gh run list -R denisolnce/porta-pagi --limit 1`).
2. **The village:** in `mokri-potok-portal`, `task roll -- sha-<full commit>`.
   A migration touching a table the village has rows in is rehearsed first on
   a copy of the latest backup (`task vm:backup`, run the new binary's
   migrations on the copy, delete the copy). Verify on the VM:
   `docker inspect` the image and health, `task vm:logs` for `migrated` lines.
3. **The demo:** bump `PAGI_DEMO_TAG` in `gaias-choice/.doco-cd.yml`, commit,
   push, verify the same way.
4. **The landing:** change `porta-pagi-cloud/site/`, wait for its CI, bump
   `PAGI_SITE_TAG` in gaias-choice. Its pictures are retaken from the live
   demo — the recipe is in `porta-pagi-cloud/CLAUDE.md`.

A private GHCR package leaves the VM on its old container with doco-cd failing
quietly: a new package must be made public before a tag naming it is pushed.

## Where the state lives

- **What each place runs:** the tags above.
- **Which rooms the village has switched on:** `mokri-potok-portal/.claude/context/village.md`
  (today: campground on, dues and roster off).
- **What waits on the owner, in order:** `porta-pagi-cloud/docs/commercial.md`
  §12, head of the section. As of 2026-10-07: how this repo goes public (a
  fresh repo whose history starts on the public day is recommended — its
  history still holds every version of the business plan) together with two
  rows in `docs/later.md` about the collective's own business; then the
  landing's real domain and footer mailbox; then thirty outreach emails,
  drafted by the assistant, sent by the owner. Not blocking those: feedback
  email (steps in `gaias-choice/deploy/README.md`), who else answers support
  email, and the product decisions in `docs/later.md` § Open decisions.
- **What the assistant owes next:** the licence key and the cap in code
  (`porta-pagi-cloud/docs/commercial.md` §11 row 11b, about a day), built
  before this repo goes public so it ships in the first public commit. The
  repo's latest commit, the module rename to `denisolnce`, is built by CI and
  not yet rolled anywhere.

## Working with the owner

- Every turn that changes files ends with the `reviewer` subagent
  (`.claude/agents/reviewer.md`); act on what it finds or surface it.
- Anything the owner is told to do, or how to do, goes into the doc that owns
  it before the turn ends — chat is never the only copy.
- A path or a command handed to the owner is one confirmed in the same turn
  (`ls`, or the command's dry form).
- Commits: lowercase, succinct, no backticks in subjects, no AI trailers.
- Parallel agents work best in git worktrees of this repo, one branch each,
  merged by the session. The i18n dictionaries conflict on every merge: merge
  them three-way by key, not by line.
- Playwright on this Mac (macOS 13) needs `playwright@1.52`; newer ones have
  no browser build for it.
