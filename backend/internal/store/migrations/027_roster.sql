-- Duty roster: the village's recurring chores and the turns houses took on
-- them. Behind the `roster` switch (features.go), off by default: this and
-- the dues ledger are the two places the portal counts, and only because a
-- village switched it on. Design: docs/design-more-rooms.md § Duty roster.
--
-- A duty is an agreement the village made, written down by a steward. It
-- outlives the steward's house (SET NULL). `every_days` is the cadence the
-- village agreed — 14 for "every two weeks" — and NULL means "when needed",
-- which is what snow is. A duty somebody has done is retired, never deleted:
-- its turns are history and the count read them.
CREATE TABLE roster_duties (
  id          INTEGER PRIMARY KEY,
  house_id    INTEGER REFERENCES houses(id) ON DELETE SET NULL,
  title       TEXT NOT NULL,
  notes       TEXT NOT NULL DEFAULT '',
  every_days  INTEGER,
  retired_at  TEXT,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- A turn is one house doing one duty on one date. `planned` until that house
-- says it is `done`; only a done turn counts. A done turn outlives the house
-- that did it (SET NULL): the lane was still cleared that day, and "next due"
-- reads off the last done turn whoever did it — a cascade would make a duty
-- look overdue the day a house leaves. A planned turn of a house that is gone
-- is nobody's plan, and the room leaves it out.
CREATE TABLE roster_turns (
  id          INTEGER PRIMARY KEY,
  duty_id     INTEGER NOT NULL REFERENCES roster_duties(id) ON DELETE CASCADE,
  house_id    INTEGER REFERENCES houses(id) ON DELETE SET NULL,
  on_date     TEXT NOT NULL,
  state       TEXT NOT NULL DEFAULT 'planned',
  note        TEXT NOT NULL DEFAULT '',
  done_at     TEXT,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
-- One house, one duty, one day: a double tap must not count twice.
CREATE UNIQUE INDEX idx_roster_turns_one ON roster_turns(duty_id, house_id, on_date);
