-- Market: buy for me. A house asks the village to buy a thing for it, mostly
-- online: a link, a few words, or both, and the most it would pay as it writes
-- it. Any other house marks it bought. Whether it was paid back is the
-- buyer's own note: repaid_at is read by the buyer only (buys.go).
CREATE TABLE buys (
  id          INTEGER PRIMARY KEY,
  -- SET NULL: a house that leaves owing for a thing does not wipe the buyer's
  -- note. Its open asks go with it (the trigger below).
  house_id    INTEGER REFERENCES houses(id) ON DELETE SET NULL,
  text        TEXT NOT NULL DEFAULT '',
  link        TEXT NOT NULL DEFAULT '',
  max_price   TEXT NOT NULL DEFAULT '',
  -- bought_at set means bought; bought_by NULL with it means the buyer left.
  bought_by   INTEGER REFERENCES houses(id) ON DELETE SET NULL,
  bought_at   TEXT,
  repaid_at   TEXT,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TRIGGER buys_open_leave_with_house BEFORE DELETE ON houses
BEGIN
  DELETE FROM buys WHERE house_id = OLD.id AND bought_at IS NULL;
END;
