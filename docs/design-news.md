# Portal News: a curated note that rings when a village gets it

Built 2026-10-08 on the owner's ask (`news.go`, `news/news.json`,
`news_test.go`, the Changelog panel's first tab); the owner's decisions are
at the end.

The owner's words: "keeping a user-facing notification-enabled changelog
file somewhere, and if it updates (maybe as a database migration?) then a
notification to all users would be sent … users won't need all of the
details … when shipping new features or fixing user-curious bugs we would
update this, possibly with a title for notif and small body. Then upon
villages rolling out new images … this notif would fire. And of course each
house (and globally for steward) it could be on/offed."

**The villager job, in one line: "what is new in our portal, and is the
thing I asked for in?" — told once, in the village's language, without the
builder's words for builders.** The git log alone answered it before:
English commit subjects, migration and backup notes included (the
calendar's subject ends "migration 029 rebuilds event answers, rehearse on
backup"), and nothing rang. The git log stays, as the panel's second tab.

## How it works

- **The notes are a file in the repo**, embedded in the binary like the
  dictionaries: `backend/internal/httpapi/news/news.json`, newest first. One
  entry is an id (the day and a slug, `2026-10-07-calendar`), its day, and
  per language a title for the lock screen and a body of at most 120
  characters, the push's whole snippet (`TestNewsFile`):

  ```json
  {"id": "2026-10-07-calendar", "on": "2026-10-07",
   "texts": {"en": {"title": "The calendar knows the sky", "body": "Moons, seasons and the eclipses we can see are in the calendar, with birthdays and events that repeat."},
             "sl": {"title": "…", "body": "…"}, "pt": {"title": "…", "body": "…"}}}
  ```

  A language the village speaks and the entry lacks falls back to the
  village's first language that has it, as the codex does.
- **The binary rings what the village has not heard yet.** On boot and in
  the 30-minute round that already writes the sky and rings birthdays, the
  backend compares the embedded ids with the ones it marked as told
  (`settings`, `news:<id>`), pushes the untold ones under a new push kind,
  `news`, and marks them. Quiet hours hold it until 07:00, as birthdays.
  **A roll is what makes it ring**: the village gets the note exactly when it
  gets the image, and never twice, because the mark lives in its database.
- **A fresh village rings nothing for the past**: on the first boot with no
  houses every id present is marked told without a push. A village that
  rolls over several notes at once gets **one** push — "Portal News" and
  their titles — and the panel lists all of them.
- **On or off per house, and village-wide by a steward**: `news` is an
  ordinary kind, so the house's own list and the steward's mute on the
  settings page already cover it. On by default, like every kind.
- **The panel shows the notes**, in the reader's language, newest first,
  from `GET /api/news` (behind the token, like everything). What the
  village has heard is what the binary carries, so the panel and the push
  never disagree.
- **Who writes a note**: the assistant, in the same commit as the
  user-visible change it describes — a feature, or a bug a villager would
  have noticed — in the village's languages, and the owner reads it in the
  diff. A change nobody sees gets no note; `CLAUDE.md` § Working rules says
  so.

## Options considered

| How the note travels | For | Against |
| --- | --- | --- |
| **A file embedded in the binary** (built) | Rings exactly when the image arrives and once, by a mark in the village's own database; a typo is fixed by editing the file until the image is rolled; translations live beside the text; the demo, wiped hourly, still shows the notes because they are in the binary | A note shipped and rolled cannot be unsaid on a village's phones — true of every push |
| A migration that inserts the note | The owner's first idea: it runs once per database by construction | Migrations are append-only, so a typo is there forever; content mixed into schema history; texts in three languages inside SQL; the demo wipes its tables hourly but not the migration ledger, so its notes would vanish after the first hour |
| Notes read off commit messages (a "news:" line) | Nothing new to remember | English only, in the builder's register; a commit message cannot be fixed after the push; ties the backend to the frontend's build-time git step |
| **No push at all**: a dot on the 🔨 chip until the note is read, or a pinned line on the board in the village's words with its thread | Reaches nobody's bedside table; a board line can be answered ("finally! thank you") where a lock screen cannot | The owner asked for a push; a dot is only seen by whoever scrolls to the foot of Home |

**How often is too often.** An active fortnight is three or four rolls. The
only brakes on the builder's voice reaching every phone are "one push per
roll" and "a change nobody sees gets no note", plus each house's opt-out and
the steward's village-wide mute. **On the lock screen** a note names no
house, no person and no absence, so its title may travel as it is
(`CLAUDE.md` § What a lock screen may say).

## What it touches

`news.json` and `news.go` (the list route, the round that rings, the
empty-village mark), the kind `news` in `Kinds` and its label in the
Notifications panel, the Changelog panel's first tab, `news_test.go` for
the file's rules and the push's lines, `CLAUDE.md` (an invariant and the
writing rule), `README.md`. No migration and no new table.

## Decided (owner, 2026-10-08)

1. **The changelog stays and gets two tabs**: Portal News, chosen by
   default, and Development, the git log as before.
2. **One push per roll**: one note by its own title and body; several as
   "Portal News" with their titles.
3. **The kind is called "Portal News".**
4. **A push, not the quiet way** (a dot on the 🔨 chip, or a pinned board
   line): "if 2 new features roll per week — notif on both makes sense".
5. **The calendar is the first note**, and it rang Mokri Potok with the
   weather's fifteen days in one push when the village rolled `9f56321`
   (2026-10-08).
