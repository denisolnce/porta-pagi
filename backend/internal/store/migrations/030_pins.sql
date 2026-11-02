-- A PIN is chosen by whoever opens a house's invite link on a device, and
-- with the house's name it signs that person in on any other browser later
-- (auth.go § PINs). It is kept on the device row, hashed with the house id;
-- a sign-in by PIN copies it onto the new device row, so a person's PIN
-- outlives the browser it was chosen in. Empty is a device that has none:
-- every device from before the PIN, every agent key.
ALTER TABLE devices ADD COLUMN pin_hash TEXT;

-- Wrong PINs per house since its last right one, and the moment the next
-- try is taken again. A right PIN or a join by the house's link clears the
-- row. Its own table so the export, which dumps `houses` whole, never
-- carries it.
CREATE TABLE pin_tries (
  house_id    INTEGER PRIMARY KEY REFERENCES houses(id) ON DELETE CASCADE,
  misses      INTEGER NOT NULL DEFAULT 0,
  wait_until  TEXT
);

-- `pairings` stays, unused: the six-digit pairing code is gone with the PIN,
-- and an image rolled back to before this file still queries the table.
