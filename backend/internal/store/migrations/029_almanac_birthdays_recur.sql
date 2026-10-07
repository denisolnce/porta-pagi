-- An event may come back: weekly, monthly or yearly. Empty is a one-off, which
-- is every row so far. A birthday is a yearly event whose starts_at is the
-- birth itself — the date, and the hour when somebody knows it.
ALTER TABLE events ADD COLUMN recur TEXT NOT NULL DEFAULT '';

-- An answer belongs to one day. A weekly work party has a headcount per
-- Saturday, so the key grows the date the answer is for; a one-off event
-- keeps '' there, which is every answer given so far. SQLite cannot widen a
-- primary key in place, so the table is rebuilt; nothing references it.
CREATE TABLE event_signups_new (
  event_id          INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  house_id          INTEGER NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
  occurs_on         TEXT NOT NULL DEFAULT '',
  created_at        TEXT NOT NULL DEFAULT (datetime('now')),
  state             TEXT NOT NULL DEFAULT 'yes',
  answered_at       TEXT,
  answered_version  INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (event_id, house_id, occurs_on)
);
INSERT INTO event_signups_new (event_id, house_id, created_at, state, answered_at, answered_version)
SELECT event_id, house_id, created_at, state, answered_at, answered_version FROM event_signups;
DROP TABLE event_signups;
ALTER TABLE event_signups_new RENAME TO event_signups;

-- The sky over the village, written by the backend once a year: new and full
-- moons, equinoxes, solstices, and the eclipses the village can see. Nobody
-- creates, edits or answers one; a house may only talk about it. Times are
-- the village's wall clock, the same shape as an event's, so the calendar
-- reads both alike. detail is JSON the page words in the reader's language.
CREATE TABLE almanac (
  id         INTEGER PRIMARY KEY,
  kind       TEXT NOT NULL,
  starts_at  TEXT NOT NULL,
  ends_at    TEXT,
  detail     TEXT NOT NULL DEFAULT '{}',
  UNIQUE (kind, starts_at)
);
