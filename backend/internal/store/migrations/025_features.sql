-- Switches: features a village turns on or off, each with its own default
-- (docs/commercial.md, the Decided block: "a list of switches"). A row exists
-- only for a switch a steward has flipped; no row means the default in code
-- (features.go). Who flipped it and when stays with the row, the same rule
-- as a village-wide mute: silent power is the problem, not the lever.
CREATE TABLE features (
  key     TEXT PRIMARY KEY,
  enabled INTEGER NOT NULL,
  set_by  INTEGER REFERENCES houses(id) ON DELETE SET NULL,
  set_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
-- The campground became a switch, off for a new village. A village that
-- existed before switches keeps every room it had.
INSERT INTO features(key, enabled) SELECT 'camp', 1 WHERE EXISTS (SELECT 1 FROM houses);
