-- Dues: who paid the water this quarter. Behind the `dues` switch, off by
-- default (features.go); the tables exist in every village and stay empty in
-- one that never switched it on.
--
-- A due is what the village asks of its houses: a title, an optional amount
-- per house with the currency as the village writes it, an optional date.
-- A steward writes it down and names the house that collects it. The portal
-- records who paid; it never moves money.
--
-- The amount is minor units (cents) and not free text, because a ledger is
-- allowed a total here — collected against expected, per due — and a total
-- needs a number. The currency is text: "€", "EUR", whatever the village says.
CREATE TABLE dues (
  id            INTEGER PRIMARY KEY,
  title         TEXT NOT NULL,
  amount_cents  INTEGER,
  currency      TEXT NOT NULL DEFAULT '',
  due_on        TEXT,
  notes         TEXT NOT NULL DEFAULT '',
  collector_id  INTEGER REFERENCES houses(id) ON DELETE SET NULL,
  house_id      INTEGER REFERENCES houses(id) ON DELETE SET NULL,
  created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

-- One row per house a due asks, written when the due is. paid_at NULL is not
-- paid. paid_by is the house that marked it — the collector or a steward.
--
-- A house that is deleted takes its unpaid rows with it (the trigger) and
-- leaves its paid ones behind without a name (SET NULL): the ledger keeps
-- what came in, and forgets what a departed house still owed. Settle with a
-- house before its stay ends.
CREATE TABLE dues_owed (
  id        INTEGER PRIMARY KEY,
  due_id    INTEGER NOT NULL REFERENCES dues(id) ON DELETE CASCADE,
  house_id  INTEGER REFERENCES houses(id) ON DELETE SET NULL,
  paid_at   TEXT,
  paid_by   INTEGER REFERENCES houses(id) ON DELETE SET NULL,
  UNIQUE (due_id, house_id)
);
CREATE INDEX idx_dues_owed_house ON dues_owed(house_id);

CREATE TRIGGER dues_owed_unpaid_leave BEFORE DELETE ON houses BEGIN
  DELETE FROM dues_owed WHERE house_id = OLD.id AND paid_at IS NULL;
END;
