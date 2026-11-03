-- A PIN is chosen by whoever opens a house's invite link on a device, and
-- with the house's name it signs that person in on any other browser later
-- (auth.go § PINs). It is kept on the device row, hashed with the house id;
-- a sign-in by PIN copies it onto the new device row, so a person's PIN
-- outlives the browser it was chosen in. Empty is a device that has none:
-- every device from before the PIN, every agent key.
ALTER TABLE devices ADD COLUMN pin_hash TEXT;

-- Wrong PINs per house since its last right one; at three the house is
-- locked until a join by its link clears the row, as a right PIN before the
-- lock does. Its own table so the export, which dumps `houses` whole, never
-- carries it.
CREATE TABLE pin_tries (
  house_id    INTEGER PRIMARY KEY REFERENCES houses(id) ON DELETE CASCADE,
  misses      INTEGER NOT NULL DEFAULT 0
);

-- `pairings` stays, unused: the six-digit pairing code is gone with the PIN,
-- and an image rolled back to before this file still queries the table.
