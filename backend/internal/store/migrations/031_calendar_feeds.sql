-- A house's calendar subscription: the secret in the link a phone's or
-- Google's calendar re-reads (ical.go). One per house; a new one replaces
-- it, and the old link stops. Kept plain like an invite, because the house
-- reads its link again whenever a member adds the calendar; not in the
-- export, which carries no credential.
CREATE TABLE calendar_feeds (
  house_id    INTEGER PRIMARY KEY REFERENCES houses(id) ON DELETE CASCADE,
  token       TEXT NOT NULL UNIQUE,
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
