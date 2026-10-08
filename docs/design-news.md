# What's new: a curated note that rings when a village gets it

Researched 2026-10-08 on the owner's ask; **nothing built**. The open
choices are at the end and are the owner's.

The owner's words: "keeping a user-facing notification-enabled changelog
file somewhere, and if it updates (maybe as a database migration?) then a
notification to all users would be sent … users won't need all of the
details … when shipping new features or fixing user-curious bugs we would
update this, possibly with a title for notif and small body. Then upon
villages rolling out new images … this notif would fire. And of course each
house (and globally for steward) it could be on/offed."

**The villager job, in one line: "what is new in our portal, and is the
thing I asked for in?" — told once, in the village's language, without the
builder's words for builders.** Today the 🔨 Changelog panel answers it from
the git log: English commit subjects, migration and backup notes included
(the calendar's subject ends "migration 029 rebuilds event answers, rehearse
on backup"), and nothing rings.

## How it would work

- **The notes are a file in the repo**, embedded in the binary like the
  dictionaries: `backend/internal/httpapi/news/news.json`, newest first. One
  entry is an id (the day and a slug, `2026-10-07-calendar`), its day, and
  per language a title for the lock screen and one or two lines of body:

  ```json
  {"id": "2026-10-07-calendar", "on": "2026-10-07",
   "texts": {"en": {"title": "The calendar knows the sky", "body": "Full moons, the seasons' turns and the eclipses we can see. Birthdays, and events that repeat every week, month or year."},
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
  rolls over several notes at once gets **one** push — the newest title,
  "and 2 more" — and the panel lists all of them.
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
  diff. A change nobody sees gets no note. `CLAUDE.md` would say so.

## Options considered

| How the note travels | For | Against |
| --- | --- | --- |
| **A file embedded in the binary** (recommended) | Rings exactly when the image arrives and once, by a mark in the village's own database; a typo is fixed by editing the file until the image is rolled; translations live beside the text; the demo, wiped hourly, still shows the notes because they are in the binary | A note shipped and rolled cannot be unsaid on a village's phones — true of every push |
| A migration that inserts the note | The owner's first idea: it runs once per database by construction | Migrations are append-only, so a typo is there forever; content mixed into schema history; texts in three languages inside SQL; the demo wipes its tables hourly but not the migration ledger, so its notes would vanish after the first hour |
| Notes read off commit messages (a "news:" line) | Nothing new to remember | English only, in the builder's register; a commit message cannot be fixed after the push; ties the backend to the frontend's build-time git step |

## What it touches

`news.json` and a small `news.go` (the list route, the round that rings,
the fresh-village mark), the kind `news` in `Kinds` and its label in the
Notifications panel, the Changelog panel's data source, `banner_test.go`
for the push's two lines, `CLAUDE.md` (an invariant and the writing rule),
`README.md`. No migration and no new table. About a day's work.

## Open choices for the owner

1. **The git-log changelog**: replace it with the notes (one panel, one
   voice), or keep it under the notes for the curious, or drop the panel's
   git half from Home and keep it in the repo only.
2. **One push per roll** naming the newest note and how many more
   (recommended), or one push per note.
3. **The kind's name in the list of notifications**: "What's new" is the
   suggestion.
4. **The first note**: the calendar (sky, birthdays, repeats) is already on
   both villages, so its note would ring nobody under the "fresh village"
   rule only for a new village; for Mokri Potok and the demo it would ring
   on the next roll. Ship it as the first note, or start with the next
   change.
